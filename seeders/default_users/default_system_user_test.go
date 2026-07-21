package default_users

import (
	"core/constants"
	"strings"
	"testing"
)

func TestDefaultSystemRoles(t *testing.T) {
	if role := defaultSystemRole(constants.SystemUserAdmin); role != constants.UserRoleAdmin {
		t.Fatalf("admin role = %q", role)
	}
	if role := defaultSystemRole(constants.SystemUserModerator); role != constants.UserRoleModerator {
		t.Fatalf("moderator role = %q", role)
	}
	if role := defaultSystemRole(constants.SystemUserSupport); role != constants.UserRoleUser {
		t.Fatalf("support role = %q", role)
	}
}

func TestGeneratedSystemPasswordIsArgon2id(t *testing.T) {
	password, err := generateSystemPassword()
	if err != nil {
		t.Fatalf("generateSystemPassword() error = %v", err)
	}
	if !strings.HasPrefix(password, "$argon2id$") {
		t.Fatalf("password is not Argon2id: %q", password)
	}
}
