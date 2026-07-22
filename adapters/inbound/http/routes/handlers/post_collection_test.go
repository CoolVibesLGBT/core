package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestUnsupportedPostCollectionsDoNotReturnUnrelatedMedia(t *testing.T) {
	app := fiber.New()
	app.Post("/", HandlePostCollectionNotImplemented())
	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != fiber.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusNotImplemented)
	}
}
