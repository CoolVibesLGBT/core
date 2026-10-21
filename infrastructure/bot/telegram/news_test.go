package telegram

import (
	"core/application/types"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestSendNewsUsesPublicPostAndInternalImageSource(t *testing.T) {
	for _, useLocal := range []bool{true, false} {
		t.Run(fmt.Sprint("local=", useLocal), func(t *testing.T) {
			type sentPhoto struct {
				caption, source string
				data            []byte
			}
			sent := make(chan sentPhoto, 1)
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var captured sentPhoto
				if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
					if err := r.ParseMultipartForm(1 << 20); err != nil {
						t.Error(err)
					} else {
						defer func() {
							if err := r.MultipartForm.RemoveAll(); err != nil {
								t.Errorf("remove multipart files: %v", err)
							}
						}()
						captured.caption = r.FormValue("caption")
						for _, files := range r.MultipartForm.File {
							for _, header := range files {
								file, err := header.Open()
								if err != nil {
									t.Error(err)
									continue
								}
								captured.data, err = io.ReadAll(file)
								_ = file.Close()
								if err != nil {
									t.Error(err)
								}
							}
						}
					}
				} else {
					var payload map[string]json.RawMessage
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					_ = json.Unmarshal(payload["caption"], &captured.caption)
					_ = json.Unmarshal(payload["photo"], &captured.source)
				}
				sent <- captured
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1,"chat":{"id":1,"type":"supergroup"},"photo":[{"file_id":"test-photo","width":1,"height":1}]}}`))
			}))
			defer api.Close()
			bot, err := telegramPackage.NewBot(telegramPackage.Settings{Token: "test-token", URL: api.URL, Client: api.Client(), Offline: true})
			if err != nil {
				t.Fatal(err)
			}
			post := &types.PublicPost{PublicID: 42, Title: map[string]string{"en": "A headline"}, Summary: map[string]string{"en": "A summary"}, Attachments: []types.PublicPostMedia{{IsPublic: true, File: types.PublicMediaFile{URL: "https://cdn.example.test/photo.jpg"}}}}
			var local []string
			if useLocal {
				path := filepath.Join(t.TempDir(), "worker-photo.jpg")
				if err := os.WriteFile(path, []byte("downloaded photo"), 0600); err != nil {
					t.Fatal(err)
				}
				local = []string{path}
			}
			if err := (&Service{Bot: bot}).SendNews(post, local); err != nil {
				t.Fatal(err)
			}
			select {
			case photo := <-sent:
				if !strings.Contains(photo.caption, "A headline") || !strings.Contains(photo.caption, "A summary") {
					t.Fatalf("caption=%q", photo.caption)
				}
				if useLocal && string(photo.data) != "downloaded photo" {
					t.Fatalf("worker image not uploaded: %q", photo.data)
				}
				if !useLocal && photo.source != post.Attachments[0].File.URL {
					t.Fatalf("public image URL=%q", photo.source)
				}
			case <-time.After(time.Second):
				t.Fatal("photo was not sent to the test server")
			}
		})
	}
}
