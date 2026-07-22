package test

import (
	app "core/infrastructure/bootstrap"
	"os"
	"strings"
	"testing"
)

func NewTestApp(t *testing.T) *app.App {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for destructive taxonomy integration tests")
	}
	t.Setenv("DATABASE_URL", dsn)
	t.Setenv("APP_ENV", "test")
	t.Setenv("USER_JWT_SECRET", "test-only-user-jwt-secret-at-least-32-bytes")

	application, err := app.InitializeApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := application.Close(); err != nil {
			t.Errorf("close test application: %v", err)
		}
	})

	// This destructive reset is permitted only on the explicitly supplied
	// integration-test database above; a developer's local .env is never read.
	err = application.DB.Exec(`
		TRUNCATE pillars, clusters
		RESTART IDENTITY CASCADE
	`).Error
	if err != nil {
		t.Fatal(err)
	}

	return application
}
