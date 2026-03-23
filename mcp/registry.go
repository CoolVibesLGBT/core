package mcp

import "sort"

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
	r.tools[definition.Name] = tool
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
