package media

import (
	"core/models/utils"
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type Media struct {
	ID       uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	PublicID int64     `gorm:"uniqueIndex;not null" json:"public_id"` //snowflake

	FileID              uuid.UUID        `gorm:"type:uuid;not null" json:"file_id"`  // FileMetadata referansı
	OwnerID             uuid.UUID        `gorm:"type:uuid;not null" json:"owner_id"` // Kullanıcı, post, blog, chat ID
	OwnerType           OwnerType        `gorm:"type:varchar(20);not null" json:"owner_type"`
	UserID              uuid.UUID        `gorm:"type:uuid;not null;index" json:"user_id"`
	Role                MediaRole        `gorm:"type:varchar(20);not null" json:"role"`   // profile, cover, post, chat_image...
	IsPublic            bool             `gorm:"not null;default:false" json:"is_public"` // Herkes görebilir mi?
	ProcessingStatus    ProcessingStatus `gorm:"type:varchar(20);not null;default:'ready';index:idx_medias_processing_status_created_at,priority:1" json:"processing_status"`
	ProcessingError     *string          `gorm:"type:text" json:"processing_error,omitempty"`
	ProcessingAttempts  int              `gorm:"not null;default:0" json:"processing_attempts"`
	ProcessingStartedAt *time.Time       `json:"processing_started_at,omitempty"`
	ProcessedAt         *time.Time       `json:"processed_at,omitempty"`

	File utils.FileMetadata `gorm:"foreignKey:FileID;references:ID;constraint:OnDelete:CASCADE" json:"file"`

	CreatedAt time.Time `gorm:"index:idx_medias_processing_status_created_at,priority:2" json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Media) TableName() string {
	return "medias"
}

func (u Media) MarshalJSON() ([]byte, error) {
	type Alias Media // recursive çağrıyı önlemek için alias
	aux := struct {
		PublicID string `json:"public_id"`
		Alias
	}{
		PublicID: strconv.FormatInt(u.PublicID, 10),
		Alias:    (Alias)(u),
	}

	return json.Marshal(aux)
}
