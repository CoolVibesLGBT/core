package usecases

import (
	"context"
	"core/application/ports"
	"fmt"
)

type AIService struct {
	generator ports.TextGenerator
}

type GenerateTextInput struct {
	Provider          string
	Prompt            string
	Model             string
	SystemInstruction string
	Temperature       *float32
}

type GenerateTextResult struct {
	Provider string
	Model    string
	Text     string
}

func NewAIService(generator ports.TextGenerator) *AIService {
	return &AIService{generator: generator}
}

func (s *AIService) ServiceName() string {
	return "AIService"
}

func (s *AIService) IsConfigured() bool {
	return s != nil && s.generator != nil && s.generator.IsConfigured()
}

func (s *AIService) Providers() []string {
	if s == nil || s.generator == nil {
		return nil
	}
	return s.generator.ProviderStrings()
}

func (s *AIService) DefaultProvider() string {
	if s == nil || s.generator == nil {
		return ""
	}
	return s.generator.DefaultProviderName()
}

func (s *AIService) DefaultModel(provider string) string {
	if s == nil || s.generator == nil {
		return ""
	}
	return s.generator.DefaultModel(provider)
}

func (s *AIService) GenerateText(ctx context.Context, input GenerateTextInput) (GenerateTextResult, error) {
	if !s.IsConfigured() {
		return GenerateTextResult{}, fmt.Errorf("no AI provider is configured")
	}

	result, err := s.generator.GenerateText(ctx, ports.TextGenerationInput{
		Provider:          input.Provider,
		Prompt:            input.Prompt,
		Model:             input.Model,
		SystemInstruction: input.SystemInstruction,
		Temperature:       input.Temperature,
	})
	if err != nil {
		return GenerateTextResult{}, err
	}

	return GenerateTextResult{
		Provider: result.Provider,
		Model:    result.Model,
		Text:     result.Text,
	}, nil
}
