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

type openAIClient struct {
	httpClient   *http.Client
	apiKey       string
	baseURL      string
	defaultModel string
}

type openAIResponseRequest struct {
	Model        string   `json:"model"`
	Input        string   `json:"input"`
	Instructions string   `json:"instructions,omitempty"`
	Temperature  *float32 `json:"temperature,omitempty"`
}

type openAIResponseEnvelope struct {
	Error      *openAIError         `json:"error,omitempty"`
	OutputText string               `json:"output_text,omitempty"`
	Output     []openAIResponseItem `json:"output,omitempty"`
}

type openAIError struct {
	Message string `json:"message"`
}

type openAIResponseItem struct {
	Type    string               `json:"type"`
	Content []openAIResponsePart `json:"content,omitempty"`
}

type openAIResponsePart struct {
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	Refusal string `json:"refusal,omitempty"`
}

func newOpenAIClient(cfg OpenAIConfig, httpClient *http.Client) (Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, nil
	}
	if httpClient == nil {
		httpClient = NewHTTPClient()
	}

	return &openAIClient{
		httpClient:   httpClient,
		apiKey:       strings.TrimSpace(cfg.APIKey),
		baseURL:      firstNonEmpty(cfg.BaseURL, DefaultOpenAIBase),
		defaultModel: strings.TrimSpace(cfg.DefaultModel),
	}, nil
}

func (c *openAIClient) Provider() Provider {
	return ProviderOpenAI
}

func (c *openAIClient) DefaultModel() string {
	return c.defaultModel
}

func (c *openAIClient) GenerateText(ctx context.Context, input GenerateTextInput) (GenerateTextResult, error) {
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

	endpoint, err := buildOpenAIEndpoint(c.baseURL)
	if err != nil {
		return GenerateTextResult{}, err
	}

	payload := openAIResponseRequest{
		Model:        model,
		Input:        prompt,
		Instructions: strings.TrimSpace(input.SystemInstruction),
		Temperature:  input.Temperature,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return GenerateTextResult{}, fmt.Errorf("marshal openai request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return GenerateTextResult{}, fmt.Errorf("build openai request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return GenerateTextResult{}, fmt.Errorf("call %q provider: %w", c.Provider(), err)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return GenerateTextResult{}, fmt.Errorf("read openai response: %w", err)
	}
	if err := resp.Body.Close(); err != nil {
		return GenerateTextResult{}, fmt.Errorf("close openai response body: %w", err)
	}

	var envelope openAIResponseEnvelope
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return GenerateTextResult{}, fmt.Errorf("decode openai response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		if envelope.Error != nil && strings.TrimSpace(envelope.Error.Message) != "" {
			return GenerateTextResult{}, fmt.Errorf("openai error: %s", strings.TrimSpace(envelope.Error.Message))
		}
		return GenerateTextResult{}, fmt.Errorf("openai error: status %d", resp.StatusCode)
	}

	text := extractOpenAIText(envelope)
	if text == "" {
		return GenerateTextResult{}, fmt.Errorf("openai returned an empty text response")
	}

	return GenerateTextResult{
		Provider: c.Provider(),
		Model:    model,
		Text:     text,
	}, nil
}

func extractOpenAIText(response openAIResponseEnvelope) string {
	if text := strings.TrimSpace(response.OutputText); text != "" {
		return text
	}

	var parts []string
	for _, item := range response.Output {
		for _, content := range item.Content {
			switch content.Type {
			case "output_text":
				if text := strings.TrimSpace(content.Text); text != "" {
					parts = append(parts, text)
				}
			case "refusal":
				if text := strings.TrimSpace(content.Refusal); text != "" {
					parts = append(parts, text)
				}
			}
		}
	}

	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func buildOpenAIEndpoint(base string) (string, error) {
	u, err := parseBaseURL(base, "https")
	if err != nil {
		return "", fmt.Errorf("parse openai base url: %w", err)
	}

	path := strings.TrimSuffix(u.Path, "/")
	switch {
	case path == "":
		u.Path = "/v1/responses"
	case strings.HasSuffix(path, "/v1"):
		u.Path = path + "/responses"
	case strings.HasSuffix(path, "/v1/responses"):
		u.Path = path
	default:
		u.Path = path + "/v1/responses"
	}

	return u.String(), nil
}
