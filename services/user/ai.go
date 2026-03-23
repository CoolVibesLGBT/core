package services

import (
	"context"
	aiConfig "core/ai"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

type AIService struct {
	client       *genai.Client
	defaultModel string
}

type GenerateTextInput struct {
	Prompt            string
	Model             string
	SystemInstruction string
	Temperature       *float32
}

func NewAIService(
	client *genai.Client,
	config aiConfig.Config,
) *AIService {
	return &AIService{
		client:       client,
		defaultModel: config.DefaultModel,
	}
}

func (s *AIService) ServiceName() string {
	return "AIService"
}

func (s *AIService) Client() *genai.Client {
	return s.client
}

func (s *AIService) DefaultModel() string {
	return s.defaultModel
}

func (s *AIService) GenerateText(ctx context.Context, input GenerateTextInput) (string, error) {
	if s.client == nil {
		return "", errors.New("google genai client is not configured; set GOOGLE_API_KEY or Vertex AI environment variables")
	}

	prompt := strings.TrimSpace(input.Prompt)
	if prompt == "" {
		return "", errors.New("prompt is required")
	}

	model := strings.TrimSpace(input.Model)
	if model == "" {
		model = s.defaultModel
	}

	config := &genai.GenerateContentConfig{}
	if input.Temperature != nil {
		config.Temperature = input.Temperature
	}

	if instruction := strings.TrimSpace(input.SystemInstruction); instruction != "" {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{
				{Text: instruction},
			},
		}
	}

	result, err := s.client.Models.GenerateContent(ctx, model, genai.Text(prompt), config)
	if err != nil {
		return "", fmt.Errorf("generate content: %w", err)
	}

	return strings.TrimSpace(result.Text()), nil
}
