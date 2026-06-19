// Package license реализует офлайн-проверку лицензионного ключа на основе
// подписанного JWT-токена (алгоритм EdDSA / Ed25519).
//
// Модель доверия асимметричная: приватный ключ есть ТОЛЬКО у вендора и
// используется утилитой cmd/licensegen для выпуска токенов. Бэкенд знает
// лишь публичный ключ и может только ПРОВЕРИТЬ подпись — подделать или
// «продлить» токен без приватного ключа невозможно.
//
// Токен самодостаточен (offline): внутри лежат план, список фич и лимиты,
// поэтому при проверке не нужен поход на сервер лицензий.
package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Claims — полезная нагрузка лицензионного токена.
//
// Пример:
//
//	{ "customer_id": "acme-llc", "plan": "pro", "exp": 1735689600,
//	  "features": ["ai_chat","analytics","kafka"],
//	  "limits": { "users": 50, "workspaces": 200 } }
type Claims struct {
	CustomerID string         `json:"customer_id"`
	Plan       string         `json:"plan"`
	Features   []string       `json:"features"`
	Limits     map[string]int `json:"limits"`
	jwt.RegisteredClaims
}

// HasFeature сообщает, входит ли фича в лицензию.
func (c *Claims) HasFeature(name string) bool {
	for _, f := range c.Features {
		if f == name {
			return true
		}
	}
	return false
}

// Limit возвращает числовой лимит по имени (например "users") и флаг наличия.
func (c *Claims) Limit(name string) (int, bool) {
	v, ok := c.Limits[name]
	return v, ok
}

// ParsePublicKey декодирует публичный ключ Ed25519 из base64.
func ParsePublicKey(b64 string) (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, fmt.Errorf("не удалось декодировать публичный ключ из base64: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("неверная длина публичного ключа Ed25519: %d (ожидалось %d)", len(raw), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(raw), nil
}

// Verify проверяет подпись и срок действия токена публичным ключом.
// Возвращает разобранные claims только если токен валиден и не истёк.
func Verify(tokenStr string, pub ed25519.PublicKey) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		// Жёстко фиксируем алгоритм: иначе злоумышленник мог бы прислать
		// токен с alg=none или симметричным HMAC и обойти проверку.
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("неожиданный алгоритм подписи: %v", t.Header["alg"])
		}
		return pub, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("невалидная лицензия")
	}
	return claims, nil
}
