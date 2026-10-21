package repositories

import (
	"context"
	"core/models"
	"core/models/chat"
	"core/models/media"
	"core/models/post"
	modelutils "core/models/utils"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type chatFileAccessFixture struct {
	db            *gorm.DB
	owner, viewer models.User
	conversation  chat.Chat
	message       post.Post
	attachment    media.Media
	avatar        media.Media
	prefix        string
}

func newChatFileAccessFixture(t *testing.T) chatFileAccessFixture {
	t.Helper()
	db := privatePhotoIntegrationDB(t)
	if err := db.AutoMigrate(&post.Post{}, &chat.Chat{}, &chat.ChatParticipant{}); err != nil {
		t.Fatal(err)
	}
	owner, viewer := privatePhotoTestUsers(t, db)
	conversation := chat.Chat{ID: uuid.New(), Type: chat.ChatTypeGroup, CreatorID: owner.ID, Visibility: "public"}
	contentType := string(post.PostKindChat)
	message := post.Post{ID: uuid.New(), PublicID: 1, AuthorID: owner.ID, ContentableID: &conversation.ID, ContentableType: &contentType, PostKind: post.PostKindMessage, CreatedAt: time.Now().UTC()}
	file := modelutils.FileMetadata{ID: uuid.New(), StoragePath: "static/uploads/fixture.png", Name: "fixture.png", MimeType: "image/png"}
	attachment := media.Media{ID: uuid.New(), PublicID: 1, FileID: file.ID, OwnerID: message.ID, OwnerType: media.OwnerPost, UserID: owner.ID, Role: media.RoleChatImage}
	avatar := media.Media{ID: uuid.New(), PublicID: 2, FileID: file.ID, OwnerID: conversation.ID, OwnerType: media.OwnerChat, UserID: owner.ID, Role: media.RoleChatImage}
	conversation.AvatarID = &avatar.ID
	participant := chat.ChatParticipant{ID: uuid.New(), ChatID: conversation.ID, UserID: viewer.ID}
	for _, record := range []any{&file, &conversation, &message, &attachment, &avatar, &participant} {
		if err := db.Omit(clause.Associations).Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
	return chatFileAccessFixture{db: db, owner: owner, viewer: viewer, conversation: conversation, message: message, attachment: attachment, avatar: avatar, prefix: "static/uploads/fixture"}
}

func TestVisibleChatFileAccessEnforcesMessageAndMembershipVisibilityIntegration(t *testing.T) {
	fixture := newChatFileAccessFixture(t)
	now := time.Now().UTC()
	for _, test := range []struct {
		name   string
		mutate func(*gorm.DB) error
		viewer uuid.UUID
		prefix string
		want   bool
	}{
		{name: "visible message", want: true},
		{name: "exact file path", prefix: fixture.prefix + ".png", want: true},
		{name: "different file", prefix: "static/uploads/other"},
		{name: "anonymous", viewer: uuid.Nil},
		{name: "outsider", viewer: uuid.New()},
		{name: "departed member", mutate: func(tx *gorm.DB) error {
			return tx.Model(&chat.ChatParticipant{}).Where("chat_id = ? AND user_id = ?", fixture.conversation.ID, fixture.viewer.ID).Update("left_at", now).Error
		}},
		{name: "cleared history", mutate: func(tx *gorm.DB) error {
			return tx.Model(&chat.ChatParticipant{}).Where("chat_id = ? AND user_id = ?", fixture.conversation.ID, fixture.viewer.ID).Update("cleared_at", now.Add(time.Hour)).Error
		}},
		{name: "message after clear", want: true, mutate: func(tx *gorm.DB) error {
			return tx.Model(&chat.ChatParticipant{}).Where("chat_id = ? AND user_id = ?", fixture.conversation.ID, fixture.viewer.ID).Update("cleared_at", now.Add(-time.Hour)).Error
		}},
		{name: "soft deleted message", mutate: func(tx *gorm.DB) error { return tx.Delete(&post.Post{}, "id = ?", fixture.message.ID).Error }},
		{name: "soft deleted chat", mutate: func(tx *gorm.DB) error { return tx.Delete(&chat.Chat{}, "id = ?", fixture.conversation.ID).Error }},
		{name: "view once", mutate: func(tx *gorm.DB) error {
			return tx.Model(&post.Post{}).Where("id = ?", fixture.message.ID).Update("view_once", true).Error
		}},
		{name: "timed message", mutate: func(tx *gorm.DB) error {
			return tx.Model(&post.Post{}).Where("id = ?", fixture.message.ID).Update("expires_in_seconds", 60).Error
		}},
		{name: "expired message", mutate: func(tx *gorm.DB) error {
			return tx.Model(&post.Post{}).Where("id = ?", fixture.message.ID).Update("expires_at", now.Add(-time.Hour)).Error
		}},
		{name: "unexpired message", want: true, mutate: func(tx *gorm.DB) error {
			return tx.Model(&post.Post{}).Where("id = ?", fixture.message.ID).Update("expires_at", now.Add(time.Hour)).Error
		}},
		{name: "wrong attachment owner type", mutate: func(tx *gorm.DB) error {
			return tx.Model(&media.Media{}).Where("id = ?", fixture.attachment.ID).Update("owner_type", media.OwnerUser).Error
		}},
		{name: "not a message", mutate: func(tx *gorm.DB) error {
			return tx.Model(&post.Post{}).Where("id = ?", fixture.message.ID).Update("post_kind", post.PostKindPost).Error
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			tx := fixture.db.Begin()
			if tx.Error != nil {
				t.Fatal(tx.Error)
			}
			defer tx.Rollback()
			if test.mutate != nil {
				if err := test.mutate(tx); err != nil {
					t.Fatal(err)
				}
			}
			viewer := fixture.viewer.ID
			if test.name == "anonymous" || test.viewer != uuid.Nil {
				viewer = test.viewer
			}
			prefix := fixture.prefix
			if test.prefix != "" {
				prefix = test.prefix
			}
			allowed, err := NewMediaRepository(tx, nil).HasVisibleChatFileAccess(context.Background(), prefix, viewer)
			if err != nil || allowed != test.want {
				t.Fatalf("allowed=%v, error=%v, want=%v", allowed, err, test.want)
			}
		})
	}
	for _, test := range []struct {
		name        string
		contentID   uuid.UUID
		contentType models.EngagementContentableType
		kind        models.EngagementKind
		actor       uuid.UUID
		want        bool
	}{
		{"message deleted for viewer", fixture.message.ID, models.EngagementContentableTypeMessage, models.EngagementKindMessageDeletedForMe, fixture.viewer.ID, false},
		{"message deleted for other", fixture.message.ID, models.EngagementContentableTypeMessage, models.EngagementKindMessageDeletedForMe, fixture.owner.ID, true},
		{"message deleted for all", fixture.message.ID, models.EngagementContentableTypeMessage, models.EngagementKindMessageDeletedForAll, fixture.owner.ID, false},
		{"chat deleted for viewer", fixture.conversation.ID, models.EngagementContentableTypeChat, models.EngagementKindChatDeletedForMe, fixture.viewer.ID, false},
		{"chat deleted for other", fixture.conversation.ID, models.EngagementContentableTypeChat, models.EngagementKindChatDeletedForMe, fixture.owner.ID, true},
		{"chat deleted for all", fixture.conversation.ID, models.EngagementContentableTypeChat, models.EngagementKindChatDeletedForAll, fixture.owner.ID, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			tx := fixture.db.Begin()
			if tx.Error != nil {
				t.Fatal(tx.Error)
			}
			defer tx.Rollback()
			aggregate := models.Engagement{ID: uuid.New(), ContentableID: test.contentID, ContentableType: test.contentType}
			if err := tx.Create(&aggregate).Error; err != nil {
				t.Fatal(err)
			}
			detail := models.EngagementDetail{ID: uuid.New(), EngagementID: aggregate.ID, EngagerID: test.actor, Kind: test.kind}
			if err := tx.Omit(clause.Associations).Create(&detail).Error; err != nil {
				t.Fatal(err)
			}
			allowed, err := NewMediaRepository(tx, nil).HasVisibleChatFileAccess(context.Background(), fixture.prefix, fixture.viewer.ID)
			if err != nil || allowed != test.want {
				t.Fatalf("allowed=%v, error=%v, want=%v", allowed, err, test.want)
			}
		})
	}
}

func TestPublicGroupAvatarRequiresCurrentAvatarAndCorrectOwnerIntegration(t *testing.T) {
	fixture := newChatFileAccessFixture(t)
	for _, test := range []struct {
		name   string
		mutate func(*gorm.DB) error
		want   bool
	}{
		{name: "current public avatar", want: true},
		{name: "private group", mutate: func(tx *gorm.DB) error {
			return tx.Model(&chat.Chat{}).Where("id = ?", fixture.conversation.ID).Update("visibility", "private").Error
		}},
		{name: "private conversation", mutate: func(tx *gorm.DB) error {
			return tx.Model(&chat.Chat{}).Where("id = ?", fixture.conversation.ID).Update("type", chat.ChatTypePrivate).Error
		}},
		{name: "removed avatar", mutate: func(tx *gorm.DB) error {
			return tx.Model(&chat.Chat{}).Where("id = ?", fixture.conversation.ID).Update("avatar_id", nil).Error
		}},
		{name: "deleted group", mutate: func(tx *gorm.DB) error { return tx.Delete(&chat.Chat{}, "id = ?", fixture.conversation.ID).Error }},
		{name: "avatar from another group", mutate: func(tx *gorm.DB) error {
			return tx.Model(&media.Media{}).Where("id = ?", fixture.avatar.ID).Update("owner_id", uuid.New()).Error
		}},
		{name: "post owned avatar", mutate: func(tx *gorm.DB) error {
			return tx.Model(&media.Media{}).Where("id = ?", fixture.avatar.ID).Update("owner_type", media.OwnerPost).Error
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			tx := fixture.db.Begin()
			if tx.Error != nil {
				t.Fatal(tx.Error)
			}
			defer tx.Rollback()
			if test.mutate != nil {
				if err := test.mutate(tx); err != nil {
					t.Fatal(err)
				}
			}
			allowed, err := NewMediaRepository(tx, nil).IsPublicGroupAvatar(context.Background(), fixture.prefix)
			if err != nil || allowed != test.want {
				t.Fatalf("allowed=%v, error=%v, want=%v", allowed, err, test.want)
			}
		})
	}
}
