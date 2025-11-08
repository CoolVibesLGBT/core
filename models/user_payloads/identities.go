package user_payloads

import (
	"coolvibes/models/post/shared"

	"github.com/google/uuid"
)

// GenderIdentity kullanıcı cinsiyet kimliğini temsil eder
type GenderIdentity struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	DisplayOrder int       `gorm:"not null" json:"display_order"` // 0,1,2...

	Name shared.LocalizedString `gorm:"type:jsonb" json:"name"`
}

// SexualOrientation kullanıcı cinsel yönelimini temsil eder
type SexualOrientation struct {
	ID           uuid.UUID              `gorm:"type:uuid;primaryKey" json:"id"`
	DisplayOrder int                    `gorm:"not null" json:"display_order"` // 0,1,2...
	Name         shared.LocalizedString `gorm:"type:jsonb" json:"name"`
}

// SexRole kullanıcının cinsel rolu (aktif/pasif/versatile vb.) için
type SexualRole struct {
	ID           uuid.UUID              `gorm:"type:uuid;primaryKey" json:"id"`
	DisplayOrder int                    `gorm:"not null" json:"display_order"` // 0,1,2...
	Name         shared.LocalizedString `gorm:"type:jsonb" json:"name"`
}

func (SexualRole) TableName() string {
	return "sexual_roles"
}
