package sitemap

import (
	"encoding/xml"
	"time"
)

type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	URLs    []URL    `xml:"url"`
}

type URL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

type SitemapIndex struct {
	XMLName  xml.Name      `xml:"sitemapindex"`
	Xmlns    string        `xml:"xmlns,attr"`
	Sitemaps []SitemapItem `xml:"sitemap"`
}

type SitemapItem struct {
	Loc       string     `xml:"loc"`
	LastMod   string     `xml:"lastmod,omitempty"`
	Published *time.Time // opsiyonel

}
