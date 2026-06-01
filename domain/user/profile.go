package user

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidBirthDate    = errors.New("invalid birth date")
	ErrFutureBirthDate     = errors.New("birth date cannot be in the future")
	ErrInvalidLatitude     = errors.New("invalid latitude")
	ErrInvalidLongitude    = errors.New("invalid longitude")
	ErrInvalidPrivacyLevel = errors.New("invalid privacy level")
)

type PrivacyLevel string

const (
	PrivacyPublic        PrivacyLevel = "public"
	PrivacyFriendsOnly   PrivacyLevel = "friends_only"
	PrivacyFollowersOnly PrivacyLevel = "followers_only"
	PrivacyMutualsOnly   PrivacyLevel = "mutuals_only"
	PrivacyPrivate       PrivacyLevel = "private"
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

func NormalizeUsername(input string) string {
	return strings.ToLower(strings.TrimSpace(input))
}

func NormalizeDisplayName(input string) string {
	return strings.TrimSpace(input)
}

func ParsePrivacyLevel(input string) (PrivacyLevel, bool, error) {
	value := PrivacyLevel(strings.TrimSpace(input))
	if value == "" {
		return "", false, nil
	}
	if !value.IsValid() {
		return "", false, ErrInvalidPrivacyLevel
	}
	return value, true, nil
}

func (p PrivacyLevel) IsValid() bool {
	switch p {
	case PrivacyPublic,
		PrivacyFriendsOnly,
		PrivacyFollowersOnly,
		PrivacyMutualsOnly,
		PrivacyPrivate:
		return true
	default:
		return false
	}
}

func ParseBirthDate(input string, now time.Time) (*time.Time, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return nil, nil
	}

	birthDate, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, ErrInvalidBirthDate
	}
	if birthDate.After(now) {
		return nil, ErrFutureBirthDate
	}
	return &birthDate, nil
}

func NewCoordinates(lat, lon float64) (Coordinates, error) {
	if lat < -90 || lat > 90 {
		return Coordinates{}, ErrInvalidLatitude
	}
	if lon < -180 || lon > 180 {
		return Coordinates{}, ErrInvalidLongitude
	}
	return Coordinates{Latitude: lat, Longitude: lon}, nil
}
