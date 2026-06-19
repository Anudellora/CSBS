package models

import "gorm.io/gorm"

// License — установленный лицензионный ключ. Сам токен самодостаточен и
// проверяется публичным ключом; в БД он лежит, чтобы лицензию можно было
// ввести через админку без правки .env и перезапуска сервера.
//
// Денормализованные поля CustomerID/Plan хранятся только для удобного
// отображения в списке и не участвуют в проверке прав.
type License struct {
	gorm.Model

	Token      string `gorm:"type:text;not null"`
	CustomerID string `gorm:"size:255"`
	Plan       string `gorm:"size:64"`
	// Active — признак текущей лицензии. Одновременно активна только одна.
	Active bool `gorm:"not null;default:true;index"`
}
