package taxonomy

import (
	"time"

	"github.com/google/uuid"
)

type ClusterIntent struct {
	ClusterID uuid.UUID `gorm:"type:uuid;primaryKey" json:"cluster_id"`
	IntentID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"intent_id"`
	CreatedAt time.Time `json:"created_at"`
}
