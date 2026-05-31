package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type stubClient struct {
	provider     Provider
	defaultModel string
}

func (c stubClient) Provider() Provider {
	return c.provider
}

func (c stubClient) DefaultModel() string {
	return c.defaultModel
}

func (c stubClient) GenerateText(context.Context, GenerateTextInput) (GenerateTextResult, error) {
	return GenerateTextResult{
		Provider: c.provider,
		Model:    c.defaultModel,
		Text:     "ok",
	}, nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestRegistryResolveSupportsDefaultAndExplicitProvider(t *testing.T) {
	registry := &Registry{
		defaultProvider: ProviderOpenAI,
		clients: map[Provider]Client{
			ProviderOpenAI: stubClient{provider: ProviderOpenAI, defaultModel: "gpt-5-mini"},
			ProviderOllama: stubClient{provider: ProviderOllama, defaultModel: "llama3.2"},
		},
	}

	client, err := registry.Resolve("")
	if err != nil {
		t.Fatalf("Resolve(default) error = %v", err)
	}
	if client.Provider() != ProviderOpenAI {
		t.Fatalf("Resolve(default) provider = %q, want %q", client.Provider(), ProviderOpenAI)
	}

	client, err = registry.Resolve("ollama")
	if err != nil {
		t.Fatalf("Resolve(ollama) error = %v", err)
	}
	if client.Provider() != ProviderOllama {
		t.Fatalf("Resolve(ollama) provider = %q, want %q", client.Provider(), ProviderOllama)
	}

	if _, err := registry.Resolve("gemini"); err == nil {
		t.Fatal("Resolve(gemini) expected error when provider is not configured")
	}
}

func TestOpenAIClientGenerateText(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/v1/responses" {
				t.Fatalf("path = %q, want %q", r.URL.Path, "/v1/responses")
			}
			if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
				t.Fatalf("authorization header = %q", got)
			}

			var body openAIResponseRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if body.Model != "gpt-5-mini" {
				t.Fatalf("model = %q, want %q", body.Model, "gpt-5-mini")
			}
			if body.Input != "write something" {
				t.Fatalf("input = %q", body.Input)
			}
			if body.Instructions != "be concise" {
				t.Fatalf("instructions = %q", body.Instructions)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"output":[{"type":"message","content":[{"type":"output_text","text":"hello from openai"}]}]}`)),
			}, nil
		}),
	}

	client, err := newOpenAIClient(OpenAIConfig{
		APIKey:       "test-key",
		BaseURL:      "https://api.openai.com",
		DefaultModel: "gpt-5-mini",
	}, httpClient)
	if err != nil {
		t.Fatalf("newOpenAIClient() error = %v", err)
	}

	result, err := client.GenerateText(context.Background(), GenerateTextInput{
		Prompt:            "write something",
		SystemInstruction: "be concise",
	})
	if err != nil {
		t.Fatalf("GenerateText() error = %v", err)
	}

	if result.Provider != ProviderOpenAI {
		t.Fatalf("provider = %q, want %q", result.Provider, ProviderOpenAI)
	}
	if result.Model != "gpt-5-mini" {
		t.Fatalf("model = %q, want %q", result.Model, "gpt-5-mini")
	}
	if result.Text != "hello from openai" {
		t.Fatalf("text = %q", result.Text)
	}
}

func TestOllamaClientGenerateText(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/api/generate" {
				t.Fatalf("path = %q, want %q", r.URL.Path, "/api/generate")
			}
			if got := r.Header.Get("Authorization"); got != "Bearer test-ollama-key" {
				t.Fatalf("authorization header = %q", got)
			}

			var body ollamaGenerateRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if body.Model != "llama3.2" {
				t.Fatalf("model = %q, want %q", body.Model, "llama3.2")
			}
			if body.Prompt != "write something" {
				t.Fatalf("prompt = %q", body.Prompt)
			}
			if body.System != "be concise" {
				t.Fatalf("system = %q", body.System)
			}
			if body.Stream {
				t.Fatal("expected stream=false")
			}
			if got := body.Options["temperature"]; got != float64(0.2) {
				t.Fatalf("temperature = %v, want 0.2", got)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"model":"llama3.2","response":"hello from ollama"}`)),
			}, nil
		}),
	}

	client, err := newOllamaClient(OllamaConfig{
		APIKey:       "test-ollama-key",
		BaseURL:      "http://127.0.0.1:11434",
		DefaultModel: "llama3.2",
	}, httpClient)
	if err != nil {
		t.Fatalf("newOllamaClient() error = %v", err)
	}

	temperature := float32(0.2)
	result, err := client.GenerateText(context.Background(), GenerateTextInput{
		Prompt:            "write something",
		SystemInstruction: "be concise",
		Temperature:       &temperature,
	})
	if err != nil {
		t.Fatalf("GenerateText() error = %v", err)
	}

	if result.Provider != ProviderOllama {
		t.Fatalf("provider = %q, want %q", result.Provider, ProviderOllama)
	}
	if result.Model != "llama3.2" {
		t.Fatalf("model = %q, want %q", result.Model, "llama3.2")
	}
	if result.Text != "hello from ollama" {
		t.Fatalf("text = %q", result.Text)
	}
}

