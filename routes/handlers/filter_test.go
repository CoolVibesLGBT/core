package handlers

import (
	"core/constants"
	"core/types"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/paginate"
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

func TestParseFiltersReadsFiberCursor(t *testing.T) {
	app := fiber.New()
	app.Use(paginate.New(paginate.Config{
		DefaultLimit: constants.DEFAULT_LIMIT,
		MaxLimit:     constants.MAXIMUM_LIMIT,
	}))
	app.Post("/", func(c fiber.Ctx) error {
		filter, err := ParseFilters(c, nil)
		if err != nil {
			return err
		}
		if filter.Limit != 7 {
			t.Fatalf("expected fiber limit 7, got %d", filter.Limit)
		}
		if filter.Cursor == nil || *filter.Cursor != 1001 {
			t.Fatalf("expected cursor 1001, got %#v", filter.Cursor)
		}
		if filter.Distance == nil || *filter.Distance != 42.5 {
			t.Fatalf("expected distance 42.5, got %#v", filter.Distance)
		}
		return nil
	})

	distance := 42.5
	cursor, err := types.NewPublicIDDistanceCursor(1001, &distance)
	if err != nil {
		t.Fatalf("NewPublicIDDistanceCursor() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/?limit=7&cursor="+url.QueryEscape(*cursor), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}
