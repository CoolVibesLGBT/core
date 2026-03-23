package ai

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/wire"
)

type Provider string

const (
	ProviderGemini Provider = "gemini"
	ProviderOpenAI Provider = "openai"
	ProviderOllama Provider = "ollama"
)

const (
	DefaultGeminiModel = "gemini-2.5-flash"
	DefaultOpenAIModel = "gpt-5-mini"
	DefaultOpenAIBase  = "https://api.openai.com"
	DefaultOllamaBase  = "http://127.0.0.1:11434"
	DefaultHTTPTimeout = 60 * time.Second
)

type Config struct {
	DefaultProvider Provider
	Gemini          GeminiConfig
	OpenAI          OpenAIConfig
	Ollama          OllamaConfig
}

type GeminiConfig struct {
	DefaultModel string
}

type OpenAIConfig struct {
	APIKey       string
	BaseURL      string
	DefaultModel string
}

type OllamaConfig struct {
	APIKey       string
	BaseURL      string
	DefaultModel string
}

type GenerateTextInput struct {
	Prompt            string
	Model             string
	SystemInstruction string
	Temperature       *float32
}

type GenerateTextResult struct {
	Provider Provider
	Model    string
	Text     string
}

type Client interface {
	Provider() Provider
	DefaultModel() string
	GenerateText(context.Context, GenerateTextInput) (GenerateTextResult, error)
}

type Registry struct {
	defaultProvider Provider
	clients         map[Provider]Client
}

var ProviderSet = wire.NewSet(NewConfig, NewHTTPClient, NewRegistry)

func NewConfig() Config {
	defaultProvider, _ := parseProvider(firstNonEmpty(
		os.Getenv("AI_DEFAULT_PROVIDER"),
		os.Getenv("AI_PROVIDER"),
	))

	genericModel := strings.TrimSpace(os.Getenv("AI_MODEL"))

	openAIModel := firstNonEmpty(os.Getenv("OPENAI_MODEL"), genericModel, DefaultOpenAIModel)
	ollamaModel := firstNonEmpty(os.Getenv("OLLAMA_MODEL"), genericModel)

	return Config{
		DefaultProvider: defaultProvider,
		Gemini: GeminiConfig{
			DefaultModel: firstNonEmpty(os.Getenv("GOOGLE_GENAI_MODEL"), genericModel, DefaultGeminiModel),
		},
		OpenAI: OpenAIConfig{
			APIKey:       strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
			BaseURL:      firstNonEmpty(os.Getenv("OPENAI_BASE_URL"), DefaultOpenAIBase),
			DefaultModel: openAIModel,
		},
		Ollama: OllamaConfig{
			APIKey:       strings.TrimSpace(os.Getenv("OLLAMA_API_KEY")),
			BaseURL:      firstNonEmpty(os.Getenv("OLLAMA_BASE_URL"), os.Getenv("OLLAMA_HOST"), DefaultOllamaBase),
			DefaultModel: ollamaModel,
		},
	}
}

func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: DefaultHTTPTimeout}
}

func NewRegistry(cfg Config, httpClient *http.Client) (*Registry, error) {
	if httpClient == nil {
		httpClient = NewHTTPClient()
	}

	registry := &Registry{
		defaultProvider: cfg.DefaultProvider,
		clients:         make(map[Provider]Client, 3),
	}

	constructors := []func(Config, *http.Client) (Client, error){
		func(cfg Config, _ *http.Client) (Client, error) {
			return newGeminiClient(cfg.Gemini)
		},
		func(cfg Config, httpClient *http.Client) (Client, error) {
			return newOpenAIClient(cfg.OpenAI, httpClient)
		},
		func(cfg Config, httpClient *http.Client) (Client, error) {
			return newOllamaClient(cfg.Ollama, httpClient)
		},
	}

	for _, construct := range constructors {
		client, err := construct(cfg, httpClient)
		if err != nil {
			return nil, err
		}
		if client == nil {
			continue
		}
		registry.clients[client.Provider()] = client
	}

	if len(registry.clients) == 0 {
		return nil, nil
	}

	if registry.defaultProvider != "" {
		if _, ok := registry.clients[registry.defaultProvider]; !ok {
			return nil, fmt.Errorf("default AI provider %q is not configured", registry.defaultProvider)
		}
		return registry, nil
	}

	providers := registry.Providers()
	registry.defaultProvider = providers[0]
	return registry, nil
}

func (r *Registry) IsConfigured() bool {
	return r != nil && len(r.clients) > 0
}

func (r *Registry) DefaultProvider() Provider {
	if r == nil {
		return ""
	}
	return r.defaultProvider
}

func (r *Registry) Providers() []Provider {
	if r == nil {
		return nil
	}

	ordered := make([]Provider, 0, len(r.clients))
	for _, provider := range []Provider{ProviderGemini, ProviderOpenAI, ProviderOllama} {
		if _, ok := r.clients[provider]; ok {
			ordered = append(ordered, provider)
		}
	}
	return ordered
}

func (r *Registry) ProviderStrings() []string {
	providers := r.Providers()
	values := make([]string, 0, len(providers))
	for _, provider := range providers {
		values = append(values, string(provider))
	}
	return values
}

func (r *Registry) DefaultModel(provider string) string {
	client, err := r.Resolve(provider)
	if err != nil {
		return ""
	}
	return client.DefaultModel()
}

func (r *Registry) Resolve(provider string) (Client, error) {
	if !r.IsConfigured() {
		return nil, fmt.Errorf("no AI providers are configured")
	}

	selected := strings.TrimSpace(provider)
	if selected == "" {
		return r.clients[r.defaultProvider], nil
	}

	parsed, ok := parseProvider(selected)
	if !ok {
		return nil, fmt.Errorf("unsupported AI provider %q; configured providers: %s", selected, strings.Join(r.ProviderStrings(), ", "))
	}

	client, ok := r.clients[parsed]
	if !ok {
		return nil, fmt.Errorf("AI provider %q is not configured; configured providers: %s", selected, strings.Join(r.ProviderStrings(), ", "))
	}

	return client, nil
}

func parseProvider(value string) (Provider, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(ProviderGemini):
		return ProviderGemini, true
	case string(ProviderOpenAI), "chatgpt":
		return ProviderOpenAI, true
	case string(ProviderOllama):
		return ProviderOllama, true
	default:
		return "", false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func isGeminiConfigured() bool {
	if os.Getenv("GOOGLE_API_KEY") != "" || os.Getenv("GEMINI_API_KEY") != "" {
		return true
	}

	if !isTruthy(os.Getenv("GOOGLE_GENAI_USE_VERTEXAI")) {
		return false
	}

	return os.Getenv("GOOGLE_CLOUD_PROJECT") != "" &&
		(os.Getenv("GOOGLE_CLOUD_LOCATION") != "" || os.Getenv("GOOGLE_CLOUD_REGION") != "")
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseBaseURL(raw string, defaultScheme string) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("base url is required")
	}

	if !strings.Contains(trimmed, "://") && !strings.HasPrefix(trimmed, "/") {
		trimmed = defaultScheme + "://" + trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, err
	}

	return parsed, nil
}
