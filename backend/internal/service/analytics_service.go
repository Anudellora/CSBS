package service

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"csbs/backend/internal/models"
	"csbs/backend/internal/repository"
	"csbs/backend/pkg/logger"
)

// AnalyticsService собирает данные для админ-дашборда: фактическую загрузку
// по дням, прогноз ML на неделю, распределение по локациям/категориям и
// текстовую рекомендацию от LLM на сегодня.
type AnalyticsService interface {
	GetDashboard() (*AnalyticsDashboard, error)
}

type analyticsServiceImpl struct {
	reservationRepo repository.ReservationRepository
	workspaceRepo   repository.WorkspaceRepository
	prediction      PredictionService
}

func NewAnalyticsService(
	rRepo repository.ReservationRepository,
	wRepo repository.WorkspaceRepository,
	prediction PredictionService,
) AnalyticsService {
	return &analyticsServiceImpl{
		reservationRepo: rRepo,
		workspaceRepo:   wRepo,
		prediction:      prediction,
	}
}

// --- DTO ---

type AnalyticsDashboard struct {
	Today        TodayStats      `json:"today"`
	MLWeek       []DayLoad       `json:"ml_week"`
	ActualLast14 []DayLoad       `json:"actual_last_14"`
	ByCategory   []CategoryCount `json:"by_category"`
	ByLocation   []LocationCount `json:"by_location"`
	LLMToday     *LLMInsight     `json:"llm_today,omitempty"`
}

type TodayStats struct {
	LoadPercent      float64 `json:"load_percent"`
	BookingsCount    int     `json:"bookings_count"`
	WorkspacesTotal  int     `json:"workspaces_total"`
	WorkspacesActive int     `json:"workspaces_active"`
}

type DayLoad struct {
	Label       string  `json:"label"`
	Date        string  `json:"date,omitempty"`
	LoadPercent float64 `json:"load_percent"`
}

type CategoryCount struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

type LocationCount struct {
	Location string `json:"location"`
	Count    int    `json:"count"`
}

type LLMInsight struct {
	Day                 string  `json:"day"`
	ExpectedWorkload    float64 `json:"expected_workload_percent"`
	RecommendedPriceRub int     `json:"recommended_price_rub"`
	Message             string  `json:"message"`
}

// --- Реализация ---

func (s *analyticsServiceImpl) GetDashboard() (*AnalyticsDashboard, error) {
	now := time.Now()
	loc := now.Location()

	workspaces, err := s.workspaceRepo.GetAll()
	if err != nil {
		return nil, err
	}
	totalWorkspaces := len(workspaces)

	// Окно для агрегатов: 30 дней назад → завтра (чтобы захватить «сегодня» целиком).
	startWindow := time.Date(now.Year(), now.Month(), now.Day()-29, 0, 0, 0, 0, loc)
	endWindow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)

	reservations, err := s.reservationRepo.GetReservationsBetween(startWindow, endWindow)
	if err != nil {
		return nil, err
	}

	dash := &AnalyticsDashboard{
		Today:        computeTodayStats(reservations, now, totalWorkspaces),
		MLWeek:       s.computeMLWeek(now),
		ActualLast14: computeActualLast14(reservations, now, totalWorkspaces),
		ByCategory:   computeByCategory(reservations),
		ByLocation:   computeByLocation(reservations),
	}

	if insight := s.fetchLLMInsight(now); insight != nil {
		dash.LLMToday = insight
	}

	return dash, nil
}

func computeTodayStats(reservations []models.Reservation, now time.Time, totalWorkspaces int) TodayStats {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	bookings := 0
	uniqueWs := map[uint]bool{}
	for _, r := range reservations {
		if r.StartTime.Before(dayStart) || !r.StartTime.Before(dayEnd) {
			continue
		}
		bookings++
		uniqueWs[r.WorkspaceID] = true
	}

	load := 0.0
	if totalWorkspaces > 0 {
		load = float64(len(uniqueWs)) / float64(totalWorkspaces) * 100
		if load > 100 {
			load = 100
		}
	}

	return TodayStats{
		LoadPercent:      roundTo(load, 1),
		BookingsCount:    bookings,
		WorkspacesTotal:  totalWorkspaces,
		WorkspacesActive: len(uniqueWs),
	}
}

