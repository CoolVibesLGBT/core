package user_payloads

import (
	"coolvibes/models/utils"

	"github.com/google/uuid"
)

// Interest = ana kategori
type Interest struct {
	ID    uuid.UUID             `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Name  utils.LocalizedString `gorm:"type:jsonb;not null" json:"name"` // Çoklu dil desteği
	Items []*InterestItem       `gorm:"foreignKey:InterestID" json:"items,omitempty"`
}

// InterestItem = alt ilgi alanı
type InterestItem struct {
	ID         uuid.UUID             `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	InterestID uuid.UUID             `gorm:"type:uuid;index;not null" json:"interest_id"`
	Name       utils.LocalizedString `gorm:"type:jsonb;not null" json:"name"` // Çoklu dil desteği
	Emoji      string                `gorm:"type:varchar(10)" json:"emoji,omitempty"`
	Interest   *Interest             `gorm:"foreignKey:InterestID;references:ID" json:"interest,omitempty"`
}

type UserInterest struct {
	ID             uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	UserID         uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"`
	InterestItemID uuid.UUID `gorm:"type:uuid;index;not null" json:"interest_item_id"`
	Notes          *string   `gorm:"type:text" json:"notes,omitempty"`

	InterestItem *InterestItem `gorm:"foreignKey:InterestItemID;references:ID" json:"interest_item,omitempty"`
}

func (UserInterest) TableName() string {
	return "user_interests"
}

func (Interest) TableName() string {
	return "interests"
}

func (InterestItem) TableName() string {
	return "interest_items"
}
