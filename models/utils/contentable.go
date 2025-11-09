package utils

import "github.com/google/uuid"

type BaseContentable struct {
	ContentableID   uuid.UUID `gorm:"type:uuid;index"`
	ContentableType string    `gorm:"size:50;index"`
}
