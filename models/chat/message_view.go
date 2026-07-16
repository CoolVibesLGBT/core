package chat

import (
	"time"

	"github.com/google/uuid"
)

// MessageView records a one-time media consumption per recipient. The
// composite unique index is the final concurrency guard; application checks
// are only used to return a friendlier error.
type MessageView struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	MessageID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uidx_chat_message_viewer" json:"message_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uidx_chat_message_viewer" json:"user_id"`
	ViewedAt  time.Time `gorm:"not null" json:"viewed_at"`
}

func (MessageView) TableName() string { return "chat_message_views" }
