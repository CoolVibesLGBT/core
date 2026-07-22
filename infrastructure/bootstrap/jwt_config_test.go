package bootstrap

import (
	"errors"
	"testing"

	"core/helpers"
)

func TestInitializeAppFailsBeforeExternalDependenciesWhenUserJWTSecretIsMissing(t *testing.T) {
	t.Setenv("USER_JWT_SECRET", "")

	application, err := InitializeApp()
	if application != nil {
		t.Fatal("InitializeApp() returned an application with missing JWT secret")
	}
	if !errors.Is(err, helpers.ErrUserJWTSecretRequired) {
		t.Fatalf("InitializeApp() error = %v, want ErrUserJWTSecretRequired", err)
	}
}
