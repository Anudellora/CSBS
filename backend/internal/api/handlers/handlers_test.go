package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"csbs/backend/internal/api/handlers"
	"csbs/backend/internal/api/middleware"
	"csbs/backend/internal/models"
	"csbs/backend/internal/service"

	"github.com/golang-jwt/jwt/v5"
)

// ─────────────────────────────────────────────────────────────────────────────
// Mock-реализации сервисных интерфейсов

// -- TariffService mock --

type mockTariffService struct {
	tariffs []models.Tariff
	err     error
}

func (m *mockTariffService) GetAll() ([]models.Tariff, error)    { return m.tariffs, m.err }
func (m *mockTariffService) CreateTariff(t *models.Tariff) error { return m.err }
func (m *mockTariffService) UpdateTariff(t *models.Tariff) error { return m.err }
func (m *mockTariffService) DeleteTariff(id uint) error          { return m.err }

var _ service.TariffService = (*mockTariffService)(nil)

// -- UserService mock --

type mockUserService struct {
	user  *models.User
	token string
	err   error
}

func (m *mockUserService) Register(name, email, phone, password, role string) (*models.User, error) {
	return m.user, m.err
}
func (m *mockUserService) Login(email, password string) (string, error)  { return m.token, m.err }
func (m *mockUserService) GetUserByID(id uint) (*models.User, error)     { return m.user, m.err }
func (m *mockUserService) GetAllUsers() ([]models.User, error)           { return nil, m.err }
func (m *mockUserService) UpdateUserStatus(id uint, status string) error { return m.err }
func (m *mockUserService) UpdateUserRole(id uint, roleName string) error { return m.err }

var _ service.UserService = (*mockUserService)(nil)

// -- ReservationService mock --

type mockReservationService struct {
	ids          []uint
	reservations []models.Reservation
	reservation  *models.Reservation
	err          error
}

func (m *mockReservationService) CreateReservation(userID, workspaceID, tariffID uint, s, e time.Time) (*models.Reservation, error) {
	return m.reservation, m.err
}
func (m *mockReservationService) GetUserReservations(userID uint) ([]models.Reservation, error) {
	return m.reservations, m.err
}
func (m *mockReservationService) GetAllReservations() ([]models.Reservation, error) {
	return m.reservations, m.err
}
func (m *mockReservationService) UpdateReservation(id, wsID, tariffID uint, s, e time.Time, status string, actor uint) (*models.Reservation, error) {
	return m.reservation, m.err
}
func (m *mockReservationService) DeleteReservation(id, actor uint) error { return m.err }
func (m *mockReservationService) GetUnavailableWorkspaceIDs(s, e time.Time) ([]uint, error) {
	return m.ids, m.err
}

var _ service.ReservationService = (*mockReservationService)(nil)

// ─────────────────────────────────────────────────────────────────────────────
// Вспомогательная функция: создаёт JWT с известным секретом бэкенда
// ─────────────────────────────────────────────────────────────────────────────

func makeJWT(userID uint, role string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": float64(userID),
		"role":    role,
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte("my-secret-key"))
	return signed
}

// withAuth добавляет JWT в контекст запроса так же, как это делает AuthMiddleware
func withAuth(r *http.Request, userID uint, role string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.UserRoleKey, role)
	return r.WithContext(ctx)
}

// ─────────────────────────────────────────────────────────────────────────────
// Unit test 1: GET /tariffs — успешный ответ со списком тарифов
// ─────────────────────────────────────────────────────────────────────────────

func TestGetAllTariffs_ReturnsListWithStatus200(t *testing.T) {
	mockTariffs := []models.Tariff{
		{Price: 500, DurationMinutes: 60, LocationID: 1},
		{Price: 1500, DurationMinutes: 240, LocationID: 1},
	}
	svc := &mockTariffService{tariffs: mockTariffs}
	h := handlers.NewTariffHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ожидался статус 200, получен %d", rec.Code)
	}

	var result []models.Tariff
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("не удалось распарсить JSON: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("ожидалось 2 тарифа, получено %d", len(result))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Unit test 2: GET /tariffs — сервис вернул ошибку → 500
// ─────────────────────────────────────────────────────────────────────────────

func TestGetAllTariffs_ServiceError_Returns500(t *testing.T) {
	svc := &mockTariffService{err: errors.New("db connection failed")}
	h := handlers.NewTariffHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("ожидался статус 500, получен %d", rec.Code)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Unit test 3: POST /users/register — успешная регистрация → 201
// ─────────────────────────────────────────────────────────────────────────────

func TestRegisterUser_ValidBody_Returns201(t *testing.T) {
	createdUser := &models.User{
		FullName: "Иван Тестов",
		Email:    "ivan@test.com",
	}
	svc := &mockUserService{user: createdUser}
	h := handlers.NewUserHandler(svc)

	body := `{"name":"Иван Тестов","email":"ivan@test.com","phone":"+79991234567","password":"Password1"}`
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("ожидался статус 201, получен %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("ожидался Content-Type application/json, получен %s", ct)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Unit test 4: POST /users/login — успешный вход устанавливает auth cookie
// ─────────────────────────────────────────────────────────────────────────────

func TestLogin_ValidCredentials_SetsAuthCookie(t *testing.T) {
	fakeToken := makeJWT(1, models.RoleUser)
	svc := &mockUserService{token: fakeToken}
	h := handlers.NewUserHandler(svc)

	body := `{"email":"ivan@test.com","password":"Password1"}`
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ожидался статус 200, получен %d: %s", rec.Code, rec.Body.String())
	}

	// Проверяем, что установлена HttpOnly-кука с токеном
	cookies := rec.Result().Cookies()
	var authCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "auth_token" {
			authCookie = c
			break
		}
	}
	if authCookie == nil {
		t.Fatal("кука auth_token не установлена в ответе")
	}
	if !authCookie.HttpOnly {
		t.Error("кука auth_token должна быть HttpOnly")
	}
	if authCookie.Value != fakeToken {
		t.Errorf("значение куки не совпадает с токеном: got %q", authCookie.Value)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Unit test 5: GET /reservations/availability — с валидным JWT возвращает IDs
// ─────────────────────────────────────────────────────────────────────────────

func TestGetAvailability_WithValidToken_ReturnsUnavailableIDs(t *testing.T) {
	svc := &mockReservationService{ids: []uint{3, 7, 12}}
	h := handlers.NewReservationHandler(svc)

	url := "/availability?start_time=2026-05-10T09:00:00Z&end_time=2026-05-10T10:00:00Z"
	req := httptest.NewRequest(http.MethodGet, url, nil)

	// Добавляем JWT в cookie — как это делает браузер после логина
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: makeJWT(1, models.RoleUser)})

	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ожидался статус 200, получен %d: %s", rec.Code, rec.Body.String())
	}

	var ids []uint
	if err := json.NewDecoder(rec.Body).Decode(&ids); err != nil {
		t.Fatalf("не удалось распарсить JSON: %v", err)
	}
	if len(ids) != 3 {
		t.Errorf("ожидалось 3 занятых места, получено %d", len(ids))
	}
}
