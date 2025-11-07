package models

import (
	"time"

	"github.com/google/uuid"
)

type Engagement struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`

	// Etkileşimin hedefi olan modelin ID ve tipi (post, comment vb.)
	TargetID   uuid.UUID `gorm:"type:uuid;index;not null" json:"target_id"`
	TargetType string    `gorm:"size:255;index;not null" json:"target_type"`

	// Etkileşim türü (like, comment, share, save vb.)
	Type string `gorm:"size:50;index;not null" json:"type"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}
