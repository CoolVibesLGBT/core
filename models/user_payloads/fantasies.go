package user_payloads

import (
	"coolvibes/models/utils"

	"github.com/google/uuid"
)

type Fantasy struct {
	ID           uuid.UUID              `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	DisplayOrder int                    `gorm:"not null" json:"display_order"` // 0,1,2...
	Slug         string                 `gorm:"type:varchar(50);not null" json:"slug"`
	Category     *utils.LocalizedString `gorm:"type:jsonb" json:"category,omitempty"`
	Label        utils.LocalizedString  `gorm:"type:jsonb;not null" json:"label"`       // Çoklu dil desteği
	Description  utils.LocalizedString  `gorm:"type:jsonb;not null" json:"description"` // Çoklu dil desteği
}

type UserFantasy struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"`
	FantasyID uuid.UUID `gorm:"type:uuid;index;not null" json:"fantasy_id"`
	Notes     *string   `gorm:"type:text" json:"notes,omitempty"`

	Fantasy *Fantasy `gorm:"foreignKey:FantasyID;references:ID" json:"fantasy,omitempty"`
}

func (UserFantasy) TableName() string {
	return "user_fantasies"
}

func (Fantasy) TableName() string {
	return "fantasies"
}
