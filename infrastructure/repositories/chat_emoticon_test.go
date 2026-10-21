package repositories

import (
	"core/models/chat"
	"errors"
	"testing"
)

func TestEmoticonMessageFormWithoutText(t *testing.T) {
	values := map[string][]string{"extras": {`{"emoticon":{"kind":"sticker","id":"greeting","url":"/animations/telegram/stickers/505b5ae14b0e1be1.tgs"}}`}}
	if err := validateMessageForm(values, nil); err != nil {
		t.Fatalf("standalone sticker rejected: %v", err)
	}
	values["view_once"] = []string{"true"}
	if err := validateMessageForm(values, nil); !errors.Is(err, chat.ErrInvalidViewOnce) {
		t.Fatalf("view-once must require actual image upload: %v", err)
	}
}
