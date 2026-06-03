package service

import (
	"context"
	"time"

	"csbs/backend/internal/models"
	"csbs/backend/internal/repository"
	"csbs/backend/pkg/email"
	"csbs/backend/pkg/logger"
)

// ReminderService раз в interval опрашивает БД и шлёт пользователям письма
// с напоминанием о ближайшей броне — за 24 часа и за 3 часа до начала.
type ReminderService struct {
	repo        repository.ReservationRepository
	emailSender *email.Sender
	interval    time.Duration
}

func NewReminderService(repo repository.ReservationRepository, sender *email.Sender, interval time.Duration) *ReminderService {
	if interval <= 0 {
		interval = time.Minute
	}
	return &ReminderService{repo: repo, emailSender: sender, interval: interval}
}

// Start запускает цикл в отдельной горутине. Останавливается через ctx.
func (s *ReminderService) Start(ctx context.Context) {
	if s.emailSender == nil || !s.emailSender.Configured() {
		logger.Warn.Println("ReminderService: SMTP не настроен — напоминания о бронях отключены")
		return
	}

	go func() {
		// Первый проход сразу, чтобы не ждать тика после рестарта.
		s.tick()

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Info.Println("ReminderService: остановка")
				return
			case <-ticker.C:
				s.tick()
			}
		}
	}()

	logger.Info.Printf("ReminderService: запущен, период проверки %s", s.interval)
}

func (s *ReminderService) tick() {
	now := time.Now()
	s.process("24h", now)
	s.process("3h", now)
}

func (s *ReminderService) process(kind string, now time.Time) {
	pending, err := s.repo.GetPendingReminders(kind, now)
	if err != nil {
		logger.Error.Printf("ReminderService: не удалось получить брони (%s): %v", kind, err)
		return
	}
	for _, r := range pending {
		s.sendOne(kind, r)
	}
}

func (s *ReminderService) sendOne(kind string, r models.Reservation) {
	if r.User.Email == "" {
		// Без email отправлять некуда, но флаг ставим — чтобы не дёргаться каждую минуту.
		logger.Warn.Printf("ReminderService: у пользователя ID=%d нет email, пропускаем бронь #%d", r.UserID, r.ID)
		if err := s.repo.MarkReminderSent(r.ID, kind); err != nil {
			logger.Error.Printf("ReminderService: не удалось пометить бронь #%d: %v", r.ID, err)
		}
		return
	}

	hoursBefore := 24
	if kind == "3h" {
		hoursBefore = 3
	}

	rem := email.BookingReminder{
		To:            r.User.Email,
		UserName:      r.User.FullName,
		WorkspaceName: r.Workspace.NameOrNumber,
		LocationName:  r.Workspace.Location.Name,
		StartTime:     r.StartTime,
		EndTime:       r.EndTime,
		HoursBefore:   hoursBefore,
	}

	if err := s.emailSender.SendBookingReminder(rem); err != nil {
		// Не помечаем как отправленное — попробуем на следующем тике (пока бронь не началась).
		logger.Error.Printf("ReminderService: SMTP ошибка для брони #%d (%s): %v", r.ID, kind, err)
		return
	}

	if err := s.repo.MarkReminderSent(r.ID, kind); err != nil {
		logger.Error.Printf("ReminderService: письмо ушло, но флаг не выставился для брони #%d: %v", r.ID, err)
	}
}
