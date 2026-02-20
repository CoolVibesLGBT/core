package models

import (
	"time"

	"github.com/google/uuid"
)

type Hashtag struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Domain          DomainKind `gorm:"type:varchar(50);not null;index" json:"domain"`
	TaggableID      uuid.UUID  `gorm:"type:uuid;index;not null" json:"taggable_id"`
	TaggableType    string     `gorm:"size:255;index;not null" json:"taggable_type"`
	Tag             string     `gorm:"size:100;index;not null" json:"tag"`
	Slug            string     `gorm:"size:100;index;not null" json:"slug"`
	ParentID        *uuid.UUID `gorm:"type:uuid;index" json:"parent_id,omitempty"`
	Parent          *Hashtag   `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	RelatedHashtags []*Hashtag `gorm:"foreignKey:ParentID" json:"related_hashtags,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}
