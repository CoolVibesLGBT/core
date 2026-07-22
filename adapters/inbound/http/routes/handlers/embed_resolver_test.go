package handlers

import (
	"encoding/json"
	"net"
	"net/url"
	"testing"
)

func TestDecodeEmbedTargetAcceptsEncodedURLAndPreservesRawPlus(t *testing.T) {
	encoded, err := decodeEmbedTarget("https%3A%2F%2Fexample.com%2Fwatch%3Fv%3Done%2520two")
	if err != nil {
		t.Fatalf("decodeEmbedTarget(encoded) error = %v", err)
	}
	if encoded.String() != "https://example.com/watch?v=one%20two" {
		t.Fatalf("decoded URL = %q", encoded.String())
	}

	raw, err := decodeEmbedTarget("https://example.com/search?q=one+two")
	if err != nil {
		t.Fatalf("decodeEmbedTarget(raw) error = %v", err)
	}
	if raw.RawQuery != "q=one+two" {
		t.Fatalf("raw query changed to %q", raw.RawQuery)
	}
}

func TestParseLinkMetadataDiscoversJSONOEmbed(t *testing.T) {
	documentURL, _ := url.Parse("https://video.example/posts/42")
	document := []byte(`<!doctype html>
<html>
  <head>
    <title>Fallback title</title>
    <meta property="og:title" content="A CoolVibes clip">
    <meta property="og:description" content="Clip description">
    <meta property="og:image" content="/images/cover.jpg">
    <meta property="og:site_name" content="Video Example">
    <meta name="twitter:card" content="player">
    <link rel="alternate" type="application/json+oembed" href="/oembed?url=https%3A%2F%2Fvideo.example%2Fposts%2F42">
  </head>
</html>`)

	og, twitter, discovery, err := parseLinkMetadata(document, documentURL)
	if err != nil {
		t.Fatalf("parseLinkMetadata() error = %v", err)
	}
	if og == nil || og.Title != "A CoolVibes clip" || og.Image != "https://video.example/images/cover.jpg" {
		t.Fatalf("unexpected OG metadata: %#v", og)
	}
	if twitter == nil || twitter.Type != "player" || twitter.Title != "Fallback title" {
		t.Fatalf("unexpected Twitter metadata: %#v", twitter)
	}
	if discovery == nil || discovery.Host != "video.example" || discovery.Path != "/oembed" {
		t.Fatalf("unexpected discovery URL: %#v", discovery)
	}
}

func TestNormalizeOEmbedSupportsFlexibleDimensions(t *testing.T) {
	var raw rawOEmbedResponse
	if err := json.Unmarshal([]byte(`{
      "version": 1.0,
      "type": "rich",
      "title": "Interactive post",
      "provider_name": "Example",
      "provider_url": "https://provider.example/",
      "thumbnail_url": "/thumb.jpg",
      "width": "640",
      "height": 360.8,
      "html": "<iframe src=\"https://player.example/embed/1\"></iframe>"
    }`), &raw); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	endpoint, _ := url.Parse("https://provider.example/oembed")
	embed := normalizeOEmbed(raw, endpoint)
	if embed == nil {
		t.Fatal("normalizeOEmbed() returned nil")
	}
	if embed.Version != "1.0" || embed.Type != "rich" || embed.Width != 640 || embed.Height != 360 {
		t.Fatalf("unexpected normalized dimensions: %#v", embed)
	}
	if embed.ThumbnailURL != "https://provider.example/thumb.jpg" {
		t.Fatalf("thumbnail URL = %q", embed.ThumbnailURL)
	}
}

func TestNormalizeOEmbedRejectsVideoWithoutHTML(t *testing.T) {
	endpoint, _ := url.Parse("https://provider.example/oembed")
	embed := normalizeOEmbed(rawOEmbedResponse{Type: "video"}, endpoint)
	if embed != nil {
		t.Fatalf("normalizeOEmbed() = %#v, want nil", embed)
	}
}

func TestWildcardOEmbedSchemeMatch(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		want    bool
	}{
		{
			pattern: "https://vimeo.com/*",
			value:   "https://vimeo.com/76979871",
			want:    true,
		},
		{
			pattern: "https://*.example.com/posts/*",
			value:   "https://media.example.com/posts/42",
			want:    true,
		},
		{
			pattern: "https://example.com/videos/*",
			value:   "https://example.com/posts/42",
			want:    false,
		},
	}

	for _, test := range tests {
		if got := wildcardOEmbedSchemeMatch(test.pattern, test.value); got != test.want {
			t.Errorf("wildcardOEmbedSchemeMatch(%q, %q) = %t, want %t", test.pattern, test.value, got, test.want)
		}
	}
}

func TestIsPublicEmbedIPBlocksInternalNetworks(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "127.0.0.1", want: false},
		{value: "10.0.0.1", want: false},
		{value: "169.254.169.254", want: false},
		{value: "100.64.0.1", want: false},
		{value: "::1", want: false},
		{value: "2606:4700:4700::1111", want: true},
		{value: "1.1.1.1", want: true},
	}

	for _, test := range tests {
		if got := isPublicEmbedIP(net.ParseIP(test.value)); got != test.want {
			t.Errorf("isPublicEmbedIP(%q) = %t, want %t", test.value, got, test.want)
		}
	}
}
