package gigachat

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	mrand "math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	oauthURL       = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	chatURL        = "https://gigachat.devices.sberbank.ru/api/v1/chat/completions"
	defaultScope   = "GIGACHAT_API_PERS"
	defaultModel   = "GigaChat"
	requestTimeout = 30 * time.Second
	maxAttempts    = 3
	baseBackoff    = 500 * time.Millisecond
)

type GigaChatClient struct {
	authKey     string
	scope       string
	model       string
	client      *http.Client
	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

func NewClient(authKey string) *GigaChatClient {
	scope := os.Getenv("GIGACHAT_SCOPE")
	if scope == "" {
		scope = defaultScope
	}
	model := os.Getenv("GIGACHAT_MODEL")
	if model == "" {
		model = defaultModel
	}

	// Сертификаты Минцифры обычно не находятся в системном хранилище — отключаем строгую проверку.
	// Для production стоит установить корневой сертификат «Russian Trusted Root CA».
	insecure := os.Getenv("GIGACHAT_INSECURE_SKIP_VERIFY") != "false"

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}

	if proxyURL := os.Getenv("GIGACHAT_PROXY_URL"); proxyURL != "" {
		if u, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}

	return &GigaChatClient{
		authKey: authKey,
		scope:   scope,
		model:   model,
		client: &http.Client{
			Timeout:   requestTimeout,
			Transport: transport,
		},
	}
}

// newRqUID — генерирует UUIDv4 без внешних зависимостей.
func newRqUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// fallback на math/rand — RqUID должен быть уникальным, но не обязан быть криптостойким.
		mrand.Read(b)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	h := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32])
}

type oauthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

func (c *GigaChatClient) getAccessToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.expiresAt.Add(-30*time.Second)) {
		return c.accessToken, nil
	}

	body := strings.NewReader("scope=" + url.QueryEscape(c.scope))
	req, err := http.NewRequest("POST", oauthURL, body)
	if err != nil {
		return "", fmt.Errorf("oauth: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("RqUID", newRqUID())
	req.Header.Set("Authorization", "Basic "+c.authKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("oauth: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oauth: status %d: %s", resp.StatusCode, string(respBody))
	}

	var ot oauthResponse
	if err := json.Unmarshal(respBody, &ot); err != nil {
		return "", fmt.Errorf("oauth: unmarshal: %w (body=%s)", err, string(respBody))
	}
	if ot.AccessToken == "" {
		return "", fmt.Errorf("oauth: empty access_token (body=%s)", string(respBody))
	}

	c.accessToken = ot.AccessToken
	// expires_at в миллисекундах
	if ot.ExpiresAt > 0 {
		c.expiresAt = time.UnixMilli(ot.ExpiresAt)
	} else {
		c.expiresAt = time.Now().Add(25 * time.Minute)
	}
	return c.accessToken, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func isRetriableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= 500
}

// GenerateContent — отправляет промпт в GigaChat и возвращает текстовый ответ.
// Промпт передаётся целиком как одно user-сообщение (system-инструкции уже внутри текста).
func (c *GigaChatClient) GenerateContent(prompt string) (string, error) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("gigachat: marshal: %w", err)
	}

	var (
		bodyBytes []byte
		lastErr   error
	)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		token, err := c.getAccessToken()
		if err != nil {
			lastErr = err
		} else {
			req, err := http.NewRequest("POST", chatURL, bytes.NewBuffer(jsonData))
			if err != nil {
				return "", fmt.Errorf("gigachat: create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := c.client.Do(req)
			if err != nil {
				lastErr = fmt.Errorf("gigachat: do request: %w", err)
			} else {
				bodyBytes, err = io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					lastErr = fmt.Errorf("gigachat: read body: %w", err)
				} else if resp.StatusCode == http.StatusOK {
					lastErr = nil
					break
				} else {
					lastErr = fmt.Errorf("gigachat: status %d: %s", resp.StatusCode, string(bodyBytes))
					// При 401 — токен мог протухнуть, сбросим, чтобы получить новый
					if resp.StatusCode == http.StatusUnauthorized {
						c.mu.Lock()
						c.accessToken = ""
						c.mu.Unlock()
					} else if !isRetriableStatus(resp.StatusCode) {
						return "", lastErr
					}
				}
			}
		}

		if attempt < maxAttempts {
			backoff := baseBackoff * time.Duration(1<<(attempt-1))
			jitter := time.Duration(mrand.Int63n(int64(baseBackoff)))
			time.Sleep(backoff + jitter)
		}
	}

	if lastErr != nil {
		return "", lastErr
	}

	var data chatResponse
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return "", fmt.Errorf("gigachat: unmarshal response: %w (body=%s)", err, string(bodyBytes))
	}
	if len(data.Choices) == 0 || data.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("gigachat: empty response (body=%s)", string(bodyBytes))
	}
	return data.Choices[0].Message.Content, nil
}
