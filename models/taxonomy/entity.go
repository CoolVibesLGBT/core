package taxonomy

import (
	"core/helpers"
	models "core/models"
	"core/models/utils"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EntityType string

const (
	EntityPerson       EntityType = "person"
	EntityOrganization EntityType = "organization"
	EntityPlace        EntityType = "place"
	EntityPlatform     EntityType = "platform"
	EntityEvent        EntityType = "event"
	EntityTopic        EntityType = "topic"
)

type Entity struct {
	ID          uuid.UUID              `gorm:"type:uuid;primaryKey" json:"id"`
	Domain      models.DomainKind      `gorm:"size:50;not null;index;default:'coolvibes'" json:"domain"`
	Type        EntityType             `gorm:"type:varchar(40);not null;index" json:"type"`
	Slug        string                 `gorm:"size:150;not null;uniqueIndex" json:"slug"`
	Name        utils.LocalizedString  `gorm:"type:jsonb;not null" json:"name"`
	Description *utils.LocalizedString `gorm:"type:jsonb" json:"description,omitempty"`

	ExternalID *string `gorm:"size:120;index" json:"external_id,omitempty"`

	IsActive  bool      `gorm:"default:true;index" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (e *Entity) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if strings.TrimSpace(e.Slug) == "" {
		e.Slug = helpers.GenerateSlug(e.Name.ToString())
	}
	return nil
}
