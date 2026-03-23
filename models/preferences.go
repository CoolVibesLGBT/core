package models

import (
	"context"
	"core/models/utils"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PreferenceItem struct {
	ID           uuid.UUID              `json:"id" gorm:"-"` // GORM ignorlasın
	DisplayOrder int                    `json:"display_order" gorm:"-"`
	BitIndex     int64                  `json:"bit_index" gorm:"-"`
	Slug         *string                `json:"slug,omitempty" gorm:"-"`
	Title        *utils.LocalizedString `json:"title,omitempty" gorm:"-"`
	Description  *utils.LocalizedString `json:"description,omitempty" gorm:"-"`
	Icon         *string                `json:"icon,omitempty" gorm:"-"`
	Visible      bool                   `json:"visible" gorm:"-"`
}

type PreferenceCategory struct {
	ID            uuid.UUID              `json:"id" gorm:"-"`
	DisplayOrder  int                    `json:"display_order" gorm:"-"`
	Tag           *string                `json:"tag,omitempty" gorm:"-"`
	Slug          *string                `json:"slug,omitempty" gorm:"-"`
	Title         *utils.LocalizedString `json:"title,omitempty" gorm:"-"`
	Description   *utils.LocalizedString `json:"description,omitempty" gorm:"-"`
	Icon          *string                `json:"icon,omitempty" gorm:"-"`
	AllowMultiple bool                   `json:"allow_multiple" gorm:"-"`
	Items         []PreferenceItem       `json:"items" gorm:"-"`
}

type PreferencesData struct {
	Attributes []PreferenceCategory `json:"attributes" gorm:"-"`
	Interests  []PreferenceCategory `json:"interests" gorm:"-"`
	Fantasies  []PreferenceCategory `json:"fantasies" gorm:"-"`
}
type Preferences struct {
	ID       uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	Category string          `gorm:"type:varchar(50);not null" json:"category"`
	Data     PreferencesData `gorm:"type:jsonb" json:"data"`
	BitCount int64           `json:"bit_count"`
}

func (Preferences) TableName() string {
	return "preferences"
}

func (p PreferencesData) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	return utils.JSONBGormValue(ctx, db, p)
}

func (p *PreferencesData) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	if err := utils.ScanJSON(value, p); err != nil {
		return fmt.Errorf("cannot decode PreferencesData: %w", err)
	}
	return nil
}

func (pc PreferenceCategory) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	return utils.JSONBGormValue(ctx, db, pc)
}

func (pc *PreferenceCategory) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	if err := utils.ScanJSON(value, pc); err != nil {
		return fmt.Errorf("cannot decode PreferenceCategory: %w", err)
	}
	return nil
}
