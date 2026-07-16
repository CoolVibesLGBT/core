package ai

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

type geminiClient struct {
	client       *genai.Client
	defaultModel string
}

func newGeminiClient(cfg GeminiConfig) (Client, error) {
	if !isGeminiConfigured() {
		return nil, nil
	}

	client, err := genai.NewClient(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	return &geminiClient{
		client:       client,
		defaultModel: strings.TrimSpace(cfg.DefaultModel),
	}, nil
}

func (c *geminiClient) Provider() Provider {
	return ProviderGemini
}

func (c *geminiClient) DefaultModel() string {
	return c.defaultModel
}

func (c *geminiClient) GenerateText(ctx context.Context, input GenerateTextInput) (GenerateTextResult, error) {
	prompt := strings.TrimSpace(input.Prompt)
	if prompt == "" {
		return GenerateTextResult{}, fmt.Errorf("prompt is required")
	}

	model := strings.TrimSpace(input.Model)
	if model == "" {
		model = c.defaultModel
	}
	if model == "" {
		return GenerateTextResult{}, fmt.Errorf("model is required for provider %q", c.Provider())
	}

	config := &genai.GenerateContentConfig{}
	if input.Temperature != nil {
		config.Temperature = input.Temperature
	}

	if instruction := strings.TrimSpace(input.SystemInstruction); instruction != "" {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: instruction}},
		}
	}

	result, err := c.client.Models.GenerateContent(ctx, model, genai.Text(prompt), config)
	if err != nil {
		return GenerateTextResult{}, fmt.Errorf("generate content with %q: %w", c.Provider(), err)
	}

	return GenerateTextResult{
		Provider: c.Provider(),
		Model:    model,
		Text:     strings.TrimSpace(result.Text()),
	}, nil
}
