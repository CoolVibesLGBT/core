package user

import (
	"errors"
	"math"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidBirthDate             = errors.New("invalid birth date")
	ErrFutureBirthDate              = errors.New("birth date cannot be in the future")
	ErrInvalidLatitude              = errors.New("invalid latitude")
	ErrInvalidLongitude             = errors.New("invalid longitude")
	ErrInvalidPrivacyLevel          = errors.New("invalid privacy level")
	ErrInvalidWebsite               = errors.New("invalid website")
	ErrCurrentPasswordRequired      = errors.New("current password is required")
	ErrPasswordConfirmationMismatch = errors.New("new password confirmation does not match")
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

func NormalizeEmail(input string) (string, bool, error) {
	value := strings.ToLower(strings.TrimSpace(input))
	if value == "" {
		return "", false, nil
	}
	if !IsValidEmail(value) {
		return "", false, ErrInvalidEmail
	}
	return value, true, nil
}

func NormalizeWebsite(input string) (string, bool, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return "", false, nil
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
		return "", false, ErrInvalidWebsite
	}
	return parsed.String(), true, nil
}

func ValidatePasswordChange(currentPassword, newPassword, confirmation string) (bool, error) {
	hasNewPassword := newPassword != "" || confirmation != ""
	if !hasNewPassword {
		return false, nil
	}
	if currentPassword == "" {
		return false, ErrCurrentPasswordRequired
	}
	if newPassword == "" || newPassword != confirmation {
		return false, ErrPasswordConfirmationMismatch
	}
	return true, nil
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
	if math.IsNaN(lat) || math.IsInf(lat, 0) || lat < -90 || lat > 90 {
		return Coordinates{}, ErrInvalidLatitude
	}
	if math.IsNaN(lon) || math.IsInf(lon, 0) || lon < -180 || lon > 180 {
		return Coordinates{}, ErrInvalidLongitude
	}
	return Coordinates{Latitude: lat, Longitude: lon}, nil
}
