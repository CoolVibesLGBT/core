package mcpserver

import (
	aiConfig "core/ai"
	services "core/services/user"
	"net/http"
	"testing"
)

func TestNewServerSkipsAIWhenClientIsNotConfigured(t *testing.T) {
	aiService := services.NewAIService(nil)

	server := NewServer(aiService, nil, nil)
	tools := server.ListTools()

	for _, tool := range tools {
		if tool.Name == "ai.generate_text" {
			t.Fatal("expected ai.generate_text to be omitted when the GenAI client is not configured")
		}
	}
}

func TestNewServerRegistersAIWhenProviderIsConfigured(t *testing.T) {
	registry, err := aiConfig.NewRegistry(aiConfig.Config{
		OpenAI: aiConfig.OpenAIConfig{
			APIKey:       "test-key",
			BaseURL:      "https://api.openai.com",
			DefaultModel: "gpt-5-mini",
		},
	}, &http.Client{})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	aiService := services.NewAIService(registry)
	server := NewServer(aiService, nil, nil)

	for _, tool := range server.ListTools() {
		if tool.Name == "ai.generate_text" {
			return
		}
	}

	t.Fatal("expected ai.generate_text to be registered when an AI provider is configured")
}
