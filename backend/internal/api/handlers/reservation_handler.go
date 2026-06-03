package handlers

import (
	"csbs/backend/internal/api/middleware"
	"csbs/backend/internal/models"
	"csbs/backend/internal/service"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

type ReservationHandler struct {
	service service.ReservationService
}

func NewReservationHandler(service service.ReservationService) *ReservationHandler {
	return &ReservationHandler{service: service}
}

func (h *ReservationHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(AuthMiddleware) // Все эндпоинты бронирований защищены JWT
	r.Post("/", h.create)
	r.Get("/", h.getUserReservations)
	r.Get("/availability", h.getAvailability)
	r.Get("/{id}/pass", h.getPass)

	// Админские роуты
	r.With(middleware.RequireRole(models.RoleCoworkAdmin, models.RoleSystemAdmin)).
		Get("/all", h.getAllReservations)
	r.With(middleware.RequireRole(models.RoleCoworkAdmin, models.RoleSystemAdmin)).
		Post("/admin", h.adminCreate)
	r.With(middleware.RequireRole(models.RoleCoworkAdmin, models.RoleSystemAdmin)).
		Put("/{id}", h.update)
	r.With(middleware.RequireRole(models.RoleCoworkAdmin, models.RoleSystemAdmin)).
		Delete("/{id}", h.delete)

	return r
}

type createReservationRequest struct {
	WorkspaceID uint   `json:"workspace_id"`
	TariffID    uint   `json:"tariff_id"`
	StartTime   string `json:"start_time"` // формат: "2026-03-15T10:00:00Z"
	EndTime     string `json:"end_time"`
}

type adminCreateReservationRequest struct {
	UserID      uint   `json:"user_id"`
	WorkspaceID uint   `json:"workspace_id"`
	TariffID    uint   `json:"tariff_id"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
}

type updateReservationRequest struct {
	WorkspaceID uint   `json:"workspace_id"`
	TariffID    uint   `json:"tariff_id"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Status      string `json:"status"`
}

func (h *ReservationHandler) create(w http.ResponseWriter, r *http.Request) {
	// Достаём user_id из контекста (положил туда AuthMiddleware)
	userID := r.Context().Value(middleware.UserIDKey).(uint)

	var req createReservationRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	startTime, _ := time.Parse(time.RFC3339, req.StartTime)
	endTime, _ := time.Parse(time.RFC3339, req.EndTime)

	reservation, err := h.service.CreateReservation(userID, req.WorkspaceID, req.TariffID, startTime, endTime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(reservation)
}

func (h *ReservationHandler) getUserReservations(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(uint)

	reservations, err := h.service.GetUserReservations(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reservations)
}

func (h *ReservationHandler) getAllReservations(w http.ResponseWriter, r *http.Request) {
	reservations, err := h.service.GetAllReservations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reservations)
}

func (h *ReservationHandler) adminCreate(w http.ResponseWriter, r *http.Request) {
	var req adminCreateReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}
	if req.UserID == 0 {
		http.Error(w, "user_id обязателен", http.StatusBadRequest)
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		http.Error(w, "Некорректный start_time", http.StatusBadRequest)
		return
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		http.Error(w, "Некорректный end_time", http.StatusBadRequest)
		return
	}

	reservation, err := h.service.CreateReservation(req.UserID, req.WorkspaceID, req.TariffID, startTime, endTime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(reservation)
}

func (h *ReservationHandler) update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Некорректный ID", http.StatusBadRequest)
		return
	}

	var req updateReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		http.Error(w, "Некорректный start_time", http.StatusBadRequest)
		return
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		http.Error(w, "Некорректный end_time", http.StatusBadRequest)
		return
	}

	actorUserID := r.Context().Value(middleware.UserIDKey).(uint)

	reservation, err := h.service.UpdateReservation(uint(id), req.WorkspaceID, req.TariffID, startTime, endTime, req.Status, actorUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reservation)
}

func (h *ReservationHandler) delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Некорректный ID", http.StatusBadRequest)
		return
	}

	actorUserID := r.Context().Value(middleware.UserIDKey).(uint)

	if err := h.service.DeleteReservation(uint(id), actorUserID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// getPass возвращает подписанный JWT-пропуск для активной брони пользователя.
// Фронт зашивает этот токен в QR-код; сканер на входе проверит подпись и срок действия.
func (h *ReservationHandler) getPass(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(uint)

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Некорректный ID", http.StatusBadRequest)
		return
	}

	reservation, err := h.service.GetReservationByID(uint(id))
	if err != nil {
		http.Error(w, "Бронирование не найдено", http.StatusNotFound)
		return
	}

	if reservation.UserID != userID {
		http.Error(w, "Бронирование принадлежит другому пользователю", http.StatusForbidden)
		return
	}

	if err := assertReservationActive(reservation); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token, err := signReservationPass(reservation)
	if err != nil {
		http.Error(w, "Не удалось создать пропуск", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token":         token,
		"reservation_id": reservation.ID,
		"workspace":     reservation.Workspace.NameOrNumber,
		"location":      reservation.Workspace.Location.Name,
		"start_time":    reservation.StartTime,
		"end_time":      reservation.EndTime,
		"expires_at":    reservation.EndTime,
	})
}

func assertReservationActive(r *models.Reservation) error {
	if r.Status == "отменено" {
		return errors.New("бронь отменена")
	}
	if time.Now().After(r.EndTime) {
		return errors.New("бронь уже завершена")
	}
	return nil
}

// signReservationPass подписывает JWT тем же секретом, что и auth-токены, но с purpose=checkin,
// чтобы пропуск нельзя было использовать для доступа к API, и наоборот.
func signReservationPass(r *models.Reservation) (string, error) {
	claims := jwt.MapClaims{
		"purpose":        "checkin",
		"reservation_id": r.ID,
		"user_id":        r.UserID,
		"workspace_id":   r.WorkspaceID,
		"start_time":     r.StartTime.Unix(),
		"exp":            r.EndTime.Unix(),
		"iat":            time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("my-secret-key"))
}

func (h *ReservationHandler) getAvailability(w http.ResponseWriter, r *http.Request) {
	startStr := r.URL.Query().Get("start_time")
	endStr := r.URL.Query().Get("end_time")

	if startStr == "" || endStr == "" {
		http.Error(w, "start_time and end_time query parameters are required", http.StatusBadRequest)
		return
	}

	startTime, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		http.Error(w, "invalid start_time format", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		http.Error(w, "invalid end_time format", http.StatusBadRequest)
		return
	}

	ids, err := h.service.GetUnavailableWorkspaceIDs(startTime, endTime)
	if err != nil {
		http.Error(w, "Failed to get availability", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ids)
}
