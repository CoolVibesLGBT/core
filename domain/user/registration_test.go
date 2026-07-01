package user

import (
	"testing"
	"time"
)

func TestNewRegistrationNormalizesAndValidates(t *testing.T) {
	registration, err := NewRegistration(RegistrationInput{
		Name:     "  Ersan  ",
		Nickname: "CoolUser",
		Password: "Secret",
		Domain:   "www.coolvibes.app/",
		Email:    "USER@Example.COM",
	})
	if err != nil {
		t.Fatalf("NewRegistration() error = %v", err)
	}

	if registration.Name != "Ersan" {
		t.Fatalf("Name = %q, want %q", registration.Name, "Ersan")
	}
	if registration.Nickname != "cooluser" {
		t.Fatalf("Nickname = %q, want %q", registration.Nickname, "cooluser")
	}
	if registration.Password != "Secret" {
		t.Fatalf("Password = %q, want %q", registration.Password, "Secret")
	}
	if registration.Domain != CoolVibes {
		t.Fatalf("Domain = %q, want %q", registration.Domain, CoolVibes)
	}
	if registration.Email != "user@example.com" {
		t.Fatalf("Email = %q, want %q", registration.Email, "user@example.com")
	}
}

func TestGetDomainKindTreatsLocalhostAsCoolVibes(t *testing.T) {
	for _, host := range []string{"localhost", "localhost:3000", "127.0.0.1:5173", "[::1]:3000"} {
		if got := GetDomainKind(host); got != CoolVibes {
			t.Fatalf("GetDomainKind(%q) = %q, want %q", host, got, CoolVibes)
		}
	}
}

func TestCredentialsPreservePasswordCase(t *testing.T) {
	credentials := NewCredentials("  CoolUser  ", "SecretPASS")
	if credentials.UserName != "cooluser" {
		t.Fatalf("UserName = %q, want %q", credentials.UserName, "cooluser")
	}
	if credentials.Password != "SecretPASS" {
		t.Fatalf("Password = %q, want %q", credentials.Password, "SecretPASS")
	}
}

func TestProfileValueObjects(t *testing.T) {
	if privacy, ok, err := ParsePrivacyLevel("private"); err != nil || !ok || privacy != PrivacyPrivate {
		t.Fatalf("ParsePrivacyLevel() = %q, %v, %v", privacy, ok, err)
	}
	if _, _, err := ParsePrivacyLevel("invalid"); err != ErrInvalidPrivacyLevel {
		t.Fatalf("invalid privacy error = %v, want %v", err, ErrInvalidPrivacyLevel)
	}

	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	birthDate, err := ParseBirthDate("2000-01-02", now)
	if err != nil {
		t.Fatalf("ParseBirthDate() error = %v", err)
	}
	if birthDate == nil || birthDate.Format("2006-01-02") != "2000-01-02" {
		t.Fatalf("BirthDate = %v", birthDate)
	}
	if _, err := ParseBirthDate("2999-01-02", now); err != ErrFutureBirthDate {
		t.Fatalf("future birth date error = %v, want %v", err, ErrFutureBirthDate)
	}

	if _, err := NewCoordinates(91, 0); err != ErrInvalidLatitude {
		t.Fatalf("invalid latitude error = %v, want %v", err, ErrInvalidLatitude)
	}
	if _, err := NewCoordinates(0, 181); err != ErrInvalidLongitude {
		t.Fatalf("invalid longitude error = %v, want %v", err, ErrInvalidLongitude)
	}
}

func TestNewRegistrationRejectsInvalidValueObjects(t *testing.T) {
	if _, err := NewRegistration(RegistrationInput{Domain: "coolvibes.app", Email: "not-an-email"}); err != ErrInvalidEmail {
		t.Fatalf("invalid email error = %v, want %v", err, ErrInvalidEmail)
	}

	if _, err := NewRegistration(RegistrationInput{Domain: "unknown.example"}); err != ErrInvalidDomain {
		t.Fatalf("invalid domain error = %v, want %v", err, ErrInvalidDomain)
	}
}

func TestPreferenceFlags(t *testing.T) {
	flags, err := PreferenceFlags("").Set(3)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	ok, err := flags.IsSet(3)
	if err != nil {
		t.Fatalf("IsSet() error = %v", err)
	}
	if !ok {
		t.Fatal("bit 3 was not set")
	}

	flags, err = flags.Unset(3)
	if err != nil {
		t.Fatalf("Unset() error = %v", err)
	}

	ok, err = flags.IsSet(3)
	if err != nil {
		t.Fatalf("IsSet() error = %v", err)
	}
	if ok {
		t.Fatal("bit 3 is still set")
	}
}

func TestEnsureDifferentPublicUsers(t *testing.T) {
	if err := EnsureDifferentPublicUsers(1, 1, InteractionBlock); err == nil {
		t.Fatal("expected self interaction error")
	}

	if err := EnsureDifferentPublicUsers(1, 2, InteractionBlock); err != nil {
		t.Fatalf("EnsureDifferentPublicUsers() error = %v", err)
	}
}

func TestInteractionEngagementPair(t *testing.T) {
	pair, err := InteractionEngagementPair(InteractionLike, true)
	if err != nil {
		t.Fatalf("InteractionEngagementPair() error = %v", err)
	}
	if pair.Given != EngagementLikeGiven || pair.Received != EngagementLikeReceived {
		t.Fatalf("like pair = %+v", pair)
	}

	pair, err = InteractionEngagementPair(InteractionLike, false)
	if err != nil {
		t.Fatalf("InteractionEngagementPair() error = %v", err)
	}
	if pair.Given != EngagementDislikeGiven || pair.Received != EngagementDislikeReceived {
		t.Fatalf("dislike pair = %+v", pair)
	}
}
