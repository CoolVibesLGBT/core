package services

import (
	"context"
	aiConfig "core/ai"
	"fmt"
)

type AIService struct {
	registry *aiConfig.Registry
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

func NewAIService(registry *aiConfig.Registry) *AIService {
	return &AIService{registry: registry}
}

func (s *AIService) ServiceName() string {
	return "AIService"
}

func (s *AIService) IsConfigured() bool {
	return s != nil && s.registry != nil && s.registry.IsConfigured()
}

func (s *AIService) Providers() []string {
	if s == nil || s.registry == nil {
		return nil
	}
	return s.registry.ProviderStrings()
}

func (s *AIService) DefaultProvider() string {
	if s == nil || s.registry == nil {
		return ""
	}
	return string(s.registry.DefaultProvider())
}

func (s *AIService) DefaultModel(provider string) string {
	if s == nil || s.registry == nil {
		return ""
	}
	return s.registry.DefaultModel(provider)
}

func (s *AIService) GenerateText(ctx context.Context, input GenerateTextInput) (GenerateTextResult, error) {
	if !s.IsConfigured() {
		return GenerateTextResult{}, fmt.Errorf("no AI provider is configured")
	}

	client, err := s.registry.Resolve(input.Provider)
	if err != nil {
		return GenerateTextResult{}, err
	}

	result, err := client.GenerateText(ctx, aiConfig.GenerateTextInput{
		Prompt:            input.Prompt,
		Model:             input.Model,
		SystemInstruction: input.SystemInstruction,
		Temperature:       input.Temperature,
	})
	if err != nil {
		return GenerateTextResult{}, err
	}

	return GenerateTextResult{
		Provider: string(result.Provider),
		Model:    result.Model,
		Text:     result.Text,
	}, nil
}
