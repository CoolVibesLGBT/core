package models

import (
	"time"

	"github.com/google/uuid"
)

type Hashtag struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`

	// Hangi modelin hashtag'i olduğu
	TaggableID   uuid.UUID `gorm:"type:uuid;index;not null" json:"taggable_id"`
	TaggableType string    `gorm:"size:255;index;not null" json:"taggable_type"`

	// Hashtag'in metni (# işareti olmadan)
	Tag string `gorm:"size:100;index;not null" json:"tag"`

	CreatedAt time.Time `json:"created_at"`
}
