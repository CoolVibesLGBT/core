package tools

import (
	"context"
	"core/application/mcpserver/internal/shared"
	"core/mcp"
	services "core/services/user"
	"strings"
)

type generateTextArguments struct {
	Prompt            string   `json:"prompt"`
	Model             string   `json:"model,omitempty"`
	SystemInstruction string   `json:"system_instruction,omitempty"`
	Temperature       *float32 `json:"temperature,omitempty"`
}

func RegisterAI(server *mcp.MCPServer, aiService *services.AIService) {
	server.RegisterTool(mcp.NewTool(
		mcp.ToolDefinition{
			Name:        "ai.generate_text",
			Title:       "Generate Text",
			Description: "Generate plain text with the configured Google GenAI client.",
			InputSchema: mcp.JSONSchema{
				Type: "object",
				Properties: map[string]any{
					"prompt":             shared.SchemaString("The user prompt to send to the model."),
					"model":              shared.SchemaString("Optional model name. Defaults to GOOGLE_GENAI_MODEL."),
					"system_instruction": shared.SchemaString("Optional system instruction."),
					"temperature":        shared.SchemaNumber("Optional sampling temperature."),
				},
				Required:             []string{"prompt"},
				AdditionalProperties: false,
			},
			OutputSchema: &mcp.JSONSchema{
				Type: "object",
				Properties: map[string]any{
					"text":  shared.SchemaString("Generated text."),
					"model": shared.SchemaString("Model used for generation."),
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
			if args.Model == "" {
				args.Model, _ = shared.LookupString(req.Arguments, "model")
			}
			if args.SystemInstruction == "" {
				args.SystemInstruction, _ = shared.LookupString(req.Arguments, "system_instruction", "systemInstruction", "system")
			}

			model := strings.TrimSpace(args.Model)
			if model == "" {
				model = aiService.DefaultModel()
			}

			text, err := aiService.GenerateText(ctx, services.GenerateTextInput{
				Prompt:            args.Prompt,
				Model:             model,
				SystemInstruction: args.SystemInstruction,
				Temperature:       args.Temperature,
			})
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"text":  text,
				"model": model,
			}, nil
		},
	))
}
