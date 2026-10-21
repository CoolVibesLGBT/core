package repositories

import (
	"context"
	"core/application/ports"
	"core/application/usecases"
	"core/helpers"
	"core/models"
	"core/models/chat"
	"core/models/media"
	"core/models/post"
	"core/models/post/payloads"
	modelutils "core/models/utils"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

func TestChatForwardIntegration(t *testing.T) {
	t.Chdir(t.TempDir())
	db := privatePhotoIntegrationDB(t)
	if err := db.AutoMigrate(&post.Post{}, &chat.Chat{}, &chat.ChatParticipant{}, &chat.MessageView{}, &payloads.Event{}, &payloads.Poll{}, &models.Hashtag{}); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{"CREATE TABLE locations (id uuid, contentable_id uuid, contentable_type text, deleted_at timestamptz)", "CREATE TABLE post_hashtags (post_id uuid, hashtag_id uuid)"} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	author, recipient := privatePhotoTestUsers(t, db)
	author.DefaultLanguage = "en"
	node, err := helpers.NewDefaultNode()
	if err != nil {
		t.Fatal(err)
	}
	mediaRepo := NewMediaRepository(db, node)
	repo := NewChatRepository(db, node, NewPostRepository(db, node, mediaRepo, nil, nil), nil, nil)
	ctx := context.Background()
	source := chat.Chat{ID: uuid.New(), Type: chat.ChatTypePrivate, CreatorID: author.ID}
	target := chat.Chat{ID: uuid.New(), Type: chat.ChatTypeGroup, CreatorID: author.ID}
	if err := db.Omit(clause.Associations).Create(&[]chat.Chat{source, target}).Error; err != nil {
		t.Fatal(err)
	}
	members := []chat.ChatParticipant{{ID: uuid.New(), ChatID: source.ID, UserID: author.ID}, {ID: uuid.New(), ChatID: target.ID, UserID: author.ID}, {ID: uuid.New(), ChatID: target.ID, UserID: recipient.ID}}
	if err := db.Create(&members).Error; err != nil {
		t.Fatal(err)
	}
	contentType := string(post.PostKindChat)
	text := modelutils.LocalizedString{"en": "original text"}
	originals := []post.Post{
		{ID: uuid.New(), PublicID: node.Generate().Int64(), AuthorID: author.ID, ContentableType: &contentType, ContentableID: &source.ID, PostKind: post.PostKindMessage, Content: &text},
		{ID: uuid.New(), PublicID: node.Generate().Int64(), AuthorID: author.ID, ContentableType: &contentType, ContentableID: &source.ID, PostKind: post.PostKindMessage, Extras: datatypes.JSON(`{"emoticon":{"kind":"sticker","id":"505b5ae14b0e1be1","url":"/animations/telegram/stickers/505b5ae14b0e1be1.tgs","emoji":"👋"}}`)},
		{ID: uuid.New(), PublicID: node.Generate().Int64(), AuthorID: author.ID, ContentableType: &contentType, ContentableID: &source.ID, PostKind: post.PostKindMessage},
	}
	if err := db.Omit(clause.Associations).Create(&originals).Error; err != nil {
		t.Fatal(err)
	}
	file := modelutils.FileMetadata{ID: uuid.New(), URL: "/fixture.png", StoragePath: "fixture.png", MimeType: "image/png", Name: "fixture.png"}
	if err := db.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	attachment := media.Media{ID: uuid.New(), PublicID: node.Generate().Int64(), FileID: file.ID, OwnerID: originals[2].ID, OwnerType: media.OwnerPost, UserID: author.ID, Role: media.RoleChatImage}
	if err := db.Omit(clause.Associations).Create(&attachment).Error; err != nil {
		t.Fatal(err)
	}
	ids := []uuid.UUID{originals[2].ID, originals[0].ID, originals[1].ID}
	forwarded, err := repo.ForwardMessages(ctx, &author, source.ID, target.ID, ids, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(forwarded) != 3 || forwarded[0].Content == nil || (*forwarded[0].Content)["en"] != "original text" {
		t.Fatal("text/order missing")
	}
	emoticon, err := chat.ParseEmoticonExtras([]string{string(forwarded[1].Extras)})
	if err != nil || emoticon == nil || emoticon.ID != "505b5ae14b0e1be1" {
		t.Fatal("sticker missing")
	}
	if len(forwarded[2].Attachments) != 1 || forwarded[2].Attachments[0].ID == attachment.ID || forwarded[2].Attachments[0].OwnerID != forwarded[2].ID || forwarded[2].Attachments[0].FileID != file.ID || forwarded[2].Attachments[0].IsPublic {
		t.Fatal("forwarded media ownership/file reference incorrect")
	}
	// The recipient has no membership in the original chat. The visible forwarded
	// attachment grants access independently and stops doing so when hidden.
	access := usecases.NewMediaAccessService(mediaRepo)
	decision, err := access.Authorize(ctx, file.StoragePath, &recipient.PublicID)
	if err != nil || !decision.Allowed || decision.Public {
		t.Fatalf("forwarded recipient file access: %+v %v", decision, err)
	}
	if err := db.Model(&chat.ChatParticipant{}).Where("chat_id = ? AND user_id = ?", target.ID, recipient.ID).Update("cleared_at", time.Now().UTC().Add(time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	decision, err = access.Authorize(ctx, file.StoragePath, &recipient.PublicID)
	if err != nil || decision.Allowed {
		t.Fatalf("cleared forwarded attachment still accessible: %+v %v", decision, err)
	}
	if err := db.Model(&chat.ChatParticipant{}).Where("chat_id = ? AND user_id = ?", target.ID, recipient.ID).Update("cleared_at", nil).Error; err != nil {
		t.Fatal(err)
	}
	for i, item := range forwarded {
		origin := chat.ForwardOriginFromExtras(item.Extras)
		if item.ID == originals[i].ID || item.AuthorID != author.ID || item.ContentableID == nil || *item.ContentableID != target.ID || origin == nil || origin.Name != author.UserName || !origin.Date.Equal(originals[i].CreatedAt) {
			t.Fatal("forward origin or target incorrect")
		}
	}
	var destination chat.Chat
	if err := db.Take(&destination, "id = ?", target.ID).Error; err != nil {
		t.Fatal(err)
	}
	if destination.LastMessageID == nil || *destination.LastMessageID != forwarded[2].ID {
		t.Fatal("last message not updated")
	}
	var member chat.ChatParticipant
	if err := db.Take(&member, "chat_id = ? AND user_id = ?", target.ID, recipient.ID).Error; err != nil {
		t.Fatal(err)
	}
	if member.UnreadCount != 3 {
		t.Fatalf("unread count: %d", member.UnreadCount)
	}
	page, err := repo.ListChatMessages(ctx, ports.ChatMessageListQuery{ChatID: target.ID, UserID: recipient.ID, Limit: 10})
	if err != nil || len(page.Messages) != 3 || chat.ForwardOriginFromExtras(page.Messages[0].Extras) == nil {
		t.Fatalf("history reload failed: %v", err)
	}
	// Recipients can forward onward; the original attribution survives the second hop.
	onward, err := repo.ForwardMessages(ctx, &recipient, target.ID, target.ID, []uuid.UUID{forwarded[0].ID}, "")
	if err != nil || len(onward) != 1 || chat.ForwardOriginFromExtras(onward[0].Extras).Name != author.UserName {
		t.Fatalf("onward forwarding: %v", err)
	}
	captioned, err := repo.ForwardMessages(ctx, &author, source.ID, target.ID, []uuid.UUID{originals[0].ID}, "A comment")
	if err != nil || len(captioned) != 2 || captioned[0].Content == nil || (*captioned[0].Content)["en"] != "A comment" || chat.ForwardOriginFromExtras(captioned[0].Extras) != nil || chat.ForwardOriginFromExtras(captioned[1].Extras) == nil {
		t.Fatalf("atomic comment + forward: %v", err)
	}
	if _, err := repo.ForwardMessages(ctx, &recipient, source.ID, target.ID, []uuid.UUID{originals[0].ID}, ""); !errors.Is(err, chat.ErrNotParticipant) {
		t.Fatalf("unauthorized source allowed: %v", err)
	}
	// The entire batch must roll back when any requested message is protected.
	if err := db.Model(&post.Post{}).Where("id = ?", originals[1].ID).Update("view_once", true).Error; err != nil {
		t.Fatal(err)
	}
	var before, after int64
	db.Model(&post.Post{}).Where("contentable_id = ?", target.ID).Count(&before)
	if _, err := repo.ForwardMessages(ctx, &author, source.ID, target.ID, ids, ""); !errors.Is(err, chat.ErrPermissionDenied) {
		t.Fatalf("view-once forward allowed: %v", err)
	}
	db.Model(&post.Post{}).Where("contentable_id = ?", target.ID).Count(&after)
	if before != after {
		t.Fatal("partially forwarded failed batch")
	}
	if _, err := repo.ForwardMessages(ctx, &author, source.ID, target.ID, []uuid.UUID{originals[0].ID, originals[0].ID}, ""); !errors.Is(err, chat.ErrMessageNotFound) {
		t.Fatalf("duplicate IDs: %v", err)
	}
	if err := repo.DeleteMessageForUser(ctx, &author, source.ID, author.ID, originals[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ForwardMessages(ctx, &author, source.ID, target.ID, []uuid.UUID{originals[0].ID}, ""); !errors.Is(err, chat.ErrMessageNotFound) {
		t.Fatalf("deleted forward allowed: %v", err)
	}
	now := time.Now().UTC().Add(time.Second)
	if err := db.Model(&chat.ChatParticipant{}).Where("chat_id = ? AND user_id = ?", source.ID, author.ID).Update("cleared_at", now).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ForwardMessages(ctx, &author, source.ID, target.ID, []uuid.UUID{originals[2].ID}, ""); !errors.Is(err, chat.ErrMessageNotFound) {
		t.Fatalf("cleared history forward allowed: %v", err)
	}
	// A normal send cannot forge the server-owned forwarded attribution.
	forged, err := repo.AddMessageToChat(ctx, map[string][]string{"chat_id": {source.ID.String()}, "content": {"ordinary"}, "extras": {`{"forwarded_from":{"name":"forged","date":"2020-01-01T00:00:00Z"}}`}}, nil, &author)
	if err != nil || chat.ForwardOriginFromExtras(forged.Extras) != nil {
		t.Fatalf("forged origin accepted: %v", err)
	}
	// Removing the original attachment changes which association is returned by
	// the file lookup; access must still follow visibility in the target chat.
	if err := db.Delete(&attachment).Error; err != nil {
		t.Fatal(err)
	}
	decision, err = access.Authorize(ctx, file.StoragePath, &recipient.PublicID)
	if err != nil || !decision.Allowed {
		t.Fatalf("forwarded file without original attachment: %+v %v", decision, err)
	}
	if err := db.Model(&chat.ChatParticipant{}).Where("chat_id = ? AND user_id = ?", target.ID, recipient.ID).Update("cleared_at", time.Now().UTC().Add(time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	decision, err = access.Authorize(ctx, file.StoragePath, &recipient.PublicID)
	if err != nil || decision.Allowed {
		t.Fatalf("cleared target association grants file access: %+v %v", decision, err)
	}
	if err := repo.SetChatMuted(ctx, &author, target.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetChatMuted(ctx, &recipient, source.ID, true); !errors.Is(err, chat.ErrNotParticipant) {
		t.Fatalf("outsider mute: %v", err)
	}
	for _, format := range []string{"tgs", "webp", "webm"} {
		raw := `{"emoticon":{"kind":"sticker","id":"8cc5a62e6fc333e1","url":"/animations/telegram/packs/8cc5a62e6fc333e1.` + format + `","emoji":"😂","width":512,"height":512}}`
		message, err := repo.AddMessageToChat(ctx, map[string][]string{"chat_id": {source.ID.String()}, "content": {""}, "extras": {raw}, "client_id": {"sticker-" + format}}, nil, &author)
		if err != nil {
			t.Fatalf("standalone %s sticker send: %v", format, err)
		}
		var stored post.Post
		if err := db.First(&stored, "id = ?", message.ID).Error; err != nil {
			t.Fatal(err)
		}
		emoticon, err := chat.ParseEmoticonExtras([]string{string(stored.Extras)})
		if err != nil || emoticon == nil || emoticon.URL != "/animations/telegram/packs/8cc5a62e6fc333e1."+format {
			t.Fatalf("%s sticker did not persist: %+v %v", format, emoticon, err)
		}
	}
}
