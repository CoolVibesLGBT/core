package repositories

import (
	"context"
	"core/application/ports"
	"core/constants"
	"core/helpers"
	"core/models"
	"core/models/chat"
	"core/models/post"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type chatMutationFixture struct {
	db          *gorm.DB
	repo        *ChatRepository
	chat        chat.Chat
	message     post.Post
	otherChat   chat.Chat
	otherMsg    post.Post
	creator     models.User
	author      models.User
	member      models.User
	chatAdmin   models.User
	systemAdmin models.User
	outsider    models.User
	leftAdmin   models.User
}

func prepareChatMutationFixture(t *testing.T) chatMutationFixture {
	t.Helper()
	db := expiryIntegrationDB(t)
	for _, model := range []interface{}{&models.User{}, &chat.Chat{}, &chat.ChatParticipant{}, &post.Post{}, &models.Engagement{}, &models.EngagementDetail{}} {
		if !db.Migrator().HasTable(model) {
			t.Skipf("chat mutation integration schema is missing %T", model)
		}
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	basePublicID := now.UnixNano()
	newUser := func(offset int64, role constants.UserRole) models.User {
		id := uuid.New()
		return models.User{
			ID: id, PublicID: basePublicID + offset, Domain: models.CoolVibes,
			UserName: "chat-authz-" + id.String(), DisplayName: fmt.Sprintf("Chat authz %d", offset),
			UserRole: role,
		}
	}

	fixture := chatMutationFixture{db: db}
	fixture.creator = newUser(1, constants.UserRoleUser)
	fixture.author = newUser(2, constants.UserRoleUser)
	fixture.member = newUser(3, constants.UserRoleUser)
	fixture.chatAdmin = newUser(4, constants.UserRoleUser)
	fixture.systemAdmin = newUser(5, constants.UserRoleAdmin)
	fixture.outsider = newUser(6, constants.UserRoleUser)
	fixture.leftAdmin = newUser(7, constants.UserRoleUser)
	users := []models.User{
		fixture.creator, fixture.author, fixture.member, fixture.chatAdmin,
		fixture.systemAdmin, fixture.outsider, fixture.leftAdmin,
	}
	if err := db.Omit(clause.Associations).
		Select("id", "public_id", "domain", "user_name", "display_name", "user_role", "created_at", "updated_at").
		Create(&users).Error; err != nil {
		t.Fatalf("create chat mutation users: %v", err)
	}

	chatID := uuid.New()
	otherChatID := uuid.New()
	contentableType := string(post.PostKindChat)
	fixture.message = post.Post{
		ID: uuid.New(), PublicID: basePublicID + 20, PostKind: post.PostKindMessage,
		Domain: models.CoolVibes, ContentableType: &contentableType, ContentableID: &chatID,
		AuthorID: fixture.author.ID, Published: true, CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute),
	}
	otherContentableType := string(post.PostKindChat)
	fixture.otherMsg = post.Post{
		ID: uuid.New(), PublicID: basePublicID + 21, PostKind: post.PostKindMessage,
		Domain: models.CoolVibes, ContentableType: &otherContentableType, ContentableID: &otherChatID,
		AuthorID: fixture.author.ID, Published: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Omit(clause.Associations).Create(&[]post.Post{fixture.message, fixture.otherMsg}).Error; err != nil {
		t.Fatalf("create chat mutation messages: %v", err)
	}

	fixture.chat = chat.Chat{
		ID: chatID, Type: chat.ChatTypeGroup, CreatorID: fixture.creator.ID,
		PinnedMsgID: &fixture.message.ID, PinnedByID: &fixture.creator.ID,
		LastMessageID: &fixture.message.ID, LastMessageTimestamp: &fixture.message.CreatedAt,
		CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now,
	}
	fixture.otherChat = chat.Chat{
		ID: otherChatID, Type: chat.ChatTypePrivate, CreatorID: fixture.author.ID,
		LastMessageID: &fixture.otherMsg.ID, LastMessageTimestamp: &fixture.otherMsg.CreatedAt,
		CreatedAt: now.Add(-time.Minute), UpdatedAt: now,
	}
	if err := db.Omit(clause.Associations).Create(&[]chat.Chat{fixture.chat, fixture.otherChat}).Error; err != nil {
		t.Fatalf("create chat mutation chats: %v", err)
	}

	leftAt := now.Add(-time.Second)
	participants := []chat.ChatParticipant{
		{ID: uuid.New(), ChatID: chatID, UserID: fixture.creator.ID, Role: chat.ParticipantRoleOwner, JoinedAt: now.Add(-2 * time.Minute)},
		{ID: uuid.New(), ChatID: chatID, UserID: fixture.author.ID, Role: chat.ParticipantRoleMember, JoinedAt: now.Add(-2 * time.Minute)},
		{ID: uuid.New(), ChatID: chatID, UserID: fixture.member.ID, Role: chat.ParticipantRoleMember, JoinedAt: now.Add(-2 * time.Minute), UnreadCount: 1},
		{ID: uuid.New(), ChatID: chatID, UserID: fixture.chatAdmin.ID, Role: chat.ParticipantRoleAdmin, JoinedAt: now.Add(-2 * time.Minute), UnreadCount: 1},
		{ID: uuid.New(), ChatID: chatID, UserID: fixture.systemAdmin.ID, Role: chat.ParticipantRoleMember, JoinedAt: now.Add(-2 * time.Minute), UnreadCount: 1},
		{ID: uuid.New(), ChatID: chatID, UserID: fixture.leftAdmin.ID, Role: chat.ParticipantRoleAdmin, JoinedAt: now.Add(-2 * time.Minute), LeftAt: &leftAt},
	}
	if err := db.Omit(clause.Associations).Create(&participants).Error; err != nil {
		t.Fatalf("create chat mutation participants: %v", err)
	}
	fixture.repo = &ChatRepository{db: db}
	return fixture
}

func countChatMutationKind(t *testing.T, fixture chatMutationFixture, contentableID uuid.UUID, contentableType models.EngagementContentableType, kind models.EngagementKind) int64 {
	t.Helper()
	var count int64
	err := fixture.db.Model(&models.EngagementDetail{}).
		Joins("JOIN engagements e ON e.id = engagement_details.engagement_id").
		Where("e.contentable_id = ? AND e.contentable_type = ? AND engagement_details.kind = ?", contentableID, contentableType, kind).
		Count(&count).Error
	if err != nil {
		t.Fatalf("count chat mutation kind: %v", err)
	}
	return count
}

func TestDeleteMessageForAllAuthorizationAndIdempotencyIntegration(t *testing.T) {
	t.Run("ordinary member denied", func(t *testing.T) {
		fixture := prepareChatMutationFixture(t)
		err := fixture.repo.DeleteMessageForAll(context.Background(), &fixture.member, fixture.chat.ID, fixture.member.ID, fixture.message.ID)
		if !errors.Is(err, chat.ErrPermissionDenied) {
			t.Fatalf("error = %v, want permission denied", err)
		}
		if count := countChatMutationKind(t, fixture, fixture.message.ID, models.EngagementContentableTypeMessage, models.EngagementKindMessageDeletedForAll); count != 0 {
			t.Fatalf("unauthorized mutation created %d deletion flags", count)
		}
	})

	t.Run("outsider and left admin denied", func(t *testing.T) {
		fixture := prepareChatMutationFixture(t)
		for _, actor := range []*models.User{&fixture.outsider, &fixture.leftAdmin} {
			err := fixture.repo.DeleteMessageForAll(context.Background(), actor, fixture.chat.ID, actor.ID, fixture.message.ID)
			if !errors.Is(err, chat.ErrNotParticipant) {
				t.Fatalf("actor %s error = %v, want not participant", actor.ID, err)
			}
		}
	})

	t.Run("author succeeds once and repairs metadata", func(t *testing.T) {
		fixture := prepareChatMutationFixture(t)
		for attempt := 0; attempt < 2; attempt++ {
			if err := fixture.repo.DeleteMessageForAll(context.Background(), &fixture.author, fixture.chat.ID, fixture.author.ID, fixture.message.ID); err != nil {
				t.Fatalf("attempt %d: %v", attempt+1, err)
			}
		}
		if count := countChatMutationKind(t, fixture, fixture.message.ID, models.EngagementContentableTypeMessage, models.EngagementKindMessageDeletedForAll); count != 1 {
			t.Fatalf("global deletion flags = %d, want 1", count)
		}
		var repaired chat.Chat
		if err := fixture.db.First(&repaired, "id = ?", fixture.chat.ID).Error; err != nil {
			t.Fatal(err)
		}
		if repaired.PinnedMsgID != nil || repaired.PinnedByID != nil || repaired.LastMessageID != nil || repaired.LastMessageTimestamp != nil {
			t.Fatalf("globally deleted last message metadata was not repaired: %#v", repaired)
		}
	})

	for name, selectActor := range map[string]func(chatMutationFixture) *models.User{
		"creator":      func(f chatMutationFixture) *models.User { return &f.creator },
		"chat admin":   func(f chatMutationFixture) *models.User { return &f.chatAdmin },
		"system admin": func(f chatMutationFixture) *models.User { return &f.systemAdmin },
	} {
		t.Run(name+" may moderate", func(t *testing.T) {
			fixture := prepareChatMutationFixture(t)
			actor := selectActor(fixture)
			if err := fixture.repo.DeleteMessageForAll(context.Background(), actor, fixture.chat.ID, actor.ID, fixture.message.ID); err != nil {
				t.Fatalf("authorized moderator error = %v", err)
			}
		})
	}
}

func TestChatMutationTargetBindingAndPhysicalDeleteAuthorizationIntegration(t *testing.T) {
	t.Run("cross-chat message IDs rejected", func(t *testing.T) {
		fixture := prepareChatMutationFixture(t)
		checks := []struct {
			name string
			call func() error
		}{
			{name: "pin", call: func() error {
				return fixture.repo.PinMessage(context.Background(), &fixture.creator, fixture.chat.ID, fixture.creator.ID, fixture.otherMsg.ID)
			}},
			{name: "delete for self", call: func() error {
				return fixture.repo.DeleteMessageForUser(context.Background(), &fixture.member, fixture.chat.ID, fixture.member.ID, fixture.otherMsg.ID)
			}},
			{name: "delete for all", call: func() error {
				return fixture.repo.DeleteMessageForAll(context.Background(), &fixture.author, fixture.chat.ID, fixture.author.ID, fixture.otherMsg.ID)
			}},
			{name: "mark read", call: func() error {
				return fixture.repo.MarkChatMessageRead(context.Background(), &fixture.member, fixture.chat.ID, []uuid.UUID{fixture.otherMsg.ID})
			}},
		}
		for _, check := range checks {
			t.Run(check.name, func(t *testing.T) {
				if err := check.call(); !errors.Is(err, chat.ErrMessageNotFound) {
					t.Fatalf("error = %v, want message not found", err)
				}
			})
		}
	})

	t.Run("ordinary member cannot physically delete", func(t *testing.T) {
		fixture := prepareChatMutationFixture(t)
		err := fixture.repo.DeleteMessage(context.Background(), &fixture.member, fixture.chat.ID, fixture.member.ID, fixture.message.ID)
		if !errors.Is(err, chat.ErrPermissionDenied) {
			t.Fatalf("error = %v, want permission denied", err)
		}
		if err := fixture.db.First(&post.Post{}, "id = ?", fixture.message.ID).Error; err != nil {
			t.Fatalf("unauthorized delete removed message: %v", err)
		}
	})

	t.Run("author physical delete is scoped and atomic", func(t *testing.T) {
		fixture := prepareChatMutationFixture(t)
		if err := fixture.repo.DeleteMessage(context.Background(), &fixture.author, fixture.chat.ID, fixture.author.ID, fixture.message.ID); err != nil {
			t.Fatalf("DeleteMessage() error = %v", err)
		}
		var deleted post.Post
		if err := fixture.db.Unscoped().First(&deleted, "id = ?", fixture.message.ID).Error; err != nil || !deleted.DeletedAt.Valid {
			t.Fatalf("message was not soft deleted atomically: deleted=%v error=%v", deleted.DeletedAt.Valid, err)
		}
		if err := fixture.db.First(&post.Post{}, "id = ?", fixture.otherMsg.ID).Error; err != nil {
			t.Fatalf("other chat message was affected: %v", err)
		}
	})

	t.Run("physical delete after global hide does not decrement unread twice", func(t *testing.T) {
		fixture := prepareChatMutationFixture(t)
		contentableType := string(post.PostKindChat)
		newer := post.Post{
			ID: uuid.New(), PublicID: fixture.message.PublicID + 100, PostKind: post.PostKindMessage,
			Domain: models.CoolVibes, ContentableType: &contentableType, ContentableID: &fixture.chat.ID,
			AuthorID: fixture.author.ID, Published: true,
			CreatedAt: fixture.message.CreatedAt.Add(time.Second), UpdatedAt: fixture.message.CreatedAt.Add(time.Second),
		}
		if err := fixture.db.Omit(clause.Associations).Create(&newer).Error; err != nil {
			t.Fatalf("create newer unread message: %v", err)
		}
		if err := fixture.db.Model(&chat.Chat{}).Where("id = ?", fixture.chat.ID).Updates(map[string]interface{}{
			"last_message_id": newer.ID, "last_message_timestamp": newer.CreatedAt,
		}).Error; err != nil {
			t.Fatalf("set newer last message: %v", err)
		}
		if err := fixture.db.Model(&chat.ChatParticipant{}).
			Where("chat_id = ? AND user_id = ?", fixture.chat.ID, fixture.member.ID).
			Update("unread_count", 2).Error; err != nil {
			t.Fatalf("set unread count: %v", err)
		}

		if err := fixture.repo.DeleteMessageForAll(context.Background(), &fixture.author, fixture.chat.ID, fixture.author.ID, fixture.message.ID); err != nil {
			t.Fatalf("DeleteMessageForAll() error = %v", err)
		}
		if err := fixture.repo.DeleteMessage(context.Background(), &fixture.author, fixture.chat.ID, fixture.author.ID, fixture.message.ID); err != nil {
			t.Fatalf("DeleteMessage() error = %v", err)
		}
		var participant chat.ChatParticipant
		if err := fixture.db.Where("chat_id = ? AND user_id = ?", fixture.chat.ID, fixture.member.ID).First(&participant).Error; err != nil {
			t.Fatal(err)
		}
		if participant.UnreadCount != 1 {
			t.Fatalf("unread count = %d, want 1 after one logical deletion", participant.UnreadCount)
		}
	})
}

func TestDeleteChatForAllRequiresActiveCreatorOrAdminIntegration(t *testing.T) {
	t.Run("member denied", func(t *testing.T) {
		fixture := prepareChatMutationFixture(t)
		err := fixture.repo.DeleteChatForAll(context.Background(), &fixture.member, fixture.chat.ID)
		if !errors.Is(err, chat.ErrPermissionDenied) {
			t.Fatalf("error = %v, want permission denied", err)
		}
	})

	t.Run("system admin must still participate", func(t *testing.T) {
		fixture := prepareChatMutationFixture(t)
		outsiderAdmin := fixture.outsider
		outsiderAdmin.UserRole = constants.UserRoleAdmin
		err := fixture.repo.DeleteChatForAll(context.Background(), &outsiderAdmin, fixture.chat.ID)
		if !errors.Is(err, chat.ErrNotParticipant) {
			t.Fatalf("error = %v, want not participant", err)
		}
	})

	for name, selectActor := range map[string]func(chatMutationFixture) *models.User{
		"creator":      func(f chatMutationFixture) *models.User { return &f.creator },
		"chat admin":   func(f chatMutationFixture) *models.User { return &f.chatAdmin },
		"system admin": func(f chatMutationFixture) *models.User { return &f.systemAdmin },
	} {
		t.Run(name+" succeeds idempotently", func(t *testing.T) {
			fixture := prepareChatMutationFixture(t)
			actor := selectActor(fixture)
			for attempt := 0; attempt < 2; attempt++ {
				if err := fixture.repo.DeleteChatForAll(context.Background(), actor, fixture.chat.ID); err != nil {
					t.Fatalf("attempt %d: %v", attempt+1, err)
				}
			}
			if count := countChatMutationKind(t, fixture, fixture.chat.ID, models.EngagementContentableTypeChat, models.EngagementKindChatDeletedForAll); count != 1 {
				t.Fatalf("global chat deletion flags = %d, want 1", count)
			}
		})
	}
}

func TestAddMessageRollsBackPostWhenChatMetadataUpdateFailsIntegration(t *testing.T) {
	fixture := prepareChatMutationFixture(t)
	node, err := helpers.NewDefaultNode()
	if err != nil {
		t.Fatalf("create snowflake node: %v", err)
	}
	postRepo := NewPostRepository(fixture.db, node, nil, nil, nil)
	chatRepo := NewChatRepository(fixture.db, node, postRepo, nil, nil)

	var beforeMessages int64
	if err := fixture.db.Model(&post.Post{}).
		Where("post_kind = ? AND contentable_type = ? AND contentable_id = ?", post.PostKindMessage, post.PostKindChat, fixture.chat.ID).
		Count(&beforeMessages).Error; err != nil {
		t.Fatal(err)
	}
	var beforeParticipant chat.ChatParticipant
	if err := fixture.db.Where("chat_id = ? AND user_id = ?", fixture.chat.ID, fixture.member.ID).First(&beforeParticipant).Error; err != nil {
		t.Fatal(err)
	}

	forcedErr := errors.New("forced chat metadata update failure")
	callbackName := "test:force_chat_metadata_failure:" + uuid.NewString()
	if err := fixture.db.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "chats" {
			_ = tx.AddError(forcedErr)
		}
	}); err != nil {
		t.Fatalf("register failure callback: %v", err)
	}
	t.Cleanup(func() { _ = fixture.db.Callback().Update().Remove(callbackName) })

	_, err = chatRepo.AddMessageToChat(context.Background(), ports.FormData{Values: map[string][]string{
		"chat_id": {fixture.chat.ID.String()},
		"content": {"must roll back"},
	}}, &fixture.author)
	if !errors.Is(err, forcedErr) {
		t.Fatalf("AddMessageToChat() error = %v, want forced failure", err)
	}

	var afterMessages int64
	if err := fixture.db.Model(&post.Post{}).
		Where("post_kind = ? AND contentable_type = ? AND contentable_id = ?", post.PostKindMessage, post.PostKindChat, fixture.chat.ID).
		Count(&afterMessages).Error; err != nil {
		t.Fatal(err)
	}
	if afterMessages != beforeMessages {
		t.Fatalf("chat messages = %d, want rollback count %d", afterMessages, beforeMessages)
	}
	var afterParticipant chat.ChatParticipant
	if err := fixture.db.Where("chat_id = ? AND user_id = ?", fixture.chat.ID, fixture.member.ID).First(&afterParticipant).Error; err != nil {
		t.Fatal(err)
	}
	if afterParticipant.UnreadCount != beforeParticipant.UnreadCount {
		t.Fatalf("unread count = %d, want rollback value %d", afterParticipant.UnreadCount, beforeParticipant.UnreadCount)
	}
}
