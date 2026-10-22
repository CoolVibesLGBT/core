package chat

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image/jpeg"
	"strings"
)

// ViewOncePreviewFromExtras exposes only a canonical, tiny JPEG, never an original URL.
func ViewOncePreviewFromExtras(raw []byte) string {
	var extras struct {
		Preview string `json:"view_once_preview"`
	}
	if json.Unmarshal(raw, &extras) != nil {
		return ""
	}
	const prefix = "data:image/jpeg;base64,"
	if len(extras.Preview) > 8192 || !strings.HasPrefix(extras.Preview, prefix) {
		return ""
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(extras.Preview, prefix))
	if err != nil {
		return ""
	}
	config, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 12 || config.Height > 12 {
		return ""
	}
	image, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return ""
	}
	var encoded bytes.Buffer
	if jpeg.Encode(&encoded, image, &jpeg.Options{Quality: 50}) != nil {
		return ""
	}
	return prefix + base64.StdEncoding.EncodeToString(encoded.Bytes())
}
