package user_payloads

import (
	"coolvibes/models/post/shared"

	"github.com/google/uuid"
)

const (
	UserAttributeHairColor          = "hair_color"          // Saç rengi
	UserAttributeEyeColor           = "eye_color"           // Göz rengi
	UserAttributeSkinColor          = "skin_color"          // Ten rengi
	UserAttributeBodyType           = "body_type"           // Vücut yapısı
	UserAttributeEthnicity          = "ethnicity"           // Etnik köken
	UserAttributeZodiac             = "zodiac_sign"         // Burç
	UserAttributeCircumcision       = "circumcision"        // Sünnet durumu kategorisi
	UserAttributePhysicalDisability = "physical_disability" // Fiziksel engel
	UserAttributeSmoking            = "smoking"             // Sigara kullanımı
	UserAttributeDrinking           = "drinking"            // Alkol kullanımı
	UserAttributeHeight             = "height"              // Boy
	UserAttributeWeight             = "weight"              // Kilo
	UserAttributeReligion           = "religion"            // Din
	UserAttributeEducation          = "education"           // Eğitim düzeyi
	UserAttributeRelationshipStatus = "relationship_status" // İlişki durumu
	UserAttributePets               = "pets"                // Evcil hayvan
	UserAttributePersonality        = "personality"         // Kişilik tipi
	UserAttributeKidsPreference     = "kids_preference"     // Çocuk tercihi
	UserAttributeDietary            = "dietary"             // Beslenme diyet
	UserAttributeHIVAIDS            = "hiv_aids_status"     // HIV / AIDS durumu
	UserAttributeBDSMInterest       = "bdsm_interest"
	UserAttributeBDSMRoles          = "bdsm_roles" // BDSM roller
	UserAttributeBDSMPlays          = "bdsm_plays" // BDSM oyun/aktivite

)

// UserAttributeOption = Attribute tipi / metadata
type Attribute struct {
	ID           uuid.UUID              `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Category     string                 `gorm:"type:varchar(50);index;not null" json:"category"` // hair_color, eye_color, body_type, skin_color, ethnicity
	DisplayOrder int                    `gorm:"not null" json:"display_order"`                   // 0,1,2...
	Name         shared.LocalizedString `gorm:"type:jsonb;not null" json:"name"`                 // Çoklu dil desteği
}

func (Attribute) TableName() string {
	return "attributes"
}

// UserAttribute = Kullanıcının seçtiği attribute
type UserAttribute struct {
	ID           uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"`
	CategoryType string    `gorm:"type:varchar(50);index;not null" json:"category_type"` // hair_color, eye_color, body_type, skin_color, ethnicity

	AttributeID uuid.UUID  `gorm:"type:uuid;index;not null" json:"attribute_id"`
	Attribute   *Attribute `gorm:"foreignKey:AttributeID;references:ID" json:"attribute,omitempty"`
	Notes       *string    `gorm:"type:text" json:"notes,omitempty"` // opsiyonel not
}

func (UserAttribute) TableName() string {
	return "user_attributes"
}