func computeActualLast14(reservations []models.Reservation, now time.Time, totalWorkspaces int) []DayLoad {
	const days = 14
	result := make([]DayLoad, 0, days)
	loc := now.Location()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	// Бакеты: ключ — yyyy-mm-dd → множество уникальных workspace_id за день.
	buckets := map[string]map[uint]bool{}
	for i := days - 1; i >= 0; i-- {
		d := dayStart.AddDate(0, 0, -i)
		buckets[d.Format("2006-01-02")] = map[uint]bool{}
	}

	for _, r := range reservations {
		key := r.StartTime.In(loc).Format("2006-01-02")
		if bucket, ok := buckets[key]; ok {
			bucket[r.WorkspaceID] = true
		}
	}

	for i := days - 1; i >= 0; i-- {
		d := dayStart.AddDate(0, 0, -i)
		key := d.Format("2006-01-02")
		uniq := len(buckets[key])
		load := 0.0
		if totalWorkspaces > 0 {
			load = float64(uniq) / float64(totalWorkspaces) * 100
			if load > 100 {
				load = 100
			}
		}
		result = append(result, DayLoad{
			Label:       d.Format("02.01"),
			Date:        key,
			LoadPercent: roundTo(load, 1),
		})
	}
	return result
}

func (s *analyticsServiceImpl) computeMLWeek(now time.Time) []DayLoad {
	weekly := s.prediction.GetWeeklyWorkload()

	// Дни недели в правильном порядке + краткие лейблы.
	order := []struct{ ru, short string }{
		{"понедельник", "Пн"},
		{"вторник", "Вт"},
		{"среда", "Ср"},
		{"четверг", "Чт"},
		{"пятница", "Пт"},
		{"суббота", "Сб"},
		{"воскресенье", "Вс"},
	}

	result := make([]DayLoad, 0, len(order))
	for _, d := range order {
		val, ok := weekly[d.ru]
		if !ok {
			continue
		}
		result = append(result, DayLoad{
			Label:       d.short,
			LoadPercent: roundTo(val, 1),
		})
	}
	return result
}

func computeByCategory(reservations []models.Reservation) []CategoryCount {
	counts := map[string]int{}
	for _, r := range reservations {
		name := r.Workspace.Category.Name
		if name == "" {
			name = "Без категории"
		}
		counts[name]++
	}
	return mapToSortedSlice(counts, func(k string, v int) CategoryCount {
		return CategoryCount{Category: k, Count: v}
	}, func(a, b CategoryCount) bool { return a.Count > b.Count })
}

func computeByLocation(reservations []models.Reservation) []LocationCount {
	counts := map[string]int{}
	for _, r := range reservations {
		name := r.Workspace.Location.Name
		if name == "" {
			name = "Без локации"
		}
		counts[name]++
	}
	return mapToSortedSlice(counts, func(k string, v int) LocationCount {
		return LocationCount{Location: k, Count: v}
	}, func(a, b LocationCount) bool { return a.Count > b.Count })
}

func (s *analyticsServiceImpl) fetchLLMInsight(now time.Time) *LLMInsight {
	dayRu := russianWeekdayLower(now.Weekday())
	raw, err := s.prediction.GetWorkloadPrediction(dayRu)
	if err != nil {
		logger.Warn.Printf("Analytics: LLM insight недоступен: %v", err)
		return nil
	}
	cleaned := stripJSONFence(raw)
	var insight LLMInsight
	if err := json.Unmarshal([]byte(cleaned), &insight); err != nil {
		logger.Warn.Printf("Analytics: не удалось распарсить LLM-ответ: %v", err)
		return nil
	}
	return &insight
}

// --- Утилиты ---

func roundTo(v float64, digits int) float64 {
	shift := 1.0
	for i := 0; i < digits; i++ {
		shift *= 10
	}
	return float64(int(v*shift+0.5)) / shift
}

func mapToSortedSlice[K comparable, T any](
	m map[K]int,
	build func(K, int) T,
	less func(a, b T) bool,
) []T {
	out := make([]T, 0, len(m))
	for k, v := range m {
		out = append(out, build(k, v))
	}
	sort.Slice(out, func(i, j int) bool { return less(out[i], out[j]) })
	return out
}

func russianWeekdayLower(wd time.Weekday) string {
	switch wd {
	case time.Monday:
		return "понедельник"
	case time.Tuesday:
		return "вторник"
	case time.Wednesday:
		return "среда"
	case time.Thursday:
		return "четверг"
	case time.Friday:
		return "пятница"
	case time.Saturday:
		return "суббота"
	case time.Sunday:
		return "воскресенье"
	}
	return ""
}

// stripJSONFence убирает оборачивание ```json ... ``` если LLM такое прислал.
func stripJSONFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if nl := strings.IndexByte(s, '\n'); nl != -1 {
			s = s[nl+1:]
		}
		if i := strings.LastIndex(s, "```"); i != -1 {
			s = s[:i]
		}
	}
	return strings.TrimSpace(s)
}
