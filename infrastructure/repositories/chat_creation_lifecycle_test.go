package repositories

import (
	"context"
	domainmedia "core/domain/media"
	"core/helpers"
	"core/models"
	"core/models/chat"
	"core/models/media"
	"core/models/post"
	"core/models/post/payloads"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

func TestCreateChatMessagePersistsPostLifecycleAndMediaIntegration(t *testing.T) {
	for _, viewOnce := range []bool{false, true} {
		name := "timed text"
		if viewOnce {
			name = "timed view-once image"
		}
		t.Run(name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			db := privatePhotoIntegrationDB(t)
			if err := db.AutoMigrate(&post.Post{}, &chat.Chat{}, &chat.ChatParticipant{}, &payloads.Event{}, &payloads.Poll{}, &models.Hashtag{}); err != nil {
				t.Fatal(err)
			}
			// Empty projection relations keep hydration enabled without requiring
			// the unrelated location/PostGIS schema in this upload fixture.
			for _, statement := range []string{
				"CREATE TABLE locations (id uuid, contentable_id uuid, contentable_type text, deleted_at timestamptz)",
				"CREATE TABLE post_hashtags (post_id uuid, hashtag_id uuid)",
			} {
				if err := db.Exec(statement).Error; err != nil {
					t.Fatal(err)
				}
			}
			author, recipient := privatePhotoTestUsers(t, db)
			author.DefaultLanguage = "en"
			entity := chat.Chat{ID: uuid.New(), Type: chat.ChatTypePrivate, CreatorID: author.ID}
			if err := db.Omit(clause.Associations).Create(&entity).Error; err != nil {
				t.Fatal(err)
			}
			// Add the recipient after sending so this test exercises persistence
			// without delivering any external notifications.
			sender := chat.ChatParticipant{ID: uuid.New(), ChatID: entity.ID, UserID: author.ID, JoinedAt: time.Now().UTC()}
			if err := db.Create(&sender).Error; err != nil {
				t.Fatal(err)
			}
			node, err := helpers.NewDefaultNode()
			if err != nil {
				t.Fatal(err)
			}
			mediaRepo := NewMediaRepository(db, node)
			postRepo := NewPostRepository(db, node, mediaRepo, nil, nil)
			repo := NewChatRepository(db, node, postRepo, nil, nil)
			values := map[string][]string{
				"chat_id": {entity.ID.String()}, "content": {"private caption"},
				"expires_in_seconds": {"60"}, "client_id": {"optimistic-id"},
			}
			var files []*domainmedia.UploadedFile
			if viewOnce {
				values["view_once"] = []string{"on"}
				files = []*domainmedia.UploadedFile{privatePhotoPNG(t)}
			}
			created, err := repo.AddMessageToChat(context.Background(), values, files, &author)
			if err != nil {
				t.Fatal(err)
			}
			if created.ClientID != "optimistic-id" {
				t.Fatalf("client ID = %q", created.ClientID)
			}
			var persisted post.Post
			if err := db.First(&persisted, "id = ?", created.ID).Error; err != nil {
				t.Fatal(err)
			}
			if persisted.ExpiresInSeconds == nil || *persisted.ExpiresInSeconds != 60 || persisted.ViewOnce != viewOnce || persisted.OpenedAt != nil || persisted.ExpiresAt != nil {
				t.Fatalf("post lifecycle was not persisted before opening: %#v", persisted)
			}
			var attachment media.Media
			if viewOnce {
				if err := db.Preload("File").First(&attachment, "owner_id = ?", created.ID).Error; err != nil {
					t.Fatal(err)
				}
				if attachment.IsPublic || attachment.OwnerType != media.OwnerPost || attachment.Role != media.RoleChatMedia {
					t.Fatalf("chat upload did not use protected post media: %#v", attachment)
				}
				if _, err := os.Stat(attachment.File.StoragePath); err != nil {
					t.Fatal(err)
				}
			}
			receiver := chat.ChatParticipant{ID: uuid.New(), ChatID: entity.ID, UserID: recipient.ID, JoinedAt: sender.JoinedAt, UnreadCount: 1}
			if err := db.Create(&receiver).Error; err != nil {
				t.Fatal(err)
			}
			now := time.Now().UTC().Truncate(time.Microsecond)
			opened, err := repo.OpenMessage(context.Background(), &recipient, entity.ID, created.ID, now)
			if err != nil {
				t.Fatal(err)
			}
			if opened.Message.ExpiresAt == nil || !opened.Message.ExpiresAt.Equal(now.Add(time.Minute)) {
				t.Fatalf("opening did not start post expiry: %#v", opened.Message)
			}
			if viewOnce {
				if opened.Media == nil || opened.Media.DataBase64 == "" {
					t.Fatal("opening did not return media")
				}
				if _, err := repo.OpenMessage(context.Background(), &recipient, entity.ID, created.ID, now.Add(time.Second)); !errors.Is(err, chat.ErrMessageAlreadySeen) {
					t.Fatalf("second opening error = %v", err)
				}
			}
			expired, err := repo.ExpireMessages(context.Background(), now.Add(61*time.Second), 10)
			if err != nil {
				t.Fatal(err)
			}
			if len(expired) != 1 || expired[0].MessageID != created.ID {
				t.Fatalf("expired messages = %#v", expired)
			}
			var count int64
			if err := db.Model(&post.Post{}).Where("id = ?", created.ID).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatal("expired post is still visible")
			}
			if viewOnce {
				if _, err := os.Stat(attachment.File.StoragePath); err != nil {
					t.Fatalf("post expiry removed the stored media: %v", err)
				}
			}
		})
	}
}
