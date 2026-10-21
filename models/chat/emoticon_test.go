package chat

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestEmoticonExtrasValidation(t *testing.T) {
	for _, tc := range []struct {
		name, kind, url string
		valid           bool
	}{
		{"original animated greeting", "sticker", "/animations/telegram/stickers/505b5ae14b0e1be1.tgs", true},
		{"bundled animation", "sticker", "/animations/telegram/assets/UtyanSearch.json", true},
		{"gif", "gif", "https://media.tenor.com/test/hello.gif", true},
		{"gif hostname spoof", "gif", "https://media.tenor.com.attacker.invalid/hello.gif", false},
		{"http gif", "gif", "http://media.tenor.com/hello.gif", false},
		{"path traversal", "sticker", "/animations/telegram/stickers/../../secret.json", false},
		{"remote sticker", "sticker", "https://example.invalid/sticker.json", false},
		{"unknown kind", "file", "https://media.tenor.com/a.gif", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, _ := json.Marshal(map[string]any{"emoticon": Emoticon{Kind: tc.kind, ID: "fixture", URL: tc.url}, "private": "never projected"})
			result, err := ParseEmoticonExtras([]string{string(raw)})
			if tc.valid && (err != nil || result == nil) {
				t.Fatalf("valid emoticon rejected: %v", err)
			}
			if !tc.valid && !errors.Is(err, ErrInvalidEmoticon) {
				t.Fatalf("invalid emoticon accepted: %+v %v", result, err)
			}
		})
	}
	if e, err := ParseEmoticonExtras([]string{`{"private":"secret"}`}); err != nil || e != nil {
		t.Fatal("unrelated extras must not become an emoticon")
	}
}
