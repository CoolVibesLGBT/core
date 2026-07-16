package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

const (
	maxEmbedHTMLBytes    = 2 << 20
	maxOEmbedJSONBytes   = 512 << 10
	maxOEmbedHTMLBytes   = 256 << 10
	maxProviderListBytes = 2 << 20
	maxEmbedRedirects    = 4
	providerListTTL      = 24 * time.Hour
)

const oEmbedProviderListURL = "https://oembed.com/providers.json"

var (
	errInvalidEmbedURL = errors.New("invalid or non-public embed URL")
	errEmbedHTMLParse  = errors.New("failed to parse embed HTML")
)

var blockedEmbedNetworks = mustParseEmbedNetworks(
	"0.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"::/128",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
	"ff00::/8",
	"2001:db8::/32",
)

type oEmbedProviderEndpoint struct {
	Schemes []string `json:"schemes"`
	URL     string   `json:"url"`
	Formats []string `json:"formats"`
}

type oEmbedProvider struct {
	ProviderName string                   `json:"provider_name"`
	ProviderURL  string                   `json:"provider_url"`
	Endpoints    []oEmbedProviderEndpoint `json:"endpoints"`
}

var oEmbedProviderCache struct {
	sync.RWMutex
	providers []oEmbedProvider
	expiresAt time.Time
}

func mustParseEmbedNetworks(values ...string) []*net.IPNet {
	networks := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		_, network, err := net.ParseCIDR(value)
		if err != nil {
			panic(err)
		}
		networks = append(networks, network)
	}
	return networks
}

func isPublicEmbedIP(ip net.IP) bool {
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
		return false
	}
	for _, network := range blockedEmbedNetworks {
		if network.Contains(ip) {
			return false
		}
	}
	return true
}

func resolvePublicEmbedIPs(ctx context.Context, hostname string) ([]net.IP, error) {
	hostname = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(hostname)), ".")
	if hostname == "" || hostname == "localhost" || strings.HasSuffix(hostname, ".localhost") || strings.HasSuffix(hostname, ".local") || strings.HasSuffix(hostname, ".internal") {
		return nil, errInvalidEmbedURL
	}

	if literal := net.ParseIP(hostname); literal != nil {
		if !isPublicEmbedIP(literal) {
			return nil, errInvalidEmbedURL
		}
		return []net.IP{literal}, nil
	}

	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, hostname)
	if err != nil || len(addresses) == 0 {
		return nil, fmt.Errorf("%w: host lookup failed", errInvalidEmbedURL)
	}

	publicIPs := make([]net.IP, 0, len(addresses))
	for _, address := range addresses {
		if !isPublicEmbedIP(address.IP) {
			return nil, errInvalidEmbedURL
		}
		publicIPs = append(publicIPs, address.IP)
	}
	return publicIPs, nil
}

func validatePublicEmbedURL(ctx context.Context, target *url.URL) error {
	if target == nil || target.Hostname() == "" || target.User != nil || (target.Scheme != "http" && target.Scheme != "https") {
		return errInvalidEmbedURL
	}
	_, err := resolvePublicEmbedIPs(ctx, target.Hostname())
	return err
}

func dialPublicEmbedAddress(ctx context.Context, network, address string) (net.Conn, error) {
	hostname, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid address", errInvalidEmbedURL)
	}
	addresses, err := resolvePublicEmbedIPs(ctx, hostname)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	var lastErr error
	for _, addressIP := range addresses {
		connection, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(addressIP.String(), port))
		if dialErr == nil {
			return connection, nil
		}
		lastErr = dialErr
	}
	return nil, lastErr
}

func newSafeEmbedHTTPClient() *http.Client {
	transport := &http.Transport{
		DialContext:           dialPublicEmbedAddress,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          20,
		MaxIdleConnsPerHost:   4,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 6 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= maxEmbedRedirects {
				return errors.New("too many embed redirects")
			}
			return validatePublicEmbedURL(request.Context(), request.URL)
		},
	}
}

