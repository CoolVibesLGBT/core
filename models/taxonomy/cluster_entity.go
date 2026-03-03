package taxonomy

import (
	"time"

	"github.com/google/uuid"
)

type ClusterEntity struct {
	ClusterID uuid.UUID `gorm:"type:uuid;primaryKey" json:"cluster_id"`
	EntityID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"entity_id"`
	CreatedAt time.Time `json:"created_at"`
}
