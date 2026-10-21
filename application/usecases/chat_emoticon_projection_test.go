package usecases

import (
	"core/models/post"
	"encoding/json"
	"gorm.io/datatypes"
	"strings"
	"testing"
)

func TestEmoticonProjectionDoesNotExposePrivateExtras(t *testing.T) {
	message := post.Post{Extras: datatypes.JSON(`{"emoticon":{"kind":"sticker","id":"greeting","url":"/animations/telegram/stickers/505b5ae14b0e1be1.tgs"},"private_extra":"must-never-leak"}`)}
	view := chatMessageProjection(&message, nil, 0)
	if view.Emoticon == nil || view.Emoticon.ID != "greeting" {
		t.Fatal("validated emoticon not projected")
	}
	raw, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "must-never-leak") {
		t.Fatal("unrelated private extras leaked")
	}
	message.ContentHidden = true
	if chatMessageProjection(&message, nil, 0).Emoticon != nil {
		t.Fatal("unopened disappearing emoticon leaked")
	}
}
