package router

import (
	"coolvibes/middleware"
	"net/http"

	"gorm.io/gorm"
)

type Route struct {
	Handler     http.HandlerFunc
	Middlewares []middleware.Middleware
}

type ActionRouter struct {
	routes       map[string]Route
	defaultRoute http.HandlerFunc
	db           *gorm.DB
}

func NewActionRouter(db *gorm.DB) *ActionRouter {
	return &ActionRouter{
		routes: make(map[string]Route),
		db:     db,
	}
}

// Register
func (ar *ActionRouter) Register(action string, handler http.HandlerFunc, mws ...middleware.Middleware) {
	ar.routes[action] = Route{
		Handler:     handler,
		Middlewares: mws,
	}
}

// Resolve
func (ar *ActionRouter) Resolve(w http.ResponseWriter, r *http.Request) {
	action := r.FormValue("action")
	if action == "" {
		action = r.URL.Query().Get("action")
	}

	route, ok := ar.routes[action]
	if !ok {
		if ar.defaultRoute != nil {
			ar.defaultRoute(w, r)
			return
		}
		http.Error(w, "Unknown action", http.StatusBadRequest)
		return
	}

	// Middleware zincirini uygula
	handler := route.Handler
	for i := len(route.Middlewares) - 1; i >= 0; i-- {
		handler = route.Middlewares[i](handler)
	}

	handler(w, r)
}
func (ar *ActionRouter) GetHandler(action string) (Route, bool) {
	route, ok := ar.routes[action]
	return route, ok
}
