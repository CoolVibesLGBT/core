package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestParseFiltersPreservesExplicitSmallLimit(t *testing.T) {
	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		filter, err := ParseFilters(c, nil)
		if err != nil {
			return err
		}
		if filter.Limit != 1 {
			t.Fatalf("expected limit 1, got %d", filter.Limit)
		}
		return nil
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("limit=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}
