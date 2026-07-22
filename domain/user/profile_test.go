package user

import (
	"errors"
	"testing"
)

func TestNormalizeProfileContactFields(t *testing.T) {
	email, present, err := NormalizeEmail("  Alice@Example.COM ")
	if err != nil || !present || email != "alice@example.com" {
		t.Fatalf("NormalizeEmail() = %q, %v, %v", email, present, err)
	}
	if _, _, err := NormalizeEmail("not-an-email"); !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("NormalizeEmail(invalid) error = %v, want %v", err, ErrInvalidEmail)
	}

	website, present, err := NormalizeWebsite(" example.com/profile ")
	if err != nil || !present || website != "https://example.com/profile" {
		t.Fatalf("NormalizeWebsite() = %q, %v, %v", website, present, err)
	}
	for _, value := range []string{"ftp://example.com", "https://user:secret@example.com", "://broken"} {
		if _, _, err := NormalizeWebsite(value); !errors.Is(err, ErrInvalidWebsite) {
			t.Fatalf("NormalizeWebsite(%q) error = %v, want %v", value, err, ErrInvalidWebsite)
		}
	}
}

func TestValidatePasswordChangeRequiresCurrentPasswordAndMatchingConfirmation(t *testing.T) {
	if changed, err := ValidatePasswordChange("", "", ""); err != nil || changed {
		t.Fatalf("ValidatePasswordChange(empty) = %v, %v", changed, err)
	}
	if _, err := ValidatePasswordChange("", "new-secret", "new-secret"); !errors.Is(err, ErrCurrentPasswordRequired) {
		t.Fatalf("missing current password error = %v", err)
	}
	if _, err := ValidatePasswordChange("current", "new-secret", "different"); !errors.Is(err, ErrPasswordConfirmationMismatch) {
		t.Fatalf("mismatched confirmation error = %v", err)
	}
	if changed, err := ValidatePasswordChange("current", "new-secret", "new-secret"); err != nil || !changed {
		t.Fatalf("valid password change = %v, %v", changed, err)
	}
}
