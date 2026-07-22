package handlers

import (
	"core/application/types"
	"core/constants"
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

func TestParseFiltersRejectsInvalidSpatialValues(t *testing.T) {
	tests := []struct {
		name string
		form url.Values
	}{
		{name: "latitude without longitude", form: url.Values{"latitude": {"41"}}},
		{name: "longitude without latitude", form: url.Values{"longitude": {"29"}}},
		{name: "latitude out of range", form: url.Values{"latitude": {"91"}, "longitude": {"29"}}},
		{name: "longitude out of range", form: url.Values{"latitude": {"41"}, "longitude": {"181"}}},
		{name: "NaN coordinate", form: url.Values{"latitude": {"NaN"}, "longitude": {"29"}}},
		{name: "infinite coordinate", form: url.Values{"latitude": {"41"}, "longitude": {"+Inf"}}},
		{name: "negative cursor distance", form: url.Values{"distance": {"-1"}}},
		{name: "NaN cursor distance", form: url.Values{"distance": {"NaN"}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/", func(c fiber.Ctx) error {
				if _, err := ParseFilters(c, nil); err == nil {
					return c.SendStatus(fiber.StatusNoContent)
				}
				return c.SendStatus(fiber.StatusBadRequest)
			})

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.form.Encode()))
			req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationForm)
			response, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			defer func() { _ = response.Body.Close() }()
			if response.StatusCode != fiber.StatusBadRequest {
				t.Fatalf("status = %d, want 400", response.StatusCode)
			}
		})
	}
}
