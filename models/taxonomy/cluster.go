package taxonomy

import (
	"strings"
	"time"

	"core/models/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Cluster struct {
	ID           uuid.UUID              `gorm:"type:uuid;primaryKey" json:"id"`
	PillarID     uuid.UUID              `gorm:"type:uuid;not null;index" json:"pillar_id"`
	Pillar       Pillar                 `gorm:"-" json:"-"`
	Name         utils.LocalizedString  `gorm:"type:jsonb;not null" json:"name"`
	Slug         string                 `gorm:"size:150;not null;index" json:"slug"`
	SearchVector string                 `gorm:"type:text;index" json:"-"`
	Description  *utils.LocalizedString `gorm:"type:jsonb" json:"description,omitempty"`
	IsActive     bool                   `gorm:"default:true;index" json:"is_active"`

	MetaTitle       *utils.LocalizedString `gorm:"type:jsonb" json:"meta_title,omitempty"`
	MetaDescription *utils.LocalizedString `gorm:"type:jsonb" json:"meta_description,omitempty"`

	Synonyms []Synonym   `gorm:"foreignKey:ClusterID;constraint:OnDelete:CASCADE" json:"synonyms,omitempty"`
	Posts    []uuid.UUID `gorm:"-" json:"post_ids,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (c *Cluster) BuildSearchVector() string {
	var parts []string

	parts = append(parts, c.Slug)

	if c.Name != nil {
		parts = append(parts, c.Name.ToString())
	}

	if c.MetaTitle != nil {
		parts = append(parts, c.MetaTitle.ToString())
	}

	if c.MetaDescription != nil {
		parts = append(parts, c.MetaDescription.ToString())
	}

	return strings.ToLower(strings.Join(parts, " "))
}

func (c *Cluster) BeforeSave(tx *gorm.DB) error {
	c.SearchVector = c.BuildSearchVector()
	return nil
}
