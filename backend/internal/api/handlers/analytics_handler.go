package handlers

import (
	"csbs/backend/internal/api/middleware"
	"csbs/backend/internal/models"
	"csbs/backend/internal/service"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type AnalyticsHandler struct {
	service service.AnalyticsService
}

func NewAnalyticsHandler(s service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{service: s}
}

func (h *AnalyticsHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(AuthMiddleware)
	r.Use(middleware.RequireRole(models.RoleCoworkAdmin, models.RoleSystemAdmin))
	r.Get("/dashboard", h.getDashboard)
	return r
}

func (h *AnalyticsHandler) getDashboard(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.GetDashboard()
	if err != nil {
		http.Error(w, "Не удалось собрать аналитику: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
