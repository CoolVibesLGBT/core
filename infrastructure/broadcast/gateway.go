package broadcast

import (
	"bytes"
	"context"
	"core/application/ports"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxUpstreamResponseBytes = 32 << 20

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type HornetConfig struct {
	BaseURL         string
	SessionToken    string
	ApplicationID   string
	ClientUserAgent string
	HTTPUserAgent   string
	Origin          string
	RefererBase     string
	NewRelic        string
	NewRelicID      string
}

type GrowlrConfig struct {
	BaseURL         string
	SessionToken    string
	ApplicationID   string
	ClientKey       string
	InstallationID  string
	OSVersion       string
	ClientVersion   string
	ClientUserAgent string
	HTTPUserAgent   string
	BuildVersion    string
	DisplayVersion  string
}

type Config struct {
	Hornet HornetConfig
	Growlr GrowlrConfig
}

type Gateway struct {
	config Config
	client HTTPClient
}

var _ ports.BroadcastGateway = (*Gateway)(nil)

func NewGateway(config Config, client HTTPClient) *Gateway {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Gateway{config: config, client: client}
}

func (g *Gateway) FetchTrending(ctx context.Context, provider ports.BroadcastProvider, query ports.BroadcastTrendingQuery) ([]byte, error) {
	payload := map[string]interface{}{
		"pageSize":  query.PageSize,
		"gender":    query.Gender,
		"latitude":  query.Latitude,
		"longitude": query.Longitude,
		"more":      query.More,
		"score":     query.Score,
	}
	return g.call(ctx, provider, "getTrendingBroadcasts", payload, "")
}

func (g *Gateway) CreateBroadcast(ctx context.Context, description string) ([]byte, error) {
	return g.call(ctx, ports.BroadcastProviderHornet, "createBroadcast", map[string]interface{}{
		"streamDescription": description,
	}, "")
}

func (g *Gateway) ViewBroadcast(ctx context.Context, input ports.BroadcastViewInput) ([]byte, error) {
	return g.call(ctx, ports.BroadcastProviderHornet, "viewBroadcast", map[string]interface{}{
		"broadcastId":   input.BroadcastID,
		"source":        input.Source,
		"viewBroadcast": true,
	}, input.BroadcastID)
}

func (g *Gateway) RequestGuestBroadcast(ctx context.Context, input ports.BroadcastGuestRequest) ([]byte, error) {
	return g.call(ctx, ports.BroadcastProviderHornet, "requestToGuestBroadcast", map[string]interface{}{
		"broadcastId":    input.BroadcastID,
		"streamClientId": input.StreamClientID,
	}, input.BroadcastID)
}

func (g *Gateway) LikeBroadcast(ctx context.Context, input ports.BroadcastLikeInput) ([]byte, error) {
	return g.call(ctx, ports.BroadcastProviderHornet, "likeBroadcast", map[string]interface{}{
		"broadcastId": input.BroadcastID,
		"viewerId":    input.ViewerID,
		"numLikes":    input.NumLikes,
	}, input.BroadcastID)
}

func (g *Gateway) call(ctx context.Context, provider ports.BroadcastProvider, action string, payload interface{}, broadcastID string) ([]byte, error) {
	if g == nil || g.client == nil {
		return nil, ports.ErrBroadcastIntegrationDisabled
	}
	endpoint, headers, enabled := g.requestConfig(provider, action, broadcastID)
	if !enabled {
		return nil, ports.ErrBroadcastIntegrationDisabled
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: encode %s payload", ports.ErrBroadcastUpstream, action)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: build %s request", ports.ErrBroadcastUpstream, action)
	}
	for key, value := range headers {
		if strings.TrimSpace(value) != "" {
			req.Header.Set(key, value)
		}
	}

	response, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s transport", ports.ErrBroadcastUpstream, action)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		if err := response.Body.Close(); err != nil {
			return nil, fmt.Errorf("%w: close %s response", ports.ErrBroadcastUpstream, action)
		}
		return nil, fmt.Errorf("%w: %s returned status %d", ports.ErrBroadcastUpstream, action, response.StatusCode)
	}
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxUpstreamResponseBytes+1))
	closeErr := response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("%w: read %s response", ports.ErrBroadcastUpstream, action)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("%w: close %s response", ports.ErrBroadcastUpstream, action)
	}
	if len(responseBody) > maxUpstreamResponseBytes {
		return nil, fmt.Errorf("%w: %s response is too large", ports.ErrBroadcastUpstream, action)
	}
	return responseBody, nil
}

func (g *Gateway) requestConfig(provider ports.BroadcastProvider, action, broadcastID string) (string, map[string]string, bool) {
	switch provider {
	case ports.BroadcastProviderHornet:
		config := g.config.Hornet
		if !hornetEnabled(config) {
			return "", nil, false
		}
		referer := ""
		if strings.TrimSpace(config.RefererBase) != "" {
			referer = strings.TrimRight(config.RefererBase, "/") + "/search/trending/all"
			if broadcastID != "" {
				referer = strings.TrimRight(config.RefererBase, "/") + "/view/" + url.PathEscape(broadcastID) + "/trending"
			}
		}
		return actionURL(config.BaseURL, action), map[string]string{
			"Accept":                 "application/json",
			"Accept-Language":        "en-US,en;q=0.8",
			"Content-Type":           "application/json; charset=UTF-8",
			"Origin":                 config.Origin,
			"Referer":                referer,
			"X-Parse-Application-Id": config.ApplicationID,
			"X-Parse-Session-Token":  config.SessionToken,
			"X-User-Agent":           config.ClientUserAgent,
			"User-Agent":             config.HTTPUserAgent,
			"Newrelic":               config.NewRelic,
			"X-Newrelic-Id":          config.NewRelicID,
		}, true
	case ports.BroadcastProviderGrowlr:
		config := g.config.Growlr
		if action != "getTrendingBroadcasts" || !growlrEnabled(config) {
			return "", nil, false
		}
		return actionURL(config.BaseURL, action), map[string]string{
			"Accept":                      "application/json",
			"Accept-Language":             "en-US,en;q=0.8",
			"Content-Type":                "application/json; charset=UTF-8",
			"X-Parse-Application-Id":      config.ApplicationID,
			"X-Parse-Session-Token":       config.SessionToken,
			"X-Parse-Client-Key":          config.ClientKey,
			"X-Parse-Installation-Id":     config.InstallationID,
			"X-Parse-OS-Version":          config.OSVersion,
			"X-Parse-Client-Version":      config.ClientVersion,
			"X-Parse-App-Build-Version":   config.BuildVersion,
			"X-Parse-App-Display-Version": config.DisplayVersion,
			"X-User-Agent":                config.ClientUserAgent,
			"User-Agent":                  config.HTTPUserAgent,
		}, true
	default:
		return "", nil, false
	}
}

func hornetEnabled(config HornetConfig) bool {
	return allConfigured(
		config.BaseURL,
		config.SessionToken,
		config.ApplicationID,
	)
}

func growlrEnabled(config GrowlrConfig) bool {
	return allConfigured(
		config.BaseURL,
		config.SessionToken,
		config.ApplicationID,
		config.ClientKey,
		config.InstallationID,
	)
}

func allConfigured(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return true
}

func actionURL(baseURL, action string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/:") + ":" + action
}
