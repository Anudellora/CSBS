package service

import (
	"errors"

	"csbs/backend/internal/models"
	"csbs/backend/internal/repository"
	"csbs/backend/pkg/license"
	"csbs/backend/pkg/logger"
)

// LicenseService связывает офлайн-менеджер лицензий (pkg/license) с хранилищем.
// Источник лицензии при старте: сначала активная запись в БД, при её отсутствии —
// токен из .env (LICENSE_KEY). Установка нового ключа через Install проверяет
// подпись, сохраняет токен в БД и тут же подменяет лицензию в рантайме —
// без перезапуска сервера.
type LicenseService interface {
	Manager() *license.Manager
	Info() license.Info
	Install(token string) (license.Info, error)
}

type licenseServiceImpl struct {
	manager *license.Manager
	repo    repository.LicenseRepository
}

// NewLicenseService создаёт сервис и сразу загружает действующую лицензию.
// envToken — значение LICENSE_KEY (резервный источник, если в БД пусто).
func NewLicenseService(manager *license.Manager, repo repository.LicenseRepository, envToken string) LicenseService {
	s := &licenseServiceImpl{manager: manager, repo: repo}
	s.bootstrap(envToken)
	return s
}

func (s *licenseServiceImpl) bootstrap(envToken string) {
	if !s.manager.HasKey() {
		logger.Info.Println("Лицензирование: публичный ключ не задан (LICENSE_PUBLIC_KEY) — платные функции отключены")
		return
	}

	// 1. Пытаемся взять активную лицензию из БД.
	if dbLicense, err := s.repo.GetActive(); err != nil {
		logger.Error.Printf("Лицензирование: ошибка чтения лицензии из БД: %v", err)
	} else if dbLicense != nil {
		if claims, err := s.manager.Load(dbLicense.Token); err != nil {
			logger.Error.Printf("Лицензирование: лицензия из БД невалидна: %v", err)
		} else {
			logger.Info.Printf("Лицензирование: активна лицензия из БД (клиент=%s, план=%s)", claims.CustomerID, claims.Plan)
			return
		}
	}

	// 2. Резервный источник — токен из .env.
	if envToken == "" {
		logger.Info.Println("Лицензирование: лицензия не установлена (нет записи в БД и пустой LICENSE_KEY)")
		return
	}
	claims, err := s.manager.Load(envToken)
	if err != nil {
		logger.Error.Printf("Лицензирование: LICENSE_KEY из .env невалиден: %v", err)
		return
	}

	// Вставляем ключ из .env в БД при старте, чтобы он стал активной лицензией.
	// Идемпотентно: сюда попадаем только если в БД ещё нет активной лицензии,
	// поэтому на последующих запусках ключ будет браться уже из БД.
	rec := &models.License{Token: envToken, CustomerID: claims.CustomerID, Plan: claims.Plan}
	if err := s.repo.SaveActive(rec); err != nil {
		logger.Error.Printf("Лицензирование: не удалось вставить лицензию из .env в БД: %v", err)
		logger.Info.Printf("Лицензирование: активна лицензия из .env (клиент=%s, план=%s)", claims.CustomerID, claims.Plan)
		return
	}
	logger.Info.Printf("Лицензирование: лицензия из .env вставлена в БД и активирована (клиент=%s, план=%s)", claims.CustomerID, claims.Plan)
}

func (s *licenseServiceImpl) Manager() *license.Manager {
	return s.manager
}

func (s *licenseServiceImpl) Info() license.Info {
	return s.manager.Info()
}

// Install проверяет и устанавливает новый лицензионный токен (из админки).
func (s *licenseServiceImpl) Install(token string) (license.Info, error) {
	if token == "" {
		return license.Info{}, errors.New("пустой лицензионный токен")
	}

	// Сначала проверяем подпись и срок — невалидный токен в БД не сохраняем.
	claims, err := s.manager.Load(token)
	if err != nil {
		return license.Info{}, err
	}

	rec := &models.License{
		Token:      token,
		CustomerID: claims.CustomerID,
		Plan:       claims.Plan,
	}
	if err := s.repo.SaveActive(rec); err != nil {
		return license.Info{}, err
	}

	logger.Info.Printf("Лицензирование: установлена новая лицензия (клиент=%s, план=%s)", claims.CustomerID, claims.Plan)
	return s.manager.Info(), nil
}