func fetchEmbedResource(ctx context.Context, client *http.Client, targetURL *url.URL, accept string, limit int64) ([]byte, error) {
	if err := validatePublicEmbedURL(ctx, targetURL); err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", accept)
	request.Header.Set("Accept-Language", "en-US,en;q=0.8")
	request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CoolVibes-oEmbed/1.0; +https://coolvibes.app)")

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("embed provider returned %s", response.Status)
	}

	content, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, errors.New("embed response exceeds size limit")
	}
	return content, nil
}

func parseLinkMetadata(document []byte, documentURL *url.URL) (*MetaData, *MetaData, *url.URL, error) {
	root, err := html.Parse(bytes.NewReader(document))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %v", errEmbedHTMLParse, err)
	}

	og := &MetaData{URL: documentURL.String()}
	twitter := &MetaData{URL: documentURL.String()}
	var discoveryURL *url.URL
	var documentTitle string

	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch strings.ToLower(node.Data) {
			case "meta":
				attributes := htmlAttributes(node)
				property := strings.ToLower(strings.TrimSpace(attributes["property"]))
				name := strings.ToLower(strings.TrimSpace(attributes["name"]))
				content := strings.TrimSpace(attributes["content"])
				applyMetadataValue(og, property, content, true)
				applyMetadataValue(twitter, name, content, false)
			case "link":
				if discoveryURL != nil {
					break
				}
				attributes := htmlAttributes(node)
				if !hasHTMLRel(attributes["rel"], "alternate") || !strings.EqualFold(strings.TrimSpace(attributes["type"]), "application/json+oembed") {
					break
				}
				discoveryURL = resolveEmbedURL(documentURL, attributes["href"])
			case "title":
				if documentTitle == "" && node.FirstChild != nil {
					documentTitle = strings.TrimSpace(node.FirstChild.Data)
				}
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)

	if og.Title == "" {
		og.Title = documentTitle
	}
	if twitter.Title == "" {
		twitter.Title = documentTitle
	}
	og.Image = resolvedEmbedURLString(documentURL, og.Image)
	og.URL = resolvedEmbedURLString(documentURL, og.URL)
	twitter.Image = resolvedEmbedURLString(documentURL, twitter.Image)
	twitter.URL = resolvedEmbedURLString(documentURL, twitter.URL)

	return emptyMetadataAsNil(og), emptyMetadataAsNil(twitter), discoveryURL, nil
}

func htmlAttributes(node *html.Node) map[string]string {
	attributes := make(map[string]string, len(node.Attr))
	for _, attribute := range node.Attr {
		attributes[strings.ToLower(attribute.Key)] = attribute.Val
	}
	return attributes
}

func hasHTMLRel(value, expected string) bool {
	for _, relation := range strings.Fields(strings.ToLower(value)) {
		if relation == expected {
			return true
		}
	}
	return false
}

func applyMetadataValue(metadata *MetaData, key, value string, openGraph bool) {
	if value == "" {
		return
	}
	if openGraph {
		switch key {
		case "og:title":
			metadata.Title = value
		case "og:description":
			metadata.Description = value
		case "og:image", "og:image:url", "og:image:secure_url":
			if metadata.Image == "" {
				metadata.Image = value
			}
		case "og:image:alt":
			metadata.ImageAlt = value
		case "og:url":
			metadata.URL = value
		case "og:site_name":
			metadata.SiteName = value
		case "og:type":
			metadata.Type = value
		case "og:locale":
			metadata.Locale = value
		}
		return
	}

	switch key {
	case "twitter:title":
		metadata.Title = value
	case "twitter:description":
		metadata.Description = value
	case "twitter:image", "twitter:image:src":
		if metadata.Image == "" {
			metadata.Image = value
		}
	case "twitter:image:alt":
		metadata.ImageAlt = value
	case "twitter:site":
		metadata.SiteName = value
	case "twitter:card":
		metadata.Type = value
	}
}

func emptyMetadataAsNil(metadata *MetaData) *MetaData {
	if metadata.Title == "" && metadata.Description == "" && metadata.Image == "" && metadata.SiteName == "" && metadata.Type == "" {
		return nil
	}
	return metadata
}

