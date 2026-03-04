package handlers

import (
	"core/constants"
	"core/utils"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/net/html"
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

type EmbedResponse struct {
	OGMeta      *MetaData `json:"og,omitempty"`
	TwitterMeta *MetaData `json:"twitter,omitempty"`
}

func HandleLinkPreview() fiber.Handler {
	return func(c fiber.Ctx) error {

		rawURL := c.FormValue("url") // veya c.Query("url") GET param için

		if rawURL == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidForm)
		}

		decodedURL, err := url.QueryUnescape(rawURL)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidForm)
		}

		resp, err := http.Get(decodedURL)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrOGFetchFailed)
		}

		defer func() {
			if cerr := resp.Body.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}()

		doc, err := html.Parse(resp.Body)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrHTMLParseFailed)
		}

		og := &MetaData{}
		twitter := &MetaData{}

		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "meta" {
				var prop, name, content string
				for _, attr := range n.Attr {
					switch attr.Key {
					case "property":
						prop = attr.Val
					case "name":
						name = attr.Val
					case "content":
						content = attr.Val
					}
				}

				// OG meta
				switch prop {
				case "og:title":
					og.Title = content
				case "og:description":
					og.Description = content
				case "og:image":
					og.Image = content
				case "og:image:alt":
					og.ImageAlt = content
				case "og:url":
					og.URL = content
				case "og:site_name":
					og.SiteName = content
				case "og:type":
					og.Type = content
				case "og:locale":
					og.Locale = content
				}

				// Twitter meta
				switch name {
				case "twitter:title":
					twitter.Title = content
				case "twitter:description":
					twitter.Description = content
				case "twitter:image":
					twitter.Image = content
				case "twitter:image:alt":
					twitter.ImageAlt = content
				case "twitter:site":
					twitter.SiteName = content
				}
			}

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}

		f(doc)

		return utils.SendSuccess(c, fiber.StatusOK, EmbedResponse{
			OGMeta:      og,
			TwitterMeta: twitter,
		})

	}
}
