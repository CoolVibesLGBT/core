package sitemap

import "encoding/xml"

type VideoURLSet struct {
	XMLName xml.Name       `xml:"urlset"`
	Xmlns   string         `xml:"xmlns,attr"`
	VideoNS string         `xml:"xmlns:video,attr"`
	URLs    []VideoURLItem `xml:"url"`
}

type VideoURLItem struct {
	Loc    string      `xml:"loc"`
	Videos []VideoMeta `xml:"video:video"`
}

type VideoMeta struct {
	ThumbnailLoc string `xml:"video:thumbnail_loc"`
	Title        string `xml:"video:title"`
	Description  string `xml:"video:description"`
	ContentLoc   string `xml:"video:content_loc,omitempty"`
	PlayerLoc    string `xml:"video:player_loc,omitempty"`
	Duration     int    `xml:"video:duration,omitempty"`
}
