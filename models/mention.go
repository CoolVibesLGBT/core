package models

import (
	"time"

	"coolvibes/models/user"

	"github.com/google/uuid"
)

type Mention struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`

	// Hangi modelin mention'ı olduğu
	MentionableID   uuid.UUID `gorm:"type:uuid;index;not null" json:"mentionable_id"`
	MentionableType string    `gorm:"size:255;index;not null" json:"mentionable_type"`

	UserID uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"`
	User   user.User `gorm:"foreignKey:UserID" json:"user"` // Mention edilen kullanıcı

	CreatedAt time.Time `json:"created_at"`
}
