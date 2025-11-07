package shared

import (
	"coolvibes/extensions"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	LocationOwnerPost  = "post"
	LocationOwnerEvent = "event"
	LocationOwnerUser  = "user"
)

// OwnerType: örn "post", "event", "user", "chat", ...
type Location struct {
	ID              uuid.UUID                `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ContentableID   uuid.UUID                `json:"contentable_id"`
	ContentableType string                   `json:"contentable_type"`
	CountryCode     *string                  `json:"country_code"` // Örn: "TR"
	Address         *string                  `gorm:"size:1024" json:"address,omitempty"`
	City            *string                  `gorm:"size:512" json:"city,omitempty"`
	Country         *string                  `gorm:"size:512" json:"country,omitempty"`
	Postal          *string                  `gorm:"size:128" json:"postal,omitempty"`
	Region          *string                  `json:"region,omitempty"`   // Örn: "Marmara"
	Timezone        *string                  `json:"timezone,omitempty"` // Örn: "Europe/Istanbul"
	Display         *string                  `json:"display"`            // "İstanbul, Türkiye"
	Latitude        *float64                 `gorm:"type:numeric(10,6)" json:"latitude,omitempty"`
	Longitude       *float64                 `gorm:"type:numeric(10,6)" json:"longitude,omitempty"`
	LocationPoint   *extensions.PostGISPoint `gorm:"type:geography(Point,4326)" json:"location_point,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

/*
	func (l *Location) Scan(value interface{}) error {
		if value == nil {
			return nil
		}
		bytes, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
		}
		return json.Unmarshal(bytes, l)
	}

	func (l Location) Value() (driver.Value, error) {
		return json.Marshal(l)
	}
*/
func (Location) TableName() string {
	return "locations"
}
