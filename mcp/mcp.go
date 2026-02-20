package mcp

type MCPServer struct {
	registry *Registry
	router   *Router
}

func NewMCPServer() *MCPServer {
	registry := NewRegistry()
	router := NewRouter(registry)

	return &MCPServer{
		registry: registry,
		router:   router,
	}
}

func (r *MCPServer) Registry() *Registry {
	return r.registry
}

func (r *MCPServer) Router() *Router {
	return r.router
}
