package taxonomy

import (
	"core/helpers"
	"core/models/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Synonym struct {
	ID           uuid.UUID             `gorm:"type:uuid;primaryKey" json:"id"`
	ClusterID    uuid.UUID             `gorm:"type:uuid;index;not null" json:"cluster_id"`
	Word         utils.LocalizedString `gorm:"type:jsonb;not null" json:"word"`
	Slug         string                `gorm:"size:150;not null;index" json:"slug"`
	IsPrimary    bool                  `gorm:"default:false" json:"is_primary"`
	SearchWeight int                   `gorm:"default:1" json:"search_weight"`
	CreatedAt    time.Time             `json:"created_at"`
}

func (s *Synonym) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	s.Slug = helpers.GenerateSlug(s.Slug)
	if s.SearchWeight <= 0 {
		s.SearchWeight = 1
	}
	return nil
}

func (s *Synonym) BeforeSave(tx *gorm.DB) error {
	s.Slug = helpers.GenerateSlug(s.Slug)
	if s.SearchWeight <= 0 {
		s.SearchWeight = 1
	}
	return nil
}
