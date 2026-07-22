package router

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestActionRouterUsesHeaderWithoutFormAction(t *testing.T) {
	actionRouter := NewActionRouter()
	actionRouter.Register("upload.test", func(c fiber.Ctx) error {
		return c.SendString("routed")
	})
	app := fiber.New()
	app.Post("/", actionRouter.Resolve)

	request := httptest.NewRequest(fiber.MethodPost, "/", strings.NewReader("body does not contain an action field"))
	request.Header.Set("X-Action", "upload.test")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
}
