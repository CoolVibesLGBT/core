package handlers

import (
	"context"
	"core/constants"
	"core/utils"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type MetaData struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	ImageAlt    string `json:"image_alt,omitempty"`
	URL         string `json:"url,omitempty"`
	SiteName    string `json:"site_name,omitempty"`
	Type        string `json:"type,omitempty"`
	Locale      string `json:"locale,omitempty"`
}

// OEmbedData is the normalized JSON representation defined by oEmbed 1.0.
// Provider HTML is returned verbatim and must be rendered in a sandboxed,
// off-origin iframe by clients.
type OEmbedData struct {
	Version         string `json:"version"`
	Type            string `json:"type"`
	Title           string `json:"title,omitempty"`
	AuthorName      string `json:"author_name,omitempty"`
	AuthorURL       string `json:"author_url,omitempty"`
	ProviderName    string `json:"provider_name,omitempty"`
	ProviderURL     string `json:"provider_url,omitempty"`
	CacheAge        int    `json:"cache_age,omitempty"`
	ThumbnailURL    string `json:"thumbnail_url,omitempty"`
	ThumbnailWidth  int    `json:"thumbnail_width,omitempty"`
	ThumbnailHeight int    `json:"thumbnail_height,omitempty"`
	URL             string `json:"url,omitempty"`
	HTML            string `json:"html,omitempty"`
	Width           int    `json:"width,omitempty"`
	Height          int    `json:"height,omitempty"`
}

type EmbedResponse struct {
	OGMeta      *MetaData   `json:"og,omitempty"`
	TwitterMeta *MetaData   `json:"twitter,omitempty"`
	OEmbed      *OEmbedData `json:"oembed,omitempty"`
}

func HandleLinkPreview() fiber.Handler {
	client := newSafeEmbedHTTPClient()

	return func(c fiber.Ctx) error {
		targetURL, err := decodeEmbedTarget(c.FormValue("url"))
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidForm)
		}

		preview, err := resolveLinkPreview(c.Context(), client, targetURL)
		if err != nil {
			switch {
			case errors.Is(err, errInvalidEmbedURL):
				return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidForm)
			case errors.Is(err, errEmbedHTMLParse):
				return utils.SendError(c, fiber.StatusUnprocessableEntity, constants.ErrHTMLParseFailed)
			default:
				return utils.SendError(c, fiber.StatusBadGateway, constants.ErrOGFetchFailed)
			}
		}

		return utils.SendSuccess(c, fiber.StatusOK, preview)
	}
}

func decodeEmbedTarget(rawURL string) (*url.URL, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, errInvalidEmbedURL
	}

	// The current web client sends encodeURIComponent(url). Accept that format
	// while also supporting callers that submit a normal absolute URL.
	lowerRawURL := strings.ToLower(rawURL)
	if strings.HasPrefix(lowerRawURL, "http%3a%2f%2f") || strings.HasPrefix(lowerRawURL, "https%3a%2f%2f") {
		decoded, err := url.QueryUnescape(rawURL)
		if err != nil {
			return nil, errInvalidEmbedURL
		}
		decoded = strings.TrimSpace(decoded)
		if strings.HasPrefix(strings.ToLower(decoded), "http://") || strings.HasPrefix(strings.ToLower(decoded), "https://") {
			rawURL = decoded
		}
	}

	targetURL, err := url.Parse(rawURL)
	if err != nil || targetURL.Hostname() == "" || targetURL.User != nil {
		return nil, errInvalidEmbedURL
	}
	if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
		return nil, errInvalidEmbedURL
	}

	return targetURL, nil
}

func resolveLinkPreview(ctx context.Context, client *http.Client, targetURL *url.URL) (EmbedResponse, error) {
	document, err := fetchEmbedResource(ctx, client, targetURL, "text/html,application/xhtml+xml", maxEmbedHTMLBytes)
	if err != nil {
		return EmbedResponse{}, err
	}

	og, twitter, discoveryURL, err := parseLinkMetadata(document, targetURL)
	if err != nil {
		return EmbedResponse{}, err
	}

	response := EmbedResponse{
		OGMeta:      og,
		TwitterMeta: twitter,
	}
	if discoveryURL == nil {
		discoveryURL = findRegisteredOEmbedEndpoint(ctx, client, targetURL)
	}
	if discoveryURL != nil {
		response.OEmbed = fetchOEmbed(ctx, client, discoveryURL, targetURL)
	}

	return response, nil
}
