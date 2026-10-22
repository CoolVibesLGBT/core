package chat

import (
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var ErrInvalidEmoticon = errors.New("invalid chat sticker or GIF")
var stickerAsset = regexp.MustCompile(`^/animations/telegram/((assets/[A-Za-z0-9_-]+|stickers/[a-f0-9]{16})\.(json|tgs)|packs/[a-f0-9]{16}\.(tgs|webp|webm))$`)

type Emoticon struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	URL    string `json:"url"`
	Emoji  string `json:"emoji,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// Read only the transport-approved emoticon; unrelated post extras remain private.
func ParseEmoticonExtras(values []string) (*Emoticon, error) {
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		return nil, nil
	}
	if len(values) != 1 || len(values[0]) > 8192 {
		return nil, ErrInvalidEmoticon
	}
	var extras struct {
		Emoticon *Emoticon `json:"emoticon"`
	}
	if err := json.Unmarshal([]byte(values[0]), &extras); err != nil {
		return nil, ErrInvalidEmoticon
	}
	e := extras.Emoticon
	if e == nil {
		return nil, nil
	}
	if e.ID == "" || len(e.ID) > 128 || len(e.Emoji) > 64 || e.Width < 0 || e.Height < 0 || e.Width > 4096 || e.Height > 4096 {
		return nil, ErrInvalidEmoticon
	}
	switch e.Kind {
	case "sticker":
		if !stickerAsset.MatchString(e.URL) {
			return nil, ErrInvalidEmoticon
		}
	case "gif":
		u, err := url.Parse(e.URL)
		if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || (u.Hostname() != "media.tenor.com" && !strings.HasSuffix(u.Hostname(), ".tenor.com")) {
			return nil, ErrInvalidEmoticon
		}
	default:
		return nil, ErrInvalidEmoticon
	}
	return e, nil
}
