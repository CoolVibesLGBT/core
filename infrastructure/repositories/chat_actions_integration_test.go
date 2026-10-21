package repositories

import (
	"context"
	"core/application/ports"
	"core/helpers"
	"core/models"
	"core/models/chat"
	"core/models/post"
	"core/models/post/payloads"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
	"testing"
	"time"
)

func TestChatEditAndReplyIntegration(t *testing.T) {
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
	entity := chat.Chat{ID: uuid.New(), Type: chat.ChatTypePrivate, CreatorID: author.ID}
	if err := db.Omit(clause.Associations).Create(&entity).Error; err != nil {
		t.Fatal(err)
	}
	participant := chat.ChatParticipant{ID: uuid.New(), ChatID: entity.ID, UserID: author.ID, JoinedAt: time.Now().UTC()}
	if err := db.Create(&participant).Error; err != nil {
		t.Fatal(err)
	}
	node, err := helpers.NewDefaultNode()
	if err != nil {
		t.Fatal(err)
	}
	mediaRepo := NewMediaRepository(db, node)
	repo := NewChatRepository(db, node, NewPostRepository(db, node, mediaRepo, nil, nil), nil, nil)
	ctx := context.Background()
	values := map[string][]string{"chat_id": {entity.ID.String()}, "content": {"original"}}
	original, err := repo.AddMessageToChat(ctx, values, nil, &author)
	if err != nil {
		t.Fatal(err)
	}
	edited, err := repo.EditMessage(ctx, &author, entity.ID, original.ID, "edited text")
	if err != nil {
		t.Fatal(err)
	}
	if edited.Content == nil || (*edited.Content)["en"] != "edited text" {
		t.Fatal("edit did not persist")
	}
	if _, err := repo.EditMessage(ctx, &recipient, entity.ID, original.ID, "intrusion"); !errors.Is(err, chat.ErrNotParticipant) {
		t.Fatalf("nonparticipant edit: %v", err)
	}
	values["reply_to"] = []string{original.ID.String()}
	values["content"] = []string{"reply"}
	reply, err := repo.AddMessageToChat(ctx, values, nil, &author)
	if err != nil {
		t.Fatal(err)
	}
	if reply.ParentID == nil || *reply.ParentID != original.ID {
		t.Fatal("reply link was not stored")
	}
	page, err := repo.ListChatMessages(ctx, ports.ChatMessageListQuery{ChatID: entity.ID, UserID: author.ID, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Messages) != 2 || page.Messages[1].Parent == nil || page.Messages[1].Parent.ID != original.ID {
		t.Fatal("reply parent missing after history reload")
	}
	values["reply_to"] = []string{uuid.NewString()}
	if _, err := repo.AddMessageToChat(ctx, values, nil, &author); !errors.Is(err, chat.ErrInvalidReply) {
		t.Fatalf("unavailable reply accepted: %v", err)
	}
	if err := repo.DeleteMessageForUser(ctx, &author, entity.ID, author.ID, original.ID); err != nil {
		t.Fatal(err)
	}
	values["reply_to"] = []string{original.ID.String()}
	if _, err := repo.AddMessageToChat(ctx, values, nil, &author); !errors.Is(err, chat.ErrInvalidReply) {
		t.Fatalf("deleted reply accepted: %v", err)
	}
	if _, err := repo.EditMessage(ctx, &author, entity.ID, original.ID, "resurrect"); !errors.Is(err, chat.ErrMessageNotFound) {
		t.Fatalf("deleted edit accepted: %v", err)
	}
	other := chat.ChatParticipant{ID: uuid.New(), ChatID: entity.ID, UserID: recipient.ID, JoinedAt: time.Now().UTC()}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.EditMessage(ctx, &recipient, entity.ID, reply.ID, "not mine"); !errors.Is(err, chat.ErrPermissionDenied) {
		t.Fatalf("another author's edit accepted: %v", err)
	}
}
