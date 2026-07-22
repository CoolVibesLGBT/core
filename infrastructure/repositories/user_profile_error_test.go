package repositories

import (
	domainuser "core/domain/user"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestTranslateUserProfileUpdateErrorMapsIdentityRaces(t *testing.T) {
	for constraint, expected := range map[string]error{
		"uidx_users_active_user_name_ci": domainuser.ErrUsernameAlreadyExists,
		"uidx_users_active_email_ci":     domainuser.ErrEmailAlreadyExists,
	} {
		databaseError := &pgconn.PgError{Code: "23505", ConstraintName: constraint}
		if actual := translateUserProfileUpdateError(databaseError); !errors.Is(actual, expected) {
			t.Fatalf("translate constraint %s = %v; want %v", constraint, actual, expected)
		}
	}

	other := errors.New("other persistence failure")
	if actual := translateUserProfileUpdateError(other); !errors.Is(actual, other) {
		t.Fatalf("unrelated error = %v; want original", actual)
	}
}
