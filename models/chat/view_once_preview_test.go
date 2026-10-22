package chat

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"testing"
)

func TestViewOncePreviewRejectsOriginalMedia(t *testing.T) {
	for _, size := range []int{12, 13, 100} {
		var data bytes.Buffer
		if err := jpeg.Encode(&data, image.NewRGBA(image.Rect(0, 0, size, size)), nil); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(map[string]string{"view_once_preview": "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data.Bytes())})
		got := ViewOncePreviewFromExtras(raw)
		if (got != "") != (size == 12) {
			t.Fatalf("size %d accepted=%v", size, got != "")
		}
	}
	for _, raw := range []string{`{}`, `{"view_once_preview":"https://example.invalid/original.jpg"}`, `{"view_once_preview":"data:image/jpeg;base64,AAAA"}`} {
		if ViewOncePreviewFromExtras([]byte(raw)) != "" {
			t.Fatal("unsafe preview accepted")
		}
	}
}
