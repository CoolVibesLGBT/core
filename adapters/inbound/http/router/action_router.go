package router

import (
	"core/adapters/inbound/http/middleware"

	"github.com/gofiber/fiber/v3"
)

type Route struct {
	Handler     fiber.Handler
	Middlewares []middleware.Middleware
}

type ActionRouter struct {
	routes       map[string]Route
	defaultRoute fiber.Handler
}

func NewActionRouter() *ActionRouter {
	return &ActionRouter{
		routes: make(map[string]Route),
	}
}

// Register
func (ar *ActionRouter) Register(action string, handler fiber.Handler, mws ...middleware.Middleware) {
	ar.routes[action] = Route{
		Handler:     handler,
		Middlewares: mws,
	}
}

// Resolve
func (ar *ActionRouter) Resolve(c fiber.Ctx) error {
	// Headers and query parameters let upload clients select an action before
	// Fiber consumes a potentially large streamed multipart body. Form action
	// remains a backward-compatible fallback for existing clients.
	action := c.Get("X-Action")
	if action == "" {
		action = c.Query("action")
	}
	if action == "" {
		action = c.FormValue("action")
	}

	route, ok := ar.routes[action]
	if !ok {
		if ar.defaultRoute != nil {
			return ar.defaultRoute(c)
		}
		return c.Status(fiber.StatusBadRequest).SendString("Unknown action")
	}

	// Middleware zincirini uygula
	handler := route.Handler
	for i := len(route.Middlewares) - 1; i >= 0; i-- {
		handler = route.Middlewares[i](handler)
	}

	return handler(c)
}
func (ar *ActionRouter) GetHandler(action string) (Route, bool) {
	route, ok := ar.routes[action]
	return route, ok
}
