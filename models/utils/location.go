package utils

import (
	"core/extensions"
	"math"
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
	ContentableID   uuid.UUID                `gorm:"type:uuid;not null;index" json:"contentable_id"`
	ContentableType string                   `gorm:"type:varchar(50);not null;index" json:"contentable_type"`
	CountryCode     *string                  `json:"country_code"` // Örn: "TR"
	Address         *string                  `gorm:"size:1024" json:"address,omitempty"`
	City            *string                  `gorm:"size:512" json:"city,omitempty"`
	Country         *string                  `gorm:"size:512" json:"country,omitempty"`
	Postal          *string                  `gorm:"size:128" json:"postal,omitempty"`
	Region          *string                  `json:"region,omitempty"` // Örn: "Marmara"
	Postcode        *string                  `gorm:"size:64" json:"postcode,omitempty"`
	ZipCode         *string                  `json:"zip_code,omitempty"` // Örn: "34000"
	Province        *string                  `json:"province,omitempty"` // Örn: "İstanbul"
	Town            *string                  `json:"town,omitempty"`     // Örn: "Kadıköy"
	Timezone        *string                  `json:"timezone,omitempty"` // Örn: "Europe/Istanbul"
	Display         *string                  `json:"display"`            // "İstanbul, Türkiye"
	Latitude        *float64                 `gorm:"type:numeric(10,6)" json:"latitude,omitempty"`
	Longitude       *float64                 `gorm:"type:numeric(10,6)" json:"longitude,omitempty"`
	LocationPoint   *extensions.PostGISPoint `gorm:"type:geography(Point,4326)" json:"location_point,omitempty"`
	IPAddress       *string                  `json:"-"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
	DeletedAt       gorm.DeletedAt           `gorm:"index" json:"deleted_at,omitempty"`
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

func NewLocationPoint(lat, lon float64) *extensions.PostGISPoint {
	return &extensions.PostGISPoint{
		Lat: lat,
		Lng: lon,
	}
}

func (l *Location) DistanceTo(lat, lon float64) float64 {
	const R = 6371000 // Dünya yarıçapı metre cinsinden
	lat1 := *l.Latitude * math.Pi / 180
	lon1 := *l.Longitude * math.Pi / 180
	lat2 := lat * math.Pi / 180
	lon2 := lon * math.Pi / 180

	dLat := lat2 - lat1
	dLon := lon2 - lon1

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c // metre cinsinden mesafe
}
