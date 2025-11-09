package chat

import (
	"coolvibes/models/media"
	"coolvibes/models/post"
	"coolvibes/models/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Chat struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Type ChatType  `gorm:"index;not null" json:"type"` // private, group, channel

	Title       *utils.LocalizedString `gorm:"type:jsonb" json:"title"` // JSONB olarak sakla
	Description *utils.LocalizedString `gorm:"type:jsonb" json:"description"`

	AvatarID *uuid.UUID   `json:"avatar_id,omitempty"`
	Avatar   *media.Media `gorm:"constraint:OnDelete:SET NULL;foreignKey:AvatarID;references:ID" json:"avatar,omitempty"`

	CreatorID   uuid.UUID  `gorm:"type:uuid;index;not null" json:"creator_id"` // UUID olmalı
	PinnedMsgID *uuid.UUID `gorm:"type:uuid;index" json:"pinned_msg_id,omitempty"`
	PinnedMsg   *post.Post `gorm:"foreignKey:PinnedMsgID;references:ID" json:"pinned_msg,omitempty"`

	LastMessageID        *uuid.UUID `gorm:"type:uuid;index" json:"last_message_id,omitempty"`
	LastMessage          *post.Post `gorm:"foreignKey:LastMessageID;references:ID" json:"last_message,omitempty"`
	LastMessageTimestamp *time.Time `gorm:"last_message_timestamp" json:"last_message_timestamp,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Participants []ChatParticipant `json:"participants,omitempty"`
	Messages     []post.Post       `gorm:"polymorphic:Contentable;polymorphicValue:chat;constraint:OnDelete:CASCADE;" json:"messages,omitempty"`
}

func (Chat) TableName() string {
	return "chats"
}
