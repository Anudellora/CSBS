package license

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// issueTestLicense выпускает валидную лицензию и возвращает токен + менеджер
// с соответствующим публичным ключом.
func issueTestLicense(t *testing.T, ttl time.Duration, features ...string) (string, *Manager) {
	t.Helper()
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	token, err := Issue(priv, IssueParams{
		CustomerID: "test-co",
		Plan:       "pro",
		Features:   features,
		Limits:     map[string]int{"users": 50},
		TTL:        ttl,
	})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	pubKey, err := ParsePublicKey(pub)
	if err != nil {
		t.Fatalf("ParsePublicKey: %v", err)
	}
	return token, NewManager(pubKey)
}

func TestVerifyValidLicense(t *testing.T) {
	token, mgr := issueTestLicense(t, time.Hour, "ai_chat", "analytics")

	claims, err := mgr.Load(token)
	if err != nil {
		t.Fatalf("Load валидной лицензии вернул ошибку: %v", err)
	}
	if claims.CustomerID != "test-co" || claims.Plan != "pro" {
		t.Fatalf("неверные claims: %+v", claims)
	}
	if !mgr.Valid() {
		t.Fatal("лицензия должна быть валидной")
	}
	if !mgr.HasFeature("ai_chat") || !mgr.HasFeature("analytics") {
		t.Fatal("ожидались фичи ai_chat и analytics")
	}
	if mgr.HasFeature("kafka") {
		t.Fatal("фичи kafka в лицензии не было")
	}
	if v, ok := mgr.Limit("users"); !ok || v != 50 {
		t.Fatalf("ожидался лимит users=50, получено %d (ok=%v)", v, ok)
	}
}

func TestVerifyRejectsTamperedToken(t *testing.T) {
	token, mgr := issueTestLicense(t, time.Hour, "ai_chat")

	// Портим символ в середине токена (в payload), не трогая последний символ
	// подписи: у него часть битов незначащая, и подмена может ничего не изменить.
	i := len(token) / 2
	for token[i] == '.' { // не попадаем в разделитель
		i++
	}
	tampered := token[:i] + flip(token[i]) + token[i+1:]
	if _, err := mgr.Load(tampered); err == nil {
		t.Fatal("подделанный токен должен быть отклонён")
	}
}

func TestVerifyRejectsWrongKey(t *testing.T) {
	token, _ := issueTestLicense(t, time.Hour, "ai_chat")
	// Менеджер с другим (чужим) публичным ключом.
	_, otherPub, _ := GenerateKeyPair()
	pubKey, _ := ParsePublicKey(otherPub)
	mgr := NewManager(pubKey)

	if _, err := mgr.Load(token); err == nil {
		t.Fatal("лицензия, подписанная другим ключом, должна быть отклонена")
	}
}

func TestExpiredLicenseIsInvalid(t *testing.T) {
	token, mgr := issueTestLicense(t, -time.Minute, "ai_chat") // уже истекла

	if _, err := mgr.Load(token); err == nil {
		t.Fatal("просроченная лицензия должна отклоняться при Load")
	}
	if mgr.Valid() {
		t.Fatal("просроченная лицензия не должна считаться валидной")
	}
}

func TestRequireFeatureMiddleware(t *testing.T) {
	token, mgr := issueTestLicense(t, time.Hour, "ai_chat")
	if _, err := mgr.Load(token); err != nil {
		t.Fatalf("Load: %v", err)
	}

	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Разрешённая фича → 200.
	rec := httptest.NewRecorder()
	mgr.RequireFeature("ai_chat")(ok).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("ожидался 200 для разрешённой фичи, получено %d", rec.Code)
	}

	// Отсутствующая фича → 402 Payment Required.
	rec = httptest.NewRecorder()
	mgr.RequireFeature("analytics")(ok).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("ожидался 402 для отсутствующей фичи, получено %d", rec.Code)
	}
}

func flip(b byte) string {
	if b == 'A' {
		return "B"
	}
	return "A"
}
