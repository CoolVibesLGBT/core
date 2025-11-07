package media

import (
	"coolvibes/models/shared"
	"time"

	"github.com/google/uuid"
)

type Media struct {
	ID       uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	PublicID int64     `gorm:"uniqueIndex;not null" json:"public_id"` //snowflake

	FileID    uuid.UUID `gorm:"type:uuid;not null" json:"file_id"`  // FileMetadata referansı
	OwnerID   uuid.UUID `gorm:"type:uuid;not null" json:"owner_id"` // Kullanıcı, post, blog, chat ID
	OwnerType OwnerType `gorm:"type:varchar(20);not null" json:"owner_type"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Role      MediaRole `gorm:"type:varchar(20);not null" json:"role"` // profile, cover, post, chat_image...
	IsPublic  bool      `gorm:"default:true" json:"is_public"`         // Herkes görebilir mi?

	File shared.FileMetadata `gorm:"foreignKey:FileID;references:ID;constraint:OnDelete:CASCADE" json:"file"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Media) TableName() string {
	return "medias"
}
