package handlers

import (
	"context"
	"core/application/ports"
	usecases "core/application/usecases"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

type broadcastHandlerGateway struct {
	called bool
}

func (g *broadcastHandlerGateway) FetchTrending(context.Context, ports.BroadcastProvider, ports.BroadcastTrendingQuery) ([]byte, error) {
	g.called = true
	return []byte(`{}`), nil
}
func (g *broadcastHandlerGateway) CreateBroadcast(context.Context, string) ([]byte, error) {
	g.called = true
	return []byte(`{}`), nil
}
func (g *broadcastHandlerGateway) ViewBroadcast(context.Context, ports.BroadcastViewInput) ([]byte, error) {
	g.called = true
	return []byte(`{}`), nil
}
func (g *broadcastHandlerGateway) RequestGuestBroadcast(context.Context, ports.BroadcastGuestRequest) ([]byte, error) {
	g.called = true
	return []byte(`{}`), nil
}
func (g *broadcastHandlerGateway) LikeBroadcast(context.Context, ports.BroadcastLikeInput) ([]byte, error) {
	g.called = true
	return []byte(`{}`), nil
}

func TestHandleCreateBroadcastRejectsAnonymousCallerBeforeGateway(t *testing.T) {
	gateway := &broadcastHandlerGateway{}
	app := fiber.New()
	app.Post("/", HandleCreateBroadcast(usecases.NewBroadcastService(gateway)))

	response, err := app.Test(httptest.NewRequest("POST", "/", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusUnauthorized)
	}
	if gateway.called {
		t.Fatal("gateway called for anonymous request")
	}
}
