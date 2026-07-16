package mcp

import (
	"context"
	"errors"
	"strings"
)

type Router struct {
	registry *Registry
}

func NewRouter(reg *Registry) *Router {
	return &Router{registry: reg}
}

func (r *Router) Call(ctx context.Context, req CallToolRequest) (CallToolResult, error) {
	req.Tool = strings.TrimSpace(req.Tool)
	if req.Tool == "" {
		return CallToolResult{}, errors.New("tool required")
	}

	tool, ok := r.registry.Get(req.Tool)
	if !ok {
		return CallToolResult{}, errors.New("tool not found")
	}

	result, err := tool.Call(ctx, req)
	if err != nil {
		return NewErrorResult(err), nil
	}

	return ToCallToolResult(result)
}

func (r *Router) List() []ToolDefinition {
	return r.registry.Definitions()
}
