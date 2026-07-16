package mcp

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrToolNameRequired      = errors.New("mcp tool name is required")
	ErrToolAlreadyRegistered = errors.New("mcp tool already registered")
)

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(tool Tool) error {
	definition := tool.Definition()
	name := strings.TrimSpace(definition.Name)
	if name == "" {
		return ErrToolNameRequired
	}
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("%w: %s", ErrToolAlreadyRegistered, name)
	}

	definition.Name = name
	r.tools[name] = NewTool(definition, tool.Call)
	return nil
}

func (r *Registry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *Registry) Definitions() []ToolDefinition {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}

	sort.Strings(names)

	definitions := make([]ToolDefinition, 0, len(names))
	for _, name := range names {
		definitions = append(definitions, r.tools[name].Definition())
	}

	return definitions
}
