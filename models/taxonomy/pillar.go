package taxonomy

import (
	domaintaxonomy "core/domain/taxonomy"
	models "core/models"
	"core/models/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Pillar struct {
	ID              uuid.UUID              `gorm:"type:uuid;primaryKey" json:"id"`
	Domain          models.DomainKind      `gorm:"size:50;not null;index;default:'coolvibes'" json:"domain"`
	Slug            string                 `gorm:"size:150;not null;uniqueIndex" json:"slug"`
	Name            utils.LocalizedString  `gorm:"type:jsonb;not null" json:"name"`
	Description     *utils.LocalizedString `gorm:"type:jsonb" json:"description,omitempty"`
	IsActive        bool                   `gorm:"default:true;index" json:"is_active"`
	MetaTitle       *utils.LocalizedString `gorm:"type:jsonb" json:"meta_title,omitempty"`
	MetaDescription *utils.LocalizedString `gorm:"type:jsonb" json:"meta_description,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	DeletedAt       gorm.DeletedAt         `gorm:"index" json:"deleted_at,omitempty"`
	Clusters        []Cluster              `gorm:"foreignKey:PillarID;constraint:OnDelete:CASCADE" json:"clusters,omitempty"`
}

func (p *Pillar) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	p.Slug = domaintaxonomy.NormalizeSlug(p.Slug)
	return nil
}

func (p *Pillar) BeforeSave(tx *gorm.DB) error {
	p.Slug = domaintaxonomy.NormalizeSlug(p.Slug)
	return nil
}
