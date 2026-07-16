package test

import (
	app "core/infrastructure/bootstrap"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func NewTestApp(t *testing.T) *app.App {
	t.Helper()

	wd, _ := os.Getwd()
	println("Working dir:", wd)

	_ = godotenv.Load("../.env")

	// Test env zorla
	err := os.Setenv("APP_ENV", "test")
	if err != nil {
		t.Fatal(err)
	}

	application, err := app.InitializeApp()
	if err != nil {
		t.Fatal(err)
	}

	// 🔥 Her test başında full temizle
	err = application.DB.Exec(`
		TRUNCATE pillars, clusters
		RESTART IDENTITY CASCADE
	`).Error
	if err != nil {
		t.Fatal(err)
	}

	return application
}
