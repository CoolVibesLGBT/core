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

	Title       string `xml:"video:title"`
	Description string `xml:"video:description"`

	ContentLoc string `xml:"video:content_loc,omitempty"`
	PlayerLoc  string `xml:"video:player_loc,omitempty"`

	Duration int `xml:"video:duration,omitempty"`

	ExpirationDate string `xml:"video:expiration_date,omitempty"`

	Rating    float64 `xml:"video:rating,omitempty"`
	ViewCount int64   `xml:"video:view_count,omitempty"`

	PublicationDate string `xml:"video:publication_date,omitempty"`

	FamilyFriendly string `xml:"video:family_friendly,omitempty"`

	Restriction *VideoRestriction `xml:"video:restriction,omitempty"`

	Price *VideoPrice `xml:"video:price,omitempty"`

	RequiresSubscription string `xml:"video:requires_subscription,omitempty"`

	Uploader *VideoUploader `xml:"video:uploader,omitempty"`

	Live string `xml:"video:live,omitempty"`
}

type VideoRestriction struct {
	Relationship string `xml:"relationship,attr"`
	Value        string `xml:",chardata"`
}

type VideoPrice struct {
	Currency string `xml:"currency,attr"`
	Value    string `xml:",chardata"`
}

type VideoUploader struct {
	Info string `xml:"info,attr"`
	Name string `xml:",chardata"`
}
