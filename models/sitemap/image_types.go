package sitemap

import "encoding/xml"

type ImageURLSet struct {
	XMLName xml.Name       `xml:"urlset"`
	Xmlns   string         `xml:"xmlns,attr"`
	ImageNS string         `xml:"xmlns:image,attr"`
	URLs    []ImageURLItem `xml:"url"`
}

type ImageURLItem struct {
	Loc    string       `xml:"loc"`
	Images []ImageEntry `xml:"image:image"`
}

type ImageEntry struct {
	Loc     string `xml:"image:loc"`
	Caption string `xml:"image:caption,omitempty"`
	Title   string `xml:"image:title,omitempty"`
}
