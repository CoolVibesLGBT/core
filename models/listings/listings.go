package listings

import (
	"coolvibes/models/media"
	"coolvibes/models/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ListingType string

const (
	// Ev ve konaklama
	ListingTypeRoommate  ListingType = "roommate"   // Ev arkadaşı
	ListingTypeRoomRent  ListingType = "room_rent"  // Kiralık oda/ev
	ListingTypePetSitter ListingType = "pet_sitter" // Evcil hayvan bakıcısı

	// İş ve eğitim
	ListingTypeJob        ListingType = "job"         // İş ilanı
	ListingTypeJobSeeking ListingType = "job_seeking" // İş arayan
	ListingTypeTutoring   ListingType = "tutoring"    // Özel ders veren
	ListingTypeFreelance  ListingType = "freelance"   // Serbest çalışma / freelance

	// Topluluk ve etkinlik
	ListingTypeEvent     ListingType = "event"     // Etkinlik / buluşma
	ListingTypeMeetup    ListingType = "meetup"    // Sosyal buluşmalar
	ListingTypeVolunteer ListingType = "volunteer" // Gönüllü işler

	// Diğer
	ListingTypeOther ListingType = "other" // Diğer ilanlar
)

type Listing struct {
	ID        uuid.UUID   `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	OwnerID   uuid.UUID   `gorm:"type:uuid;index;not null" json:"owner_id"`      // ilan sahibi
	OwnerType string      `gorm:"type:varchar(50);not null" json:"owner_type"`   // user, organization vb.
	Type      ListingType `gorm:"type:varchar(50);not null" json:"listing_type"` // ilan türü

	Title       string `gorm:"type:varchar(255);not null" json:"title"`
	Description string `gorm:"type:text" json:"description,omitempty"`

	Location *utils.Location `gorm:"embedded" json:"location,omitempty"` // opsiyonel location bilgisi

	Attributes map[string]string `gorm:"type:jsonb" json:"attributes,omitempty"`
	// generic alanlar: örneğin: oda sayısı, maaş, tecrübe yılı, vs.

	Media []*media.Media `gorm:"polymorphic:Owner;polymorphicValue:listing;constraint:OnDelete:CASCADE" json:"media,omitempty"`

	IsActive  bool           `gorm:"default:true" json:"is_active"`
	ExpiresAt *time.Time     `json:"expires_at,omitempty"` // ilan geçerlilik süresi
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
