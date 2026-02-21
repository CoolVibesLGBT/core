package sitemap

import (
	"encoding/xml"
	"strings"
	"time"
)

const (
	DefaultXMLNS      = "http://www.sitemaps.org/schemas/sitemap/0.9"
	MaxURLsPerSitemap = 50000
)

type SitemapBuilder struct {
	baseURL string
}

func NewSitemapBuilder(baseURL string) *SitemapBuilder {
	baseURL = strings.TrimRight(baseURL, "/")
	return &SitemapBuilder{
		baseURL: baseURL,
	}
}

func (b *SitemapBuilder) BuildLoc(path string) string {
	path = strings.TrimLeft(path, "/")
	return b.baseURL + "/" + path
}

func (b *SitemapBuilder) FormatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func (b *SitemapBuilder) calculatePriority(published *time.Time) (string, string) {

	if published == nil {
		return "0.5", "weekly"
	}

	age := time.Since(*published)

	switch {
	case age <= 12*time.Hour:
		return "1.0", "hourly"
	case age <= 24*time.Hour:
		return "0.9", "hourly"
	case age <= 7*24*time.Hour:
		return "0.8", "daily"
	case age <= 30*24*time.Hour:
		return "0.7", "daily"
	default:
		return "0.5", "weekly"
	}
}

func (b *SitemapBuilder) BuildURLSitemap(items []SitemapItem) ([]byte, error) {

	urlSet := URLSet{
		Xmlns: DefaultXMLNS,
	}

	for _, item := range items {

		priority, freq := b.calculatePriority(item.Published)

		lastMod := item.LastMod
		if lastMod == "" && item.Published != nil {
			lastMod = b.FormatTime(*item.Published)
		}

		urlSet.URLs = append(urlSet.URLs, URL{
			Loc:        item.Loc,
			LastMod:    lastMod,
			Priority:   priority,
			ChangeFreq: freq,
		})
	}

	output, err := xml.MarshalIndent(urlSet, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), output...), nil
}

func (b *SitemapBuilder) BuildSplitSitemaps(items []SitemapItem) ([][]byte, error) {

	var result [][]byte

	for i := 0; i < len(items); i += MaxURLsPerSitemap {

		end := i + MaxURLsPerSitemap
		if end > len(items) {
			end = len(items)
		}

		xmlPart, err := b.BuildURLSitemap(items[i:end])
		if err != nil {
			return nil, err
		}

		result = append(result, xmlPart)
	}

	return result, nil
}
