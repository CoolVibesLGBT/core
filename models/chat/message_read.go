package chat

import (
	"coolvibes/models"
	"time"

	"github.com/google/uuid"
)

type MessageRead struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	MessageID uuid.UUID `gorm:"type:uuid;index;not null"`
	UserID    uuid.UUID `gorm:"type:uuid;index;not null"`
	ReadAt    time.Time `gorm:"autoCreateTime"`

	Message Message
	User    models.User
}

func (MessageRead) TableName() string {
	return "messages_reads"
}
