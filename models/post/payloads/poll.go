package payloads

import (
	"coolvibes/models"
	"coolvibes/models/utils"
	"time"

	"github.com/google/uuid"
)

const (
	ContentablePollPost = "post"
)

type PollKind string

const (
	PollKindSingle   PollKind = "single"
	PollKindMultiple PollKind = "multiple"
	PollKindRanked   PollKind = "ranked"
	PollKindWeighted PollKind = "weighted"
)

type Poll struct {
	ID              uuid.UUID             `gorm:"type:uuid;primaryKey" json:"id"`
	ContentableID   uuid.UUID             `json:"contentable_id"`
	ContentableType string                `json:"contentable_type"`
	Question        utils.LocalizedString `gorm:"type:jsonb" json:"question"`
	Duration        string                `gorm:"default:0" json:"duration"`
	Kind            PollKind              `gorm:"type:varchar(16);default:'single'" json:"kind"`
	MaxSelectable   int                   `gorm:"default:1" json:"max_selectable"` // 🔥 Pro seçim sistemi
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`

	Choices []PollChoice `gorm:"foreignKey:PollID;constraint:OnDelete:CASCADE" json:"choices,omitempty"`
}

type PollChoice struct {
	ID           uuid.UUID             `gorm:"type:uuid;primaryKey" json:"id"`
	PollID       uuid.UUID             `gorm:"type:uuid;index;not null" json:"poll_id"`
	DisplayOrder int                   `gorm:"default:0" json:"display_order"`
	Label        utils.LocalizedString `gorm:"type:jsonb" json:"label"`
	VoteCount    int                   `gorm:"default:0" json:"vote_count"`

	Votes []PollVote `gorm:"foreignKey:ChoiceID;constraint:OnDelete:CASCADE" json:"votes,omitempty"`
}

type PollVote struct {
	ID        uuid.UUID   `gorm:"type:uuid;primaryKey" json:"id"`
	ChoiceID  uuid.UUID   `gorm:"type:uuid;index;not null" json:"choice_id"`
	UserID    uuid.UUID   `gorm:"type:uuid;index;not null" json:"user_id"`
	User      models.User `gorm:"foreignKey:UserID;references:ID" json:"user"`
	Weight    int         `gorm:"default:1" json:"weight"` // weighted için
	Rank      int         `gorm:"default:0" json:"rank"`   // ranked için
	CreatedAt time.Time   `json:"created_at"`
}
