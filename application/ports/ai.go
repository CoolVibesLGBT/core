package ports

import "context"

type TextGenerationInput struct {
	Provider          string
	Prompt            string
	Model             string
	SystemInstruction string
	Temperature       *float32
}

type TextGenerationResult struct {
	Provider string
	Model    string
	Text     string
}

type TextGenerator interface {
	IsConfigured() bool
	ProviderStrings() []string
	DefaultProviderName() string
	DefaultModel(provider string) string
	GenerateText(context.Context, TextGenerationInput) (TextGenerationResult, error)
}
