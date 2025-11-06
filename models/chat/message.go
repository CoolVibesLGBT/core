package chat

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"bifrost/models/user"
)

type Message struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	ChatID   uuid.UUID `gorm:"type:uuid;index;not null"`
	SenderID uuid.UUID `gorm:"type:uuid;index;not null"`

	ReplyToID       *uuid.UUID
	ForwardedFromID *uuid.UUID

	Type        MessageType   `gorm:"type:varchar(16);index;not null"`
	Content     string        `gorm:"type:text"`
	PayloadType *string       `gorm:"size:32;index"`
	PayloadID   *uuid.UUID    `gorm:"type:uuid;index"`
	Status      MessageStatus `gorm:"type:varchar(16);default:'delivered';index"`
	IsSystem    bool
	IsPinned    bool

	// Relations
	Chat          Chat `gorm:"foreignKey:ChatID;constraint:OnDelete:CASCADE"`
	Sender        user.User
	ReplyTo       *Message `gorm:"foreignKey:ReplyToID"`
	ForwardedFrom *user.User

	Reads []MessageRead `gorm:"foreignKey:MessageID"`

	CreatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Message) TableName() string {
	return "messages"
}
