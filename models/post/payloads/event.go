package payloads

import (
	"time"

	"coolvibes/models/post/shared"
	global_shared "coolvibes/models/shared"

	"github.com/google/uuid"
)

type EventAttendee struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	EventID uuid.UUID `gorm:"type:uuid;not null;index" json:"event_id"`
	UserID  uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`

	Status   string    `gorm:"size:32;default:'interested'" json:"status"` // "going", "interested", "invited", "declined"
	JoinedAt time.Time `gorm:"autoCreateTime" json:"joined_at"`
}

type Event struct {
	ID          uuid.UUID              `gorm:"type:uuid;primaryKey" json:"id"`
	PostID      uuid.UUID              `gorm:"type:uuid;uniqueIndex;not null" json:"post_id"`
	Title       shared.LocalizedString `gorm:"type:jsonb" json:"title"`
	Description shared.LocalizedString `gorm:"type:jsonb" json:"description"`
	StartTime   *time.Time             `json:"start_time,omitempty"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	//	Location    *global_shared.Location `gorm:"polymorphic:Contentable;constraint:OnDelete:CASCADE" json:"location,omitempty"`
	Location *global_shared.Location `gorm:"polymorphic:Contentable;polymorphicValue:event;constraint:OnDelete:CASCADE" json:"location,omitempty"`

	Type      string          `gorm:"size:64;index" json:"type"`
	Attendees []EventAttendee `gorm:"foreignKey:EventID;constraint:OnDelete:CASCADE" json:"attendees,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

func (Event) TableName() string {
	return "events"
}

func (EventAttendee) TableName() string {
	return "event_attendees"
}