func resolveEmbedURL(baseURL *url.URL, rawURL string) *url.URL {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil
	}
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	resolvedURL := baseURL.ResolveReference(parsedURL)
	if resolvedURL.User != nil || resolvedURL.Hostname() == "" || (resolvedURL.Scheme != "http" && resolvedURL.Scheme != "https") {
		return nil
	}
	return resolvedURL
}

func resolvedEmbedURLString(baseURL *url.URL, rawURL string) string {
	resolvedURL := resolveEmbedURL(baseURL, rawURL)
	if resolvedURL == nil {
		return ""
	}
	return resolvedURL.String()
}

func findRegisteredOEmbedEndpoint(ctx context.Context, client *http.Client, targetURL *url.URL) *url.URL {
	providers := cachedOEmbedProviders(ctx, client)
	if len(providers) == 0 {
		return nil
	}

	target := targetURL.String()
	for _, provider := range providers {
		for _, endpoint := range provider.Endpoints {
			if !supportsOEmbedJSON(endpoint.Formats) {
				continue
			}
			matched := false
			for _, scheme := range endpoint.Schemes {
				if wildcardOEmbedSchemeMatch(scheme, target) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}

			endpointURL := strings.ReplaceAll(endpoint.URL, "{format}", "json")
			if parsedEndpoint := resolveEmbedURL(targetURL, endpointURL); parsedEndpoint != nil {
				return parsedEndpoint
			}
		}
	}
	return nil
}

func cachedOEmbedProviders(ctx context.Context, client *http.Client) []oEmbedProvider {
	now := time.Now()
	oEmbedProviderCache.RLock()
	if len(oEmbedProviderCache.providers) > 0 && now.Before(oEmbedProviderCache.expiresAt) {
		providers := oEmbedProviderCache.providers
		oEmbedProviderCache.RUnlock()
		return providers
	}
	oEmbedProviderCache.RUnlock()

	oEmbedProviderCache.Lock()
	defer oEmbedProviderCache.Unlock()
	if len(oEmbedProviderCache.providers) > 0 && now.Before(oEmbedProviderCache.expiresAt) {
		return oEmbedProviderCache.providers
	}

	registryURL, err := url.Parse(oEmbedProviderListURL)
	if err != nil {
		return oEmbedProviderCache.providers
	}
	registryContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	payload, err := fetchEmbedResource(registryContext, client, registryURL, "application/json", maxProviderListBytes)
	if err != nil {
		return oEmbedProviderCache.providers
	}

	var providers []oEmbedProvider
	if err := json.Unmarshal(payload, &providers); err != nil {
		return oEmbedProviderCache.providers
	}
	oEmbedProviderCache.providers = providers
	oEmbedProviderCache.expiresAt = now.Add(providerListTTL)
	return oEmbedProviderCache.providers
}

func supportsOEmbedJSON(formats []string) bool {
	if len(formats) == 0 {
		return true
	}
	for _, format := range formats {
		if strings.EqualFold(strings.TrimSpace(format), "json") {
			return true
		}
	}
	return false
}

func wildcardOEmbedSchemeMatch(pattern, value string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	value = strings.ToLower(strings.TrimSpace(value))
	if pattern == "" || value == "" {
		return false
	}

	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == value
	}
	if parts[0] != "" && !strings.HasPrefix(value, parts[0]) {
		return false
	}
	if last := parts[len(parts)-1]; last != "" && !strings.HasSuffix(value, last) {
		return false
	}

	position := 0
	for index, part := range parts {
		if part == "" {
			continue
		}
		if index == 0 {
			position = len(part)
			continue
		}
		relativeIndex := strings.Index(value[position:], part)
		if relativeIndex < 0 {
			return false
		}
		position += relativeIndex + len(part)
	}
	return true
}

