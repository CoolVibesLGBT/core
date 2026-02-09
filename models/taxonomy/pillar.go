package taxonomy

import (
	"coolvibes/models/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Pillar struct {
	ID          uuid.UUID              `gorm:"type:uuid;primaryKey" json:"id"`
	Slug        string                 `gorm:"size:150;not null;uniqueIndex" json:"slug"`
	Name        utils.LocalizedString  `gorm:"type:jsonb;not null" json:"name"`
	Description *utils.LocalizedString `gorm:"type:jsonb" json:"description,omitempty"`

	IsActive bool `gorm:"default:true;index" json:"is_active"`

	MetaTitle       *utils.LocalizedString `gorm:"type:jsonb" json:"meta_title,omitempty"`
	MetaDescription *utils.LocalizedString `gorm:"type:jsonb" json:"meta_description,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Clusters []Cluster `gorm:"foreignKey:PillarID;constraint:OnDelete:SET NULL" json:"clusters,omitempty"` // opsiyonel: pillar silinirse cluster'lar null olur
}
