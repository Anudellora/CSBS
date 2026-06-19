package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	ServerPort      string
	GeminiAPIKey    string
	GigaChatAuthKey string
	SMTPHost        string
	SMTPPort        string
	SMTPUser        string
	SMTPPassword    string
	AppURL          string
	// KafkaBrokers — список адресов брокеров через запятую.
	// Пустое значение → Kafka не используется (продюсер и консьюмеры не стартуют).
	KafkaBrokers string
	// LicensePublicKey — публичный ключ Ed25519 (base64) для проверки лицензий.
	// Пусто → лицензирование выключено, платные функции недоступны.
	LicensePublicKey string
	// LicenseKey — подписанный лицензионный токен (резервный источник, если в БД пусто).
	LicenseKey string
}

// Load - загружает переменные из .env файла и возвращает структуру Config
func Load() *Config {
	// Пытаемся загрузить .env, но если файла нет,
	// то просто игнорируем ошибку и читаем системные переменные
	if err := godotenv.Load(); err != nil {
		log.Println("Не удалось найти файл .env. Используются системные переменные окружения")
	}

	return &Config{
		DBHost:           os.Getenv("DB_HOST"),
		DBPort:           os.Getenv("DB_PORT"),
		DBUser:           os.Getenv("DB_USER"),
		DBPassword:       os.Getenv("DB_PASSWORD"),
		DBName:           os.Getenv("DB_NAME"),
		ServerPort:       os.Getenv("SERVER_PORT"),
		GeminiAPIKey:     os.Getenv("GEMINI_API_KEY"),
		GigaChatAuthKey:  os.Getenv("GIGACHAT_AUTH_KEY"),
		SMTPHost:         os.Getenv("SMTP_HOST"),
		SMTPPort:         os.Getenv("SMTP_PORT"),
		SMTPUser:         os.Getenv("SMTP_USER"),
		SMTPPassword:     os.Getenv("SMTP_PASSWORD"),
		AppURL:           os.Getenv("APP_URL"),
		KafkaBrokers:     os.Getenv("KAFKA_BROKERS"),
		LicensePublicKey: os.Getenv("LICENSE_PUBLIC_KEY"),
		LicenseKey:       os.Getenv("LICENSE_KEY"),
	}
}
