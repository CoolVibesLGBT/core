package models

import (
	"time"

	"gorm.io/gorm"
)

type VapidKey struct {
	ID         uint   `gorm:"primaryKey"`
	PublicKey  string `gorm:"type:text;not null"`
	PrivateKey string `gorm:"type:text;not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}
