package taxonomy

import (
	"time"

	"coolvibes/models/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Cluster struct {
	ID          uuid.UUID              `gorm:"type:uuid;primaryKey" json:"id"`
	Pillar      string                 `gorm:"size:50;not null;index" json:"pillar"`
	Name        utils.LocalizedString  `gorm:"type:jsonb;not null" json:"name"`
	Slug        string                 `gorm:"size:150;not null" json:"slug"`
	Description *utils.LocalizedString `gorm:"type:jsonb" json:"description,omitempty"`
	IsActive    bool                   `gorm:"default:true;index" json:"is_active"`

	MetaTitle       *utils.LocalizedString `gorm:"type:jsonb" json:"meta_title,omitempty"`
	MetaDescription *utils.LocalizedString `gorm:"type:jsonb" json:"meta_description,omitempty"`

	Synonyms []Synonym   `gorm:"foreignKey:ClusterID;constraint:OnDelete:CASCADE" json:"synonyms,omitempty"`
	Posts    []uuid.UUID `gorm:"-" json:"post_ids,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
