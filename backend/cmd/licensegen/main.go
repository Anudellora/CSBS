// Утилита вендора для выпуска лицензий. Запускать ТОЛЬКО на доверенной машине,
// где хранится приватный ключ. На бэкенд клиента она не нужна.
//
// 1) Сгенерировать пару ключей (один раз):
//
//	go run ./cmd/licensegen -genkeys
//
//	→ выведет приватный (хранить у себя) и публичный (в LICENSE_PUBLIC_KEY) ключи.
//
// 2) Выпустить лицензию клиенту:
//
//	go run ./cmd/licensegen \
//	  -key "<приватный ключ base64>" \
//	  -customer "acme-llc" -plan pro \
//	  -features ai_chat,analytics,kafka \
//	  -users 50 -workspaces 200 -days 365
//
//	→ выведет подписанный токен для LICENSE_KEY.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"csbs/backend/pkg/license"
)

func main() {
	genKeys := flag.Bool("genkeys", false, "сгенерировать новую пару ключей Ed25519 и выйти")
	privKey := flag.String("key", "", "приватный ключ Ed25519 в base64 (для выпуска лицензии)")
	customer := flag.String("customer", "", "идентификатор клиента (customer_id)")
	plan := flag.String("plan", "pro", "название тарифа")
	features := flag.String("features", "ai_chat,analytics,kafka", "список фич через запятую")
	users := flag.Int("users", 0, "лимит пользователей (0 — без лимита)")
	workspaces := flag.Int("workspaces", 0, "лимит рабочих мест (0 — без лимита)")
	days := flag.Int("days", 365, "срок действия лицензии в днях")
	flag.Parse()

	if *genKeys {
		priv, pub, err := license.GenerateKeyPair()
		if err != nil {
			fail("не удалось сгенерировать ключи: %v", err)
		}
		fmt.Println("# Приватный ключ — ХРАНИТЬ В СЕКРЕТЕ, НЕ КОММИТИТЬ:")
		fmt.Printf("LICENSE_PRIVATE_KEY=%s\n\n", priv)
		fmt.Println("# Публичный ключ — положить в backend/.env бэкенда:")
		fmt.Printf("LICENSE_PUBLIC_KEY=%s\n", pub)
		return
	}

	if *privKey == "" || *customer == "" {
		fmt.Fprintln(os.Stderr, "Ошибка: для выпуска лицензии нужны флаги -key и -customer.")
		fmt.Fprintln(os.Stderr, "Подсказка: сгенерируйте ключи через -genkeys, либо см. -h.")
		flag.Usage()
		os.Exit(2)
	}

	limits := map[string]int{}
	if *users > 0 {
		limits["users"] = *users
	}
	if *workspaces > 0 {
		limits["workspaces"] = *workspaces
	}

	token, err := license.Issue(*privKey, license.IssueParams{
		CustomerID: *customer,
		Plan:       *plan,
		Features:   splitCSV(*features),
		Limits:     limits,
		TTL:        time.Duration(*days) * 24 * time.Hour,
	})
	if err != nil {
		fail("не удалось выпустить лицензию: %v", err)
	}

	fmt.Println("# Лицензия выпущена. Передайте клиенту строку ниже (в LICENSE_KEY бэкенда):")
	fmt.Println(token)
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
