package handlers

import (
	"csbs/backend/internal/api/middleware"
	"csbs/backend/internal/models"
	"csbs/backend/internal/service"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type LicenseHandler struct {
	service service.LicenseService
}

func NewLicenseHandler(s service.LicenseService) *LicenseHandler {
	return &LicenseHandler{service: s}
}

func (h *LicenseHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(AuthMiddleware)

	// Текущее состояние лицензии видят оба админа.
	r.With(middleware.RequireRole(models.RoleCoworkAdmin, models.RoleSystemAdmin)).
		Get("/", h.getInfo)

	// Установить новый ключ может только системный администратор.
	r.With(middleware.RequireRole(models.RoleSystemAdmin)).
		Post("/", h.install)

	return r
}

func (h *LicenseHandler) getInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.service.Info())
}

type installLicenseReq struct {
	Token string `json:"token"`
}

func (h *LicenseHandler) install(w http.ResponseWriter, r *http.Request) {
	var req installLicenseReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	info, err := h.service.Install(req.Token)
	if err != nil {
		// Невалидная подпись/просроченный токен — это ошибка ввода, а не сервера.
		http.Error(w, "Лицензия не принята: "+err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, info)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
