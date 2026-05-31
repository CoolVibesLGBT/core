package user

import "testing"

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
	if registration.Password != "secret" {
		t.Fatalf("Password = %q, want %q", registration.Password, "secret")
	}
	if registration.Domain != CoolVibes {
		t.Fatalf("Domain = %q, want %q", registration.Domain, CoolVibes)
	}
	if registration.Email != "user@example.com" {
		t.Fatalf("Email = %q, want %q", registration.Email, "user@example.com")
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
