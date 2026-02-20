package mcp

import "errors"

type Router struct {
	registry *Registry
}

func NewRouter(reg *Registry) *Router {
	return &Router{registry: reg}
}

func (r *Router) Route(msg Envelope) (Envelope, error) {

	if msg.Target == "" {
		return Envelope{}, errors.New("target required")
	}

	agent, ok := r.registry.Get(msg.Target)
	if !ok {
		return Envelope{}, errors.New("target not found")
	}

	return agent.Handle(msg)
}
