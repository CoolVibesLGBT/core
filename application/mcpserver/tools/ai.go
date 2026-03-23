package tools

import (
	"context"
	"core/application/mcpserver/internal/shared"
	"core/mcp"
	services "core/services/user"
	"strings"
)

type generateTextArguments struct {
	Provider          string   `json:"provider,omitempty"`
	Prompt            string   `json:"prompt"`
	Model             string   `json:"model,omitempty"`
	SystemInstruction string   `json:"system_instruction,omitempty"`
	Temperature       *float32 `json:"temperature,omitempty"`
}

func RegisterAI(server *mcp.MCPServer, aiService *services.AIService) {
	if aiService == nil || !aiService.IsConfigured() {
		return
	}

	providers := aiService.Providers()
	providerSchema := shared.SchemaString("Optional provider. Defaults to the configured default AI provider.")
	if len(providers) > 0 {
		providerSchema["enum"] = providers
	}

	server.RegisterTool(mcp.NewTool(
		mcp.ToolDefinition{
			Name:        "ai.generate_text",
			Title:       "Generate Text",
			Description: "Generate plain text with one of the configured AI providers.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]any{
					"provider":           providerSchema,
					"prompt":             shared.SchemaString("The user prompt to send to the model."),
					"model":              shared.SchemaString("Optional model name. Defaults to the selected provider's configured model."),
					"system_instruction": shared.SchemaString("Optional system instruction."),
					"temperature":        shared.SchemaNumber("Optional sampling temperature."),
				},
				Required:             []string{"prompt"},
				AdditionalProperties: false,
			},
			OutputSchema: &mcp.JSONSchema{
				Type: "object",
				Properties: map[string]any{
					"text":     shared.SchemaString("Generated text."),
					"model":    shared.SchemaString("Model used for generation."),
					"provider": shared.SchemaString("Provider used for generation."),
				},
				AdditionalProperties: false,
			},
			Annotations: &mcp.ToolAnnotations{
				Title:         "Generate Text",
				ReadOnlyHint:  true,
				OpenWorldHint: true,
			},
		},
		func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
			args, err := mcp.DecodeArguments[generateTextArguments](req.Arguments)
			if err != nil {
				return nil, err
			}

			if args.Prompt == "" {
				args.Prompt, _ = shared.LookupString(req.Arguments, "prompt")
			}
			if args.Provider == "" {
				args.Provider, _ = shared.LookupString(req.Arguments, "provider")
			}
			if args.Model == "" {
				args.Model, _ = shared.LookupString(req.Arguments, "model")
			}
			if args.SystemInstruction == "" {
				args.SystemInstruction, _ = shared.LookupString(req.Arguments, "system_instruction", "systemInstruction", "system")
			}

			model := strings.TrimSpace(args.Model)
			if model == "" {
				model = aiService.DefaultModel(args.Provider)
			}

			result, err := aiService.GenerateText(ctx, services.GenerateTextInput{
				Provider:          args.Provider,
				Prompt:            args.Prompt,
				Model:             model,
				SystemInstruction: args.SystemInstruction,
				Temperature:       args.Temperature,
			})
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"text":     result.Text,
				"model":    result.Model,
				"provider": result.Provider,
			}, nil
		},
	))
}
