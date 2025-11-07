package payloads

import (
	"coolvibes/models/post/shared"
	"time"

	"github.com/google/uuid"
)

const (
	ContentablePollPost = "post"
)

type Poll struct {
	ID              uuid.UUID              `gorm:"type:uuid;primaryKey" json:"id"`
	PostID          uuid.UUID              `gorm:"type:uuid;index;not null" json:"post_id"`
	ContentableID   uuid.UUID              `json:"contentable_id"`
	ContentableType string                 `json:"contentable_type"`
	Question        shared.LocalizedString `gorm:"type:jsonb" json:"question"`
	Duration        string                 `gorm:"default:0" json:"duration"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`

	Choices []PollChoice `gorm:"foreignKey:PollID;constraint:OnDelete:CASCADE" json:"choices,omitempty"`
}

type PollChoice struct {
	ID        uuid.UUID              `gorm:"type:uuid;primaryKey" json:"id"`
	PollID    uuid.UUID              `gorm:"type:uuid;index;not null" json:"poll_id"`
	Label     shared.LocalizedString `gorm:"type:jsonb" json:"label"`
	VoteCount int                    `gorm:"default:0" json:"vote_count"`

	Votes []PollVote `gorm:"foreignKey:ChoiceID;constraint:OnDelete:CASCADE" json:"votes,omitempty"`
}

type PollVote struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ChoiceID  uuid.UUID `gorm:"type:uuid;index;not null" json:"choice_id"`
	UserID    uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}
