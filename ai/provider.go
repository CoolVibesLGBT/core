package ai

import (
	"context"
	"os"
	"strings"

	"github.com/google/wire"
	"google.golang.org/genai"
)

const DefaultModel = "gemini-2.5-flash"

type Config struct {
	DefaultModel string
}

var ProviderSet = wire.NewSet(NewConfig, NewClient)

func NewConfig() Config {
	model := strings.TrimSpace(os.Getenv("GOOGLE_GENAI_MODEL"))
	if model == "" {
		model = DefaultModel
	}

	return Config{
		DefaultModel: model,
	}
}

func NewClient() (*genai.Client, error) {
	if !isConfigured() {
		return nil, nil
	}

	return genai.NewClient(context.Background(), nil)
}

func isConfigured() bool {
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
