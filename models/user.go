package models

import (
	"core/constants"
	"core/models/media"
	"core/models/utils"
	"errors"
	"math/big"

	"encoding/hex"
	"encoding/json"
	"strconv"

	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Story struct {
	ID     uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User   *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`

	MediaID uuid.UUID    `gorm:"type:uuid;not null" json:"media_id"`
	Media   *media.Media `gorm:"constraint:OnDelete:CASCADE;foreignKey:MediaID;references:ID" json:"media"`

	Caption    *string   `gorm:"type:text" json:"caption,omitempty"`
	ExpiresAt  time.Time `gorm:"index" json:"expires_at"`
	IsExpired  bool      `gorm:"default:false" json:"is_expired"`
	IsArchived bool      `gorm:"default:false" json:"is_archived"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type TravelData struct {
	VisitedCountries pq.StringArray `gorm:"type:text[]" json:"visited_countries"`
	TravelFrequency  string         `json:"travel_frequency"`                   // örn: "aylık"
	FavoritePlaces   pq.StringArray `gorm:"type:text[]" json:"favorite_places"` // opsiyonel
}

// Ziyaret Edilen Ülkeler
type CountryVisit struct {
	ISOCode   string    `json:"iso_code"`             // Örn: "FR"
	Name      string    `json:"name"`                 // Örn: "France"
	VisitedAt time.Time `json:"visited_at,omitempty"` // İsteğe bağlı
	Notes     string    `json:"notes,omitempty"`
}

// Sevilen Şehirler
type FavoriteCity struct {
	City      string    `json:"city"`                 // Örn: "Tokyo"
	Country   string    `json:"country"`              // Örn: "Japan"
	ISOCode   string    `json:"iso_code"`             // Örn: "JP"
	Reason    string    `json:"reason,omitempty"`     // Neden favori?
	LastVisit time.Time `json:"last_visit,omitempty"` // Son ziyaret tarihi
}

// Seyahat Planı
type TravelPlan struct {
	City        string                  `json:"city"`     // Örn: "Barcelona"
	Country     string                  `json:"country"`  // Örn: "Spain"
	ISOCode     string                  `json:"iso_code"` // Örn: "ES"
	StartDate   time.Time               `json:"start_date"`
	EndDate     time.Time               `json:"end_date"`
	Purpose     constants.TravelPurpose `json:"purpose,omitempty"` // Enum: vacation, work, etc.
	WithFriends bool                    `json:"with_friends"`      // Yalnız mı gidiyor?
	Notes       string                  `json:"notes,omitempty"`
	IsPublic    bool                    `json:"is_public"` // Profilde gözükebilir mi?
}

type Subscription struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

type User struct {
	ID               uuid.UUID              `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	PublicID         int64                  `gorm:"uniqueIndex;not null" json:"public_id"`
	SocketID         *string                `json:"socket_id,omitempty"`
	Domain           DomainKind             `gorm:"size:50;not null;index;default:'coolvibes'" json:"domain"`
	UserName         string                 `json:"username"`
	DisplayName      string                 `json:"displayname"`
	Email            string                 `json:"-"`
	Password         string                 `json:"-"`
	Bio              *utils.LocalizedString `gorm:"type:jsonb" json:"bio,omitempty"`
	DateOfBirth      *time.Time             `json:"date_of_birth,omitempty"`
	Balance          decimal.Decimal        `gorm:"type:numeric(38,18);default:0" json:"balance"`
	PrivacyLevel     constants.PrivacyLevel `gorm:"type:varchar(20);default:'public'" json:"privacy_level"`
	PreferencesFlags string                 `gorm:"column:preferences_flags" json:"preferences_flags"`
	UserRole         constants.UserRole     `json:"user_role"`
	IsOnline         bool                   `gorm:"default:false" json:"is_online"`
	IsPremium        bool                   `gorm:"default:false" json:"is_premium"`
	IsBot            bool                   `gorm:"default:false" json:"-"`
	IsActive         bool                   `gorm:"default:false" json:"is_active"`
	IsLive           bool                   `gorm:"default:false" json:"is_live"`
	IsProcessed      bool                   `gorm:"default:false" json:"is_processed"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	LastOnline       *time.Time             `json:"last_online,omitempty"`
	Location         *utils.Location        `gorm:"polymorphic:Contentable;polymorphicValue:user;constraint:OnDelete:CASCADE" json:"location,omitempty"`
	DefaultLanguage  string                 `gorm:"type:varchar(8);default:'en'" json:"default_language"`
	AvatarID         *uuid.UUID             `json:"avatar_id,omitempty"`
	Avatar           *media.Media           `gorm:"constraint:OnDelete:SET NULL;foreignKey:AvatarID;references:ID" json:"avatar,omitempty"`
	CoverID          *uuid.UUID             `json:"cover_id,omitempty"`
	Cover            *media.Media           `gorm:"constraint:OnDelete:SET NULL;foreignKey:CoverID;references:ID" json:"cover,omitempty"`
	Stories          []Story                `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"stories,omitempty"`
	Languages        pq.StringArray         `gorm:"type:text[]" json:"languages"`
	Hobbies          pq.StringArray         `gorm:"type:text[]" json:"hobbies,omitempty"`
	MoviesGenres     pq.StringArray         `gorm:"type:text[]" json:"movies_genres,omitempty"`
	TVShowsGenres    pq.StringArray         `gorm:"type:text[]" json:"tv_shows_genres,omitempty"`
	TheaterGenres    pq.StringArray         `gorm:"type:text[]" json:"theater_genres,omitempty"`
	CinemaGenres     pq.StringArray         `gorm:"type:text[]" json:"cinema_genres,omitempty"`
	ArtInterests     pq.StringArray         `gorm:"type:text[]" json:"art_interests,omitempty"`
	Entertainment    pq.StringArray         `gorm:"type:text[]" json:"entertainment,omitempty"`
	Travel           TravelData             `gorm:"embedded;embeddedPrefix:travel_" json:"travel"`
	Engagements      *Engagement            `gorm:"polymorphic:Contentable;polymorphicValue:user;constraint:OnDelete:CASCADE" json:"engagements,omitempty"`
	Wallet           Wallet                 `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"wallet,omitempty"`
	Subscriptions    datatypes.JSON         `gorm:"type:jsonb" json:"-"`
	BroadcastInfo    datatypes.JSON         `gorm:"type:jsonb" json:"broadcast_info"`
	DeletedAt        gorm.DeletedAt         `gorm:"index" json:"deleted_at,omitempty"`
	jwt.StandardClaims
}

func (u User) MarshalJSON() ([]byte, error) {
	type Alias User // recursive çağrıyı önlemek için alias
	aux := struct {
		PublicID string `json:"public_id"`
		Alias
	}{
		PublicID: strconv.FormatInt(u.PublicID, 10),
		Alias:    (Alias)(u),
	}

	return json.Marshal(aux)
}
func (User) TableName() string {
	return "users"
}

func (TravelPlan) TableName() string {
	return "user_travel_plans"
}

func (FavoriteCity) TableName() string {
	return "user_favorite_cities"
}

func (CountryVisit) TableName() string {
	return "user_country_visits"
}

func (u *User) SetPreference(bitIndex int) error {
	if bitIndex < 0 {
		return errors.New("bitIndex must be non-negative")
	}

	flags := big.NewInt(0)
	if u.PreferencesFlags != "" {
		bytes, err := hex.DecodeString(u.PreferencesFlags)
		if err != nil {
			return err
		}
		flags.SetBytes(bytes)
	}
	flags.SetBit(flags, bitIndex, 1)

	u.PreferencesFlags = hex.EncodeToString(flags.Bytes())
	return nil
}

func (u *User) IsPreferenceSet(bitIndex int) (bool, error) {
	if bitIndex < 0 {
		return false, errors.New("bitIndex must be non-negative")
	}

	flags := big.NewInt(0)
	if u.PreferencesFlags != "" {
		bytes, err := hex.DecodeString(u.PreferencesFlags)
		if err != nil {
			return false, err
		}
		flags.SetBytes(bytes)
	}

	return flags.Bit(bitIndex) == 1, nil
}

func (u *User) UnsetPreference(bitIndex int) error {
	if bitIndex < 0 {
		return errors.New("bitIndex must be non-negative")
	}
	flags := big.NewInt(0)
	if u.PreferencesFlags != "" {
		bytes, err := hex.DecodeString(u.PreferencesFlags)
		if err != nil {
			return err
		}
		flags.SetBytes(bytes)
	}
	flags.SetBit(flags, bitIndex, 0)

	u.PreferencesFlags = hex.EncodeToString(flags.Bytes())
	return nil
}
