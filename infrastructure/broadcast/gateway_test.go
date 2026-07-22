package broadcast

import (
	"context"
	"core/application/ports"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type httpClientFunc func(*http.Request) (*http.Response, error)

func (f httpClientFunc) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestGatewayFailsClosedWhenProviderCredentialsAreMissing(t *testing.T) {
	called := false
	gateway := NewGateway(Config{}, httpClientFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, errors.New("unexpected call")
	}))

	_, err := gateway.FetchTrending(context.Background(), ports.BroadcastProviderHornet, ports.BroadcastTrendingQuery{PageSize: 1})
	if !errors.Is(err, ports.ErrBroadcastIntegrationDisabled) {
		t.Fatalf("FetchTrending() error = %v, want disabled", err)
	}
	if called {
		t.Fatal("HTTP client called with incomplete credentials")
	}
}

func TestGatewayInjectsConfiguredHeadersWithoutRequiringOptionalTelemetry(t *testing.T) {
	config := Config{Hornet: HornetConfig{
		BaseURL:       "https://broadcast.example/functions/live",
		SessionToken:  "test-session-token",
		ApplicationID: "test-application",
	}}
	gateway := NewGateway(config, httpClientFunc(func(request *http.Request) (*http.Response, error) {
		if got := request.URL.String(); got != "https://broadcast.example/functions/live:getTrendingBroadcasts" {
			t.Fatalf("URL = %q", got)
		}
		if got := request.Header.Get("X-Parse-Session-Token"); got != config.Hornet.SessionToken {
			t.Fatalf("session header = %q", got)
		}
		if got := request.Header.Get("Newrelic"); got != "" {
			t.Fatalf("optional Newrelic header = %q, want empty", got)
		}
		if got := request.Header.Get("Referer"); got != "" {
			t.Fatalf("optional Referer header = %q, want empty", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"result":{}}`)),
		}, nil
	}))

	if _, err := gateway.FetchTrending(context.Background(), ports.BroadcastProviderHornet, ports.BroadcastTrendingQuery{PageSize: 10}); err != nil {
		t.Fatalf("FetchTrending() error = %v", err)
	}
}

func TestGatewayMapsUpstreamStatusWithoutLeakingCredentials(t *testing.T) {
	const sessionToken = "sensitive-test-token"
	gateway := NewGateway(Config{Hornet: HornetConfig{
		BaseURL:       "https://broadcast.example/functions/live",
		SessionToken:  sessionToken,
		ApplicationID: "test-application",
	}}, httpClientFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader(`{"error":"no"}`)),
		}, nil
	}))

	_, err := gateway.CreateBroadcast(context.Background(), "description")
	if !errors.Is(err, ports.ErrBroadcastUpstream) {
		t.Fatalf("CreateBroadcast() error = %v, want upstream", err)
	}
	if strings.Contains(err.Error(), sessionToken) {
		t.Fatal("gateway error leaked session token")
	}
}
