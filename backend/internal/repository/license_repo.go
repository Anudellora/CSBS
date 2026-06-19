package repository

import (
	"errors"

	"csbs/backend/internal/models"

	"gorm.io/gorm"
)

type LicenseRepository interface {
	// GetActive возвращает текущую активную лицензию или (nil, nil), если её нет.
	GetActive() (*models.License, error)
	// SaveActive делает новую лицензию единственной активной (старые гасит)
	// в одной транзакции.
	SaveActive(license *models.License) error
}

type licenseRepositoryImpl struct {
	db *gorm.DB
}

func NewLicenseRepository(db *gorm.DB) LicenseRepository {
	return &licenseRepositoryImpl{db: db}
}

func (r *licenseRepositoryImpl) GetActive() (*models.License, error) {
	var license models.License
	err := r.db.Where("active = ?", true).Order("id DESC").First(&license).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &license, nil
}

func (r *licenseRepositoryImpl) SaveActive(license *models.License) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Гасим все ранее активные лицензии, чтобы активной осталась одна.
		if err := tx.Model(&models.License{}).
			Where("active = ?", true).
			Update("active", false).Error; err != nil {
			return err
		}
		license.Active = true
		return tx.Create(license).Error
	})
}
