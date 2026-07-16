package mcpserver

import (
	usecases "core/application/usecases"
	aiConfig "core/infrastructure/ai"
	"net/http"
	"testing"
)

func TestNewServerSkipsAIWhenClientIsNotConfigured(t *testing.T) {
	aiService := usecases.NewAIService(nil)

	server, err := NewServer(aiService, nil, nil)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
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

	aiService := usecases.NewAIService(registry)
	server, err := NewServer(aiService, nil, nil)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	for _, tool := range server.ListTools() {
		if tool.Name == "ai.generate_text" {
			return
		}
	}

	t.Fatal("expected ai.generate_text to be registered when an AI provider is configured")
}