func TestBuildOpenAIEndpointAcceptsV1Base(t *testing.T) {
	endpoint, err := buildOpenAIEndpoint("https://example.com/v1")
	if err != nil {
		t.Fatalf("buildOpenAIEndpoint() error = %v", err)
	}
	if endpoint != "https://example.com/v1/responses" {
		t.Fatalf("endpoint = %q", endpoint)
	}
}

func TestBuildOllamaEndpointAcceptsAPIRoot(t *testing.T) {
	endpoint, err := buildOllamaEndpoint("https://example.com/api")
	if err != nil {
		t.Fatalf("buildOllamaEndpoint() error = %v", err)
	}
	if endpoint != "https://example.com/api/generate" {
		t.Fatalf("endpoint = %q", endpoint)
	}
}

func TestBuildOllamaEndpointAcceptsHostWithoutScheme(t *testing.T) {
	endpoint, err := buildOllamaEndpoint("127.0.0.1:11434")
	if err != nil {
		t.Fatalf("buildOllamaEndpoint() error = %v", err)
	}
	if endpoint != "http://127.0.0.1:11434/api/generate" {
		t.Fatalf("endpoint = %q", endpoint)
	}
}

func TestParseProviderAcceptsChatGPTAlias(t *testing.T) {
	provider, ok := parseProvider("ChatGPT")
	if !ok {
		t.Fatal("parseProvider(ChatGPT) expected success")
	}
	if provider != ProviderOpenAI {
		t.Fatalf("provider = %q, want %q", provider, ProviderOpenAI)
	}
}

func TestNewConfigDefaults(t *testing.T) {
	t.Setenv("AI_DEFAULT_PROVIDER", "")
	t.Setenv("AI_PROVIDER", "")
	t.Setenv("AI_MODEL", "")
	t.Setenv("GOOGLE_GENAI_MODEL", "")
	t.Setenv("OPENAI_MODEL", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "")
	t.Setenv("OLLAMA_MODEL", "")
	t.Setenv("OLLAMA_BASE_URL", "")
	t.Setenv("OLLAMA_HOST", "")

	cfg := NewConfig()

	if cfg.Gemini.DefaultModel != DefaultGeminiModel {
		t.Fatalf("Gemini.DefaultModel = %q, want %q", cfg.Gemini.DefaultModel, DefaultGeminiModel)
	}
	if cfg.OpenAI.DefaultModel != DefaultOpenAIModel {
		t.Fatalf("OpenAI.DefaultModel = %q, want %q", cfg.OpenAI.DefaultModel, DefaultOpenAIModel)
	}
	if strings.TrimSpace(cfg.Ollama.BaseURL) != DefaultOllamaBase {
		t.Fatalf("Ollama.BaseURL = %q, want %q", cfg.Ollama.BaseURL, DefaultOllamaBase)
	}
}
