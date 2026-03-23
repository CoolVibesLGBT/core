package mcp

import (
	"fmt"
	"sort"
	"strings"
)

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(tool Tool) {
	definition := tool.Definition()
	name := strings.TrimSpace(definition.Name)
	if name == "" {
		panic("mcp tool name is required")
	}
	if _, exists := r.tools[name]; exists {
		panic(fmt.Sprintf("mcp tool already registered: %s", name))
	}

	definition.Name = name
	r.tools[name] = NewTool(definition, tool.Call)
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
