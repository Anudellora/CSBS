package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// КЛЮЧЕВОЙ ПРИНЦИП БЕЗОПАСНОСТИ:
// функции из этого файла (генерация ключей и подпись токенов) используются
// ТОЛЬКО на стороне вендора — в утилите cmd/licensegen. Приватный ключ не
// должен попадать на бэкенд клиента и тем более в репозиторий.

// GenerateKeyPair генерирует новую пару ключей Ed25519 и возвращает их в base64.
// priv — хранить в секрете у вендора, pub — класть в LICENSE_PUBLIC_KEY бэкенда.
func GenerateKeyPair() (privB64, pubB64 string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(priv),
		base64.StdEncoding.EncodeToString(pub),
		nil
}

// ParsePrivateKey декодирует приватный ключ Ed25519 из base64.
func ParsePrivateKey(b64 string) (ed25519.PrivateKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, fmt.Errorf("не удалось декодировать приватный ключ из base64: %w", err)
	}
	if len(raw) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("неверная длина приватного ключа Ed25519: %d (ожидалось %d)", len(raw), ed25519.PrivateKeySize)
	}
	return ed25519.PrivateKey(raw), nil
}

// IssueParams — параметры выпускаемой лицензии.
type IssueParams struct {
	CustomerID string
	Plan       string
	Features   []string
	Limits     map[string]int
	TTL        time.Duration // срок действия от текущего момента
}

// Issue подписывает лицензионный токен приватным ключом (base64).
func Issue(privB64 string, p IssueParams) (string, error) {
	priv, err := ParsePrivateKey(privB64)
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := &Claims{
		CustomerID: p.CustomerID,
		Plan:       p.Plan,
		Features:   p.Features,
		Limits:     p.Limits,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   p.CustomerID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(p.TTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	return token.SignedString(priv)
}
