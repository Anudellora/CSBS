package license

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Manager держит активную лицензию в памяти и потокобезопасно отвечает на
// вопросы «валидна ли лицензия» и «есть ли фича». Лицензию можно подменить
// в рантайме через Load (например, после установки нового ключа из админки)
// без перезапуска сервера.
type Manager struct {
	mu     sync.RWMutex
	pub    ed25519.PublicKey
	claims *Claims
}

// NewManager создаёт менеджер с публичным ключом, которым будут проверяться
// все лицензии. Если pub == nil, любая лицензия считается невалидной.
func NewManager(pub ed25519.PublicKey) *Manager {
	return &Manager{pub: pub}
}

// HasKey сообщает, настроен ли публичный ключ (без него проверка невозможна).
func (m *Manager) HasKey() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.pub) == ed25519.PublicKeySize
}

// Load проверяет токен публичным ключом и, если он валиден, делает его
// активной лицензией. Возвращает разобранные claims или ошибку проверки.
func (m *Manager) Load(tokenStr string) (*Claims, error) {
	m.mu.RLock()
	pub := m.pub
	m.mu.RUnlock()

	claims, err := Verify(tokenStr, pub)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.claims = claims
	m.mu.Unlock()
	return claims, nil
}

// Valid сообщает, есть ли активная и непросроченная лицензия.
// Срок проверяется на текущий момент, а не только в момент Load,
// поэтому лицензия «протухнет» сама по достижении exp.
func (m *Manager) Valid() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.validLocked()
}

func (m *Manager) validLocked() bool {
	if m.claims == nil {
		return false
	}
	if exp := m.claims.ExpiresAt; exp != nil && exp.Before(time.Now()) {
		return false
	}
	return true
}

// HasFeature == лицензия валидна И содержит указанную фичу.
func (m *Manager) HasFeature(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.validLocked() && m.claims.HasFeature(name)
}

// Limit возвращает лимит по имени, если лицензия валидна.
func (m *Manager) Limit(name string) (int, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.validLocked() {
		return 0, false
	}
	return m.claims.Limit(name)
}

// Info — безопасный для отдачи наружу снимок состояния лицензии
// (без самого токена и без публичного ключа).
type Info struct {
	Active     bool           `json:"active"`
	CustomerID string         `json:"customer_id,omitempty"`
	Plan       string         `json:"plan,omitempty"`
	Features   []string       `json:"features,omitempty"`
	Limits     map[string]int `json:"limits,omitempty"`
	ExpiresAt  *time.Time     `json:"expires_at,omitempty"`
	Reason     string         `json:"reason,omitempty"`
}

// Info возвращает текущее состояние лицензии для отображения в админке.
func (m *Manager) Info() Info {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.claims == nil {
		reason := "лицензия не установлена"
		if len(m.pub) != ed25519.PublicKeySize {
			reason = "не настроен публичный ключ лицензии (LICENSE_PUBLIC_KEY)"
		}
		return Info{Active: false, Reason: reason}
	}

	info := Info{
		CustomerID: m.claims.CustomerID,
		Plan:       m.claims.Plan,
		Features:   m.claims.Features,
		Limits:     m.claims.Limits,
	}
	if exp := m.claims.ExpiresAt; exp != nil {
		t := exp.Time
		info.ExpiresAt = &t
	}
	if m.validLocked() {
		info.Active = true
	} else {
		info.Reason = "срок действия лицензии истёк"
	}
	return info
}

// RequireFeature — HTTP-middleware: пропускает запрос только если лицензия
// валидна и содержит фичу, иначе отвечает 402 Payment Required.
func (m *Manager) RequireFeature(feature string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !m.Valid() {
				writeError(w, "Лицензия отсутствует или истекла")
				return
			}
			if !m.HasFeature(feature) {
				writeError(w, "Функция \""+feature+"\" недоступна в вашем тарифе")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusPaymentRequired) // 402
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