type rawOEmbedResponse struct {
	Version         json.RawMessage `json:"version"`
	Type            string          `json:"type"`
	Title           string          `json:"title"`
	AuthorName      string          `json:"author_name"`
	AuthorURL       string          `json:"author_url"`
	ProviderName    string          `json:"provider_name"`
	ProviderURL     string          `json:"provider_url"`
	CacheAge        json.RawMessage `json:"cache_age"`
	ThumbnailURL    string          `json:"thumbnail_url"`
	ThumbnailWidth  json.RawMessage `json:"thumbnail_width"`
	ThumbnailHeight json.RawMessage `json:"thumbnail_height"`
	URL             string          `json:"url"`
	HTML            string          `json:"html"`
	Width           json.RawMessage `json:"width"`
	Height          json.RawMessage `json:"height"`
}

func fetchOEmbed(ctx context.Context, client *http.Client, discoveryURL, targetURL *url.URL) *OEmbedData {
	endpoint := *discoveryURL
	query := endpoint.Query()
	if query.Get("url") == "" {
		query.Set("url", targetURL.String())
	}
	if query.Get("format") == "" {
		query.Set("format", "json")
	}
	query.Set("maxwidth", "760")
	query.Set("maxheight", "640")
	endpoint.RawQuery = query.Encode()

	payload, err := fetchEmbedResource(ctx, client, &endpoint, "application/json+oembed,application/json", maxOEmbedJSONBytes)
	if err != nil {
		return nil
	}

	var raw rawOEmbedResponse
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil
	}
	return normalizeOEmbed(raw, &endpoint)
}

func normalizeOEmbed(raw rawOEmbedResponse, endpoint *url.URL) *OEmbedData {
	embedType := strings.ToLower(strings.TrimSpace(raw.Type))
	if embedType != "photo" && embedType != "video" && embedType != "link" && embedType != "rich" {
		return nil
	}

	embed := &OEmbedData{
		Version:         normalizeOEmbedVersion(raw.Version),
		Type:            embedType,
		Title:           strings.TrimSpace(raw.Title),
		AuthorName:      strings.TrimSpace(raw.AuthorName),
		AuthorURL:       resolvedEmbedURLString(endpoint, raw.AuthorURL),
		ProviderName:    strings.TrimSpace(raw.ProviderName),
		ProviderURL:     resolvedEmbedURLString(endpoint, raw.ProviderURL),
		CacheAge:        flexibleJSONInt(raw.CacheAge),
		ThumbnailURL:    resolvedEmbedURLString(endpoint, raw.ThumbnailURL),
		ThumbnailWidth:  flexibleJSONInt(raw.ThumbnailWidth),
		ThumbnailHeight: flexibleJSONInt(raw.ThumbnailHeight),
		URL:             resolvedEmbedURLString(endpoint, raw.URL),
		HTML:            strings.TrimSpace(raw.HTML),
		Width:           flexibleJSONInt(raw.Width),
		Height:          flexibleJSONInt(raw.Height),
	}
	if embed.Version == "" {
		embed.Version = "1.0"
	}
	if len(embed.HTML) > maxOEmbedHTMLBytes {
		return nil
	}

	switch embed.Type {
	case "photo":
		if embed.URL == "" {
			return nil
		}
	case "video", "rich":
		if embed.HTML == "" {
			return nil
		}
	}

	return embed
}

func normalizeOEmbedVersion(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if json.Unmarshal(raw, &value) == nil {
		return strings.TrimSpace(value)
	}
	var number json.Number
	if json.Unmarshal(raw, &number) == nil {
		return number.String()
	}
	return ""
}

func flexibleJSONInt(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var number json.Number
	if json.Unmarshal(raw, &number) == nil {
		if integer, err := strconv.Atoi(number.String()); err == nil {
			return max(integer, 0)
		}
		if decimal, err := strconv.ParseFloat(number.String(), 64); err == nil {
			return max(int(decimal), 0)
		}
	}
	var value string
	if json.Unmarshal(raw, &value) == nil {
		if decimal, err := strconv.ParseFloat(value, 64); err == nil {
			return max(int(decimal), 0)
		}
	}
	return 0
}
