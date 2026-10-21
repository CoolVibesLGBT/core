package repositories

import (
	"context"
	"core/application/ports"
	"core/models/chat"
	"core/models/post"
	modelutils "core/models/utils"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

func TestOpenMessageRespectsDeletionAndClearedHistoryIntegration(t *testing.T) {
	for _, timed := range []bool{true, false} {
		kind := "reply parent"
		if timed {
			kind = "timed open"
		}
		for _, visibility := range []string{"visible", "deleted for me", "deleted for all", "cleared history"} {
			t.Run(kind+"/"+visibility, func(t *testing.T) {
				db := privatePhotoIntegrationDB(t)
				if err := db.AutoMigrate(&post.Post{}, &chat.Chat{}, &chat.ChatParticipant{}, &chat.MessageView{}); err != nil {
					t.Fatal(err)
				}
				author, recipient := privatePhotoTestUsers(t, db)
				now := time.Now().UTC().Truncate(time.Microsecond)
				entity := chat.Chat{ID: uuid.New(), Type: chat.ChatTypePrivate, CreatorID: author.ID}
				if err := db.Omit(clause.Associations).Create(&entity).Error; err != nil {
					t.Fatal(err)
				}
				participants := []chat.ChatParticipant{
					{ID: uuid.New(), ChatID: entity.ID, UserID: author.ID, Role: chat.ParticipantRoleOwner, JoinedAt: now.Add(-time.Hour)},
					{ID: uuid.New(), ChatID: entity.ID, UserID: recipient.ID, Role: chat.ParticipantRoleMember, JoinedAt: now.Add(-time.Hour)},
				}
				if err := db.Create(&participants).Error; err != nil {
					t.Fatal(err)
				}
				duration := 60
				contentableType := string(post.PostKindChat)
				message := post.Post{
					ID: uuid.New(), PublicID: 100, AuthorID: author.ID, PostKind: post.PostKindMessage,
					ContentableID: &entity.ID, ContentableType: &contentableType,
					Content:          modelutils.MakeLocalizedString("en", "removed secret"),
					ExpiresInSeconds: &duration, CreatedAt: now.Add(-time.Minute),
				}
				if !timed {
					message.ExpiresInSeconds = nil
				}
				if err := db.Omit(clause.Associations).Create(&message).Error; err != nil {
					t.Fatal(err)
				}
				repo := &ChatRepository{db: db}
				ctx := context.Background()
				var err error
				switch visibility {
				case "deleted for me":
					err = repo.DeleteMessageForUser(ctx, &recipient, entity.ID, recipient.ID, message.ID)
				case "deleted for all":
					err = repo.DeleteMessageForAll(ctx, &author, entity.ID, author.ID, message.ID)
				case "cleared history":
					err = repo.DeleteChatHistoryForUser(ctx, &recipient, entity.ID)
				}
				if err != nil {
					t.Fatal(err)
				}
				if !timed {
					if err := db.Exec("CREATE TABLE locations (id uuid, contentable_id uuid, contentable_type text, deleted_at timestamptz)").Error; err != nil {
						t.Fatal(err)
					}
					reply := message
					reply.ID, reply.PublicID, reply.ParentID = uuid.New(), 101, &message.ID
					reply.CreatedAt = now.Add(time.Second)
					reply.Content = modelutils.MakeLocalizedString("en", "reply to removed message")
					if err := db.Omit(clause.Associations).Create(&reply).Error; err != nil {
						t.Fatal(err)
					}
					page, err := repo.ListChatMessages(ctx, ports.ChatMessageListQuery{ChatID: entity.ID, UserID: recipient.ID, Limit: 10})
					if err != nil {
						t.Fatal(err)
					}
					visibleReply := findMessage(page.Messages, reply.ID)
					if visibleReply == nil {
						t.Fatal("reply was lost with its hidden parent")
					}
					if visibility == "visible" {
						if visibleReply.Parent == nil || visibleReply.Parent.Content == nil {
							t.Fatal("visible reply parent was hidden")
						}
					} else if visibleReply.Parent != nil {
						t.Fatalf("hidden message leaked through reply parent: %#v", visibleReply.Parent)
					}
					return
				}
				opened, err := repo.OpenMessage(ctx, &recipient, entity.ID, message.ID, now)
				if visibility == "visible" {
					if err != nil || opened == nil || opened.Message.Content == nil {
						t.Fatalf("visible message rejected: %#v %v", opened, err)
					}
					return
				}
				if !errors.Is(err, chat.ErrMessageNotFound) || opened != nil {
					t.Fatalf("hidden message was opened: result=%#v err=%v", opened, err)
				}
				var persisted post.Post
				if err := db.First(&persisted, "id = ?", message.ID).Error; err != nil {
					t.Fatal(err)
				}
				if persisted.OpenedAt != nil || persisted.ExpiresAt != nil {
					t.Fatal("rejected opening started expiry")
				}
				var views int64
				if err := db.Model(&chat.MessageView{}).Where("message_id = ?", message.ID).Count(&views).Error; err != nil {
					t.Fatal(err)
				}
				if views != 0 {
					t.Fatal("rejected opening recorded consumption")
				}
			})
		}
	}
}
