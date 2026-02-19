package sitemap

import "encoding/xml"

type NewsURLSet struct {
	XMLName xml.Name  `xml:"urlset"`
	Xmlns   string    `xml:"xmlns,attr"`
	News    string    `xml:"xmlns:news,attr"`
	URLs    []NewsURL `xml:"url"`
}

type NewsURL struct {
	Loc  string   `xml:"loc"`
	News NewsMeta `xml:"news:news"`
}

type NewsMeta struct {
	Publication     Publication `xml:"news:publication"`
	PublicationDate string      `xml:"news:publication_date"`
	Title           string      `xml:"news:title"`
}

type Publication struct {
	Name     string `xml:"news:name"`
	Language string `xml:"news:language"`
}
