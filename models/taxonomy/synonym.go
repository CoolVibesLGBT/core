package taxonomy

import (
	"core/models/utils"
	"time"

	"github.com/google/uuid"
)

type Synonym struct {
	ID           uuid.UUID             `gorm:"type:uuid;primaryKey" json:"id"`
	ClusterID    uuid.UUID             `gorm:"type:uuid;index;not null" json:"cluster_id"`
	Word         utils.LocalizedString `gorm:"type:jsonb;not null" json:"word"`
	Slug         string                `gorm:"size:150;not null;uniqueIndex" json:"slug"`
	IsPrimary    bool                  `gorm:"default:false" json:"is_primary"`
	SearchWeight int                   `gorm:"default:1" json:"search_weight"`
	CreatedAt    time.Time             `json:"created_at"`
}
