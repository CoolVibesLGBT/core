package notifications

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Delivery is persisted in the same transaction as the in-app notification.
// Its notification ID is also the stable push tag used across retries.
type Delivery struct {
	ID                     uuid.UUID  `gorm:"type:uuid;primaryKey"`
	AvailableAt            time.Time  `gorm:"not null;index:idx_notification_delivery_due,priority:2"`
	CompletedAt            *time.Time `gorm:"index:idx_notification_delivery_due,priority:1"`
	LeaseUntil             *time.Time
	LeaseToken             *uuid.UUID     `gorm:"type:uuid"`
	Attempts               int            `gorm:"not null;default:0"`
	LastError              string         `gorm:"type:text"`
	RemainingSubscriptions datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (Delivery) TableName() string { return "notification_deliveries" }
