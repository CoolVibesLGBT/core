package chat

import (
	"bifrost/models/user"
	"time"

	"github.com/google/uuid"
)

type ChatParticipant struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	ChatID      uuid.UUID       `gorm:"type:uuid;index;not null" json:"chat_id"`
	UserID      uuid.UUID       `gorm:"type:uuid;index;not null" json:"user_id"`
	Role        ParticipantRole `gorm:"type:varchar(32);default:'member'" json:"role"`
	IsMuted     bool            `json:"is_muted"`
	JoinedAt    time.Time       `json:"joined_at"`
	LeftAt      *time.Time      `json:"left_at,omitempty"`
	UnreadCount int             `gorm:"default:0" json:"unread_count"`
	Chat        Chat            `json:"chat,omitempty"`
	User        user.User       `json:"user,omitempty"`

	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	LastReadAt *time.Time `json:"last_read_at,omitempty"`
}
