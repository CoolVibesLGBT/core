package telegram

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	telegramPackage "gopkg.in/telebot.v4"
)

func TestBotSettingsAreNetworkFreeAndBounded(t *testing.T) {
	settings := botSettings("token")
	if !settings.Offline {
		t.Fatal("bot settings must be offline so NewBot does not call getMe")
	}
	if settings.Client == nil || settings.Client.Timeout != telegramHTTPTimeout {
		t.Fatalf("HTTP timeout = %v, want %v", settings.Client, telegramHTTPTimeout)
	}
	if settings.Client.Timeout <= 0 || settings.Client.Timeout > 10*time.Second {
		t.Fatalf("HTTP timeout must be positive and bounded, got %v", settings.Client.Timeout)
	}
}

func TestNewDisablesTelegramWithoutToken(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")

	service, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if service != nil {
		t.Fatalf("New() service = %#v, want nil", service)
	}
}

func TestNewConfiguredBotDoesNotContactTelegram(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")

	service, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if service == nil || service.Bot == nil {
		t.Fatal("New() did not create an offline bot")
	}
	if service.Bot.Me == nil || service.Bot.Me.ID != 0 {
		t.Fatalf("offline bot identity = %#v, want empty identity", service.Bot.Me)
	}
}

func TestRegisterWebhookUsesEnvironmentAndDisablesLongPolling(t *testing.T) {
	var payload map[string]string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bottest-token/setWebhook" {
			t.Errorf("request path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer api.Close()

	bot, err := telegramPackage.NewBot(telegramPackage.Settings{
		Token:   "test-token",
		URL:     api.URL,
		Client:  api.Client(),
		Offline: true,
	})
	if err != nil {
		t.Fatalf("NewBot() error = %v", err)
	}
	service := &Service{Bot: bot}

	t.Setenv("TELEGRAM_WEBHOOK_URL", "https://api.example.test/webhook/telegram")
	t.Setenv("TELEGRAM_WEBHOOK_SECRET", "safe_secret-123")

	if err := service.RegisterWebhook(); err != nil {
		t.Fatalf("RegisterWebhook() error = %v", err)
	}
	if payload["url"] != "https://api.example.test/webhook/telegram" {
		t.Fatalf("webhook URL = %q", payload["url"])
	}
	if payload["secret_token"] != "safe_secret-123" {
		t.Fatalf("secret token = %q", payload["secret_token"])
	}
	if !service.webhookMode.Load() {
		t.Fatal("webhook mode was not enabled")
	}

	started := time.Now()
	service.Start()
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("Start() blocked in webhook mode for %v", elapsed)
	}
}

func TestRegisterWebhookFailsClosedBeforeNetwork(t *testing.T) {
	requests := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer api.Close()

	bot, err := telegramPackage.NewBot(telegramPackage.Settings{
		Token:   "test-token",
		URL:     api.URL,
		Client:  api.Client(),
		Offline: true,
	})
	if err != nil {
		t.Fatalf("NewBot() error = %v", err)
	}
	service := &Service{Bot: bot}

	t.Setenv("TELEGRAM_WEBHOOK_URL", "https://api.example.test/webhook/telegram")
	t.Setenv("TELEGRAM_WEBHOOK_SECRET", "")
	if err := service.RegisterWebhook(); err == nil {
		t.Fatal("RegisterWebhook() error = nil, want missing-secret error")
	}
	if requests != 0 {
		t.Fatalf("Telegram API requests = %d, want 0", requests)
	}

	t.Setenv("TELEGRAM_WEBHOOK_URL", "http://insecure.example.test/webhook/telegram")
	t.Setenv("TELEGRAM_WEBHOOK_SECRET", "safe_secret")
	if err := service.RegisterWebhook(); err == nil {
		t.Fatal("RegisterWebhook() error = nil, want invalid-URL error")
	}
	if requests != 0 {
		t.Fatalf("Telegram API requests = %d, want 0", requests)
	}
}
