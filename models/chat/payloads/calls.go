// payloads/call.go
package payloads

import (
	"time"

	"github.com/google/uuid"
)

type CallStatus string

const (
	CallMissed   CallStatus = "missed"
	CallRejected CallStatus = "rejected"
	CallEnded    CallStatus = "ended"
	CallOngoing  CallStatus = "ongoing"
)

type Call struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	CallerID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	ReceiverID uuid.UUID  `gorm:"type:uuid;not null;index"`
	CallType   string     `gorm:"type:varchar(8);not null"` // audio, video
	Status     CallStatus `gorm:"type:varchar(16);not null"`
	Duration   int        `gorm:"not null"` // saniye cinsinden

	StartedAt *time.Time
	EndedAt   *time.Time
	Note      *string

	CreatedAt time.Time
}

func (Call) TableName() string {
	return "messages_call"
}
