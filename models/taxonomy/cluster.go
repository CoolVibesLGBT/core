package taxonomy

import (
	"strings"
	"time"

	"core/helpers"
	models "core/models"
	"core/models/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Cluster struct {
	ID       uuid.UUID         `gorm:"type:uuid;primaryKey" json:"id"`
	Domain   models.DomainKind `gorm:"size:50;not null;index;default:'coolvibes'" json:"domain"`
	PillarID uuid.UUID         `gorm:"type:uuid;not null;index" json:"pillar_id"`
	Pillar   Pillar            `gorm:"foreignKey:PillarID" json:"-"`

	ParentID        *uuid.UUID             `gorm:"type:uuid;index" json:"parent_id,omitempty"`
	Parent          *Cluster               `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children        []Cluster              `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Name            utils.LocalizedString  `gorm:"type:jsonb;not null" json:"name"`
	Slug            string                 `gorm:"size:150;not null;index" json:"slug"`
	SearchVector    string                 `gorm:"type:text;index" json:"-"`
	Description     *utils.LocalizedString `gorm:"type:jsonb" json:"description,omitempty"`
	IsActive        bool                   `gorm:"default:true;index" json:"is_active"`
	MetaTitle       *utils.LocalizedString `gorm:"type:jsonb" json:"meta_title,omitempty"`
	MetaDescription *utils.LocalizedString `gorm:"type:jsonb" json:"meta_description,omitempty"`

	Intents   []Intent       `gorm:"many2many:cluster_intents;constraint:OnDelete:CASCADE" json:"intents,omitempty"`
	Entities  []Entity       `gorm:"many2many:cluster_entities;constraint:OnDelete:CASCADE" json:"entities,omitempty"`
	Synonyms  []Synonym      `gorm:"foreignKey:ClusterID;constraint:OnDelete:CASCADE" json:"synonyms,omitempty"`
	Posts     []uuid.UUID    `gorm:"-" json:"post_ids,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (c *Cluster) BuildSearchVector() string {
	var parts []string

	parts = append(parts, c.Slug)

	if c.Slug != "" {
		normalized := helpers.GenerateSlug(c.Slug)  // gay-bar
		strict := helpers.SlugifyStrict(normalized) // gaybar

		parts = append(parts, normalized)
		parts = append(parts, strict)
	}

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
	c.Slug = helpers.GenerateSlug(c.Slug)
	c.SearchVector = c.BuildSearchVector()
	return nil
}

func (c *Cluster) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return c.BeforeSave(tx)
}
