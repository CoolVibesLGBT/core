package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

type recordingTelegramProcessor struct {
	calls int
}

func (p *recordingTelegramProcessor) ProcessUpdate(any) error {
	p.calls++
	return nil
}

type typedNilTelegramProcessor struct{}

func (*typedNilTelegramProcessor) ProcessUpdate(any) error { return nil }

func TestTelegramWebhookSecretFailsClosed(t *testing.T) {
	processor := &recordingTelegramProcessor{}
	app := fiber.New()
	app.Post("/telegram", HandleTelegramUpdates(processor))

	t.Setenv("TELEGRAM_WEBHOOK_SECRET", "")
	request := httptest.NewRequest(http.MethodPost, "/telegram", nil)
	request.Header.Set("X-Telegram-Bot-Api-Secret-Token", "")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusForbidden)
	}
	if processor.calls != 0 {
		t.Fatalf("processor calls = %d, want 0", processor.calls)
	}
}

func TestTelegramWebhookRequiresExactSecret(t *testing.T) {
	processor := &recordingTelegramProcessor{}
	app := fiber.New()
	app.Post("/telegram", HandleTelegramUpdates(processor))

	t.Setenv("TELEGRAM_WEBHOOK_SECRET", "expected_secret")
	request := httptest.NewRequest(http.MethodPost, "/telegram", nil)
	request.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong_secret")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusForbidden)
	}
	if processor.calls != 0 {
		t.Fatalf("processor calls = %d, want 0", processor.calls)
	}
}

func TestTelegramWebhookAcceptsConfiguredSecret(t *testing.T) {
	processor := &recordingTelegramProcessor{}
	app := fiber.New()
	app.Post("/telegram", HandleTelegramUpdates(processor))

	t.Setenv("TELEGRAM_WEBHOOK_SECRET", "expected_secret")
	request := httptest.NewRequest(http.MethodPost, "/telegram", strings.NewReader(`{"update_id":1}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Telegram-Bot-Api-Secret-Token", "expected_secret")

	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
	if processor.calls != 1 {
		t.Fatalf("processor calls = %d, want 1", processor.calls)
	}
}

func TestTelegramWebhookRejectsTypedNilProcessor(t *testing.T) {
	var processor *typedNilTelegramProcessor
	app := fiber.New()
	app.Post("/telegram", HandleTelegramUpdates(processor))

	t.Setenv("TELEGRAM_WEBHOOK_SECRET", "expected_secret")
	request := httptest.NewRequest(http.MethodPost, "/telegram", strings.NewReader(`{"update_id":1}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Telegram-Bot-Api-Secret-Token", "expected_secret")

	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusServiceUnavailable)
	}
}
