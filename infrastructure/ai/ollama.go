package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ollamaClient struct {
	httpClient   *http.Client
	apiKey       string
	baseURL      string
	defaultModel string
}

type ollamaGenerateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	System  string         `json:"system,omitempty"`
	Stream  bool           `json:"stream"`
	Options map[string]any `json:"options,omitempty"`
}

type ollamaGenerateResponse struct {
	Error    string `json:"error,omitempty"`
	Model    string `json:"model,omitempty"`
	Response string `json:"response,omitempty"`
}

func newOllamaClient(cfg OllamaConfig, httpClient *http.Client) (Client, error) {
	if strings.TrimSpace(cfg.DefaultModel) == "" {
		return nil, nil
	}
	if httpClient == nil {
		httpClient = NewHTTPClient()
	}

	return &ollamaClient{
		httpClient:   httpClient,
		apiKey:       strings.TrimSpace(cfg.APIKey),
		baseURL:      firstNonEmpty(cfg.BaseURL, DefaultOllamaBase),
		defaultModel: strings.TrimSpace(cfg.DefaultModel),
	}, nil
}

func (c *ollamaClient) Provider() Provider {
	return ProviderOllama
}

func (c *ollamaClient) DefaultModel() string {
	return c.defaultModel
}

func (c *ollamaClient) GenerateText(ctx context.Context, input GenerateTextInput) (GenerateTextResult, error) {
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

	endpoint, err := buildOllamaEndpoint(c.baseURL)
	if err != nil {
		return GenerateTextResult{}, err
	}

	payload := ollamaGenerateRequest{
		Model:  model,
		Prompt: prompt,
		System: strings.TrimSpace(input.SystemInstruction),
		Stream: false,
	}
	if input.Temperature != nil {
		payload.Options = map[string]any{"temperature": *input.Temperature}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return GenerateTextResult{}, fmt.Errorf("marshal ollama request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return GenerateTextResult{}, fmt.Errorf("build ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return GenerateTextResult{}, fmt.Errorf("call %q provider: %w", c.Provider(), err)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return GenerateTextResult{}, fmt.Errorf("read ollama response: %w", err)
	}
	if err := resp.Body.Close(); err != nil {
		return GenerateTextResult{}, fmt.Errorf("close ollama response body: %w", err)
	}

	var envelope ollamaGenerateResponse
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return GenerateTextResult{}, fmt.Errorf("decode ollama response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		if message := strings.TrimSpace(envelope.Error); message != "" {
			return GenerateTextResult{}, fmt.Errorf("ollama error: %s", message)
		}
		return GenerateTextResult{}, fmt.Errorf("ollama error: status %d", resp.StatusCode)
	}

	text := strings.TrimSpace(envelope.Response)
	if text == "" {
		return GenerateTextResult{}, fmt.Errorf("ollama returned an empty text response")
	}

	if envelope.Model != "" {
		model = strings.TrimSpace(envelope.Model)
	}

	return GenerateTextResult{
		Provider: c.Provider(),
		Model:    model,
		Text:     text,
	}, nil
}

func buildOllamaEndpoint(base string) (string, error) {
	u, err := parseBaseURL(base, "http")
	if err != nil {
		return "", fmt.Errorf("parse ollama base url: %w", err)
	}

	path := strings.TrimSuffix(u.Path, "/")
	switch {
	case path == "":
		u.Path = "/api/generate"
	case strings.HasSuffix(path, "/api"):
		u.Path = path + "/generate"
	case strings.HasSuffix(path, "/api/generate"):
		u.Path = path
	default:
		u.Path = path + "/api/generate"
	}

	return u.String(), nil
}
