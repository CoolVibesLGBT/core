package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UserProfileWriter is the command-side port for the profile aggregate. It is
// deliberately separate from profile/public query ports.
type UserProfileWriter interface {
	UpdateUserProfile(ctx context.Context, update UserProfileUpdate) error
}

// UserProfileUpdate is the atomic persistence command for editable profile
// fields. Nil pointers mean "leave the stored value unchanged". Keeping the
// command narrow prevents a lightweight/session User projection from
// overwriting credentials or unrelated account state.
type UserProfileUpdate struct {
	UserID       uuid.UUID
	UserName     *string
	DisplayName  *string
	Email        *string
	PasswordHash *string
	Website      *string
	Bio          map[string]string
	DateOfBirth  *time.Time
	PrivacyLevel *string
	Location     *UserProfileLocationUpdate
}

// UserProfileLocationUpdate carries validated coordinates and profile
// location metadata without leaking the GORM persistence model into the
// application command boundary.
type UserProfileLocationUpdate struct {
	CountryCode string
	Address     string
	City        string
	Country     string
	Region      string
	Timezone    string
	Display     string
	Latitude    float64
	Longitude   float64
}
