package repositories

import (
	"context"
	upload "core/adapters/media"
	"core/application/ports"
	"core/application/types"
	domainmedia "core/domain/media"
	"core/models"
	"core/models/chat"
	"core/models/media"
	"core/models/post"
	modelutils "core/models/utils"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestChatMessagesByChatIDQueryFiltersDeletedMessagesAndClearTime(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &ChatRepository{db: db}
	userID := uuid.New()
	chatID := uuid.New()
	clearedAt := time.Now().UTC()

	var posts []post.Post
	tx := repo.messagesByChatIDQuery(userID, chatID, &clearedAt).
		Order("posts.created_at ASC").
		Find(&posts)
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	for _, fragment := range []string{
		"posts.post_kind",
		"posts.contentable_type",
		"posts.contentable_id",
		"NOT IN (",
		"FROM \"engagements\"",
		"FROM \"engagement_details\"",
		"posts.created_at >",
		"posts.expires_at IS NULL OR posts.expires_at >",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected SQL to contain %q, got %s", fragment, sql)
		}
	}
	vars := stringifyVars(tx.Statement.Vars)
	for _, value := range []string{"message_deleted_for_me", "message_deleted_for_all"} {
		if !strings.Contains(vars, value) {
			t.Fatalf("expected query vars to contain %q, got %s", value, vars)
		}
	}
}

func TestChatListPageQueryUsesCompositeCursorAndLimitPlusOne(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &ChatRepository{db: db}
	userID := uuid.New()
	cursorID := uuid.New()
	cursorTime := time.Now().UTC()

	var chats []chat.Chat
	tx := repo.chatsByUserIDQuery(context.Background(), ports.ChatListQuery{
		UserID: userID,
		Cursor: &ports.ChatListCursor{ActivityAt: cursorTime, ChatID: cursorID},
		Limit:  20,
	}).
		Order(chatListActivityExpression + " DESC, chats.id DESC").
		Limit(21).
		Find(&chats)
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	for _, fragment := range []string{
		"chats.id IN (",
		"user_id =",
		"left_at IS NULL",
		"NOT IN (",
		"engager_id =",
		"COALESCE(chats.last_message_timestamp, chats.created_at) <",
		"chats.id <",
		"ORDER BY COALESCE(chats.last_message_timestamp, chats.created_at) DESC, chats.id DESC",
		"LIMIT",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected chat page SQL to contain %q, got %s", fragment, sql)
		}
	}
}

func TestPrivateChatLookupRequiresBothParticipantsToBeActive(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &ChatRepository{db: db}
	fromUser := uuid.New()
	toUser := uuid.New()

	var entity chat.Chat
	tx := repo.privateChatBetweenUsersQuery(fromUser, toUser, time.Now().UTC()).First(&entity)
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	for _, fragment := range []string{
		"cp1.chat_id = chats.id AND cp1.left_at IS NULL",
		"cp2.chat_id = chats.id AND cp2.left_at IS NULL",
		"cp1.user_id",
		"cp2.user_id",
		"chats.deleted_at IS NULL",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("private chat lookup SQL is missing %q: %s", fragment, sql)
		}
	}
}

func TestChatMessagePageQueryUsesPublicIDCursorAndLimitPlusOne(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &ChatRepository{db: db}
	userID := uuid.New()
	chatID := uuid.New()
	cursor := int64(9001)

	var messages []post.Post
	tx := repo.chatMessagesPageQuery(context.Background(), ports.ChatMessageListQuery{
		UserID: userID,
		ChatID: chatID,
		Cursor: &ports.ChatMessageListCursor{PublicID: cursor},
		Limit:  20,
	}, nil).
		Limit(21).
		Find(&messages)
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	for _, fragment := range []string{
		"posts.post_kind",
		"posts.contentable_type",
		"posts.contentable_id",
		"posts.public_id <",
		"ORDER BY posts.public_id DESC",
		"LIMIT",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected message page SQL to contain %q, got %s", fragment, sql)
		}
	}
}

func TestValidateMessageFormRequiresTextOrAttachment(t *testing.T) {
	if err := validateMessageForm(map[string][]string{"content": {"  "}}, nil); err != chat.ErrEmptyMessage {
		t.Fatalf("validateMessageForm() error = %v, want %v", err, chat.ErrEmptyMessage)
	}
	if err := validateMessageForm(map[string][]string{"content": {"hello"}}, nil); err != nil {
		t.Fatalf("text message should be valid: %v", err)
	}
	if err := validateMessageForm(nil, []*domainmedia.UploadedFile{nil}); err != nil {
		t.Fatalf("attachment message should be valid: %v", err)
	}
}

func TestValidateMessageFormRequiresExactlyOneImageForViewOnce(t *testing.T) {
	image := upload.FromBytes("image.jpg", "image/jpeg", []byte("image"))
	values := map[string][]string{"view_once": {"true"}}
	if err := validateMessageForm(values, []*domainmedia.UploadedFile{image}); err != nil {
		t.Fatalf("valid view-once image rejected: %v", err)
	}
	if err := validateMessageForm(values, []*domainmedia.UploadedFile{upload.FromBytes("image.jpg", "application/octet-stream", []byte("image"))}); err != nil {
		t.Fatalf("image extension with generic browser MIME rejected: %v", err)
	}
	for name, files := range map[string][]*domainmedia.UploadedFile{
		"no image":        nil,
		"two images":      {image, image},
		"video":           {upload.FromBytes("image.jpg", "video/mp4", []byte("image"))},
		"image and video": {image, upload.FromBytes("image.jpg", "video/mp4", []byte("image"))},
		"nil file":        {nil},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateMessageForm(values, files); err != chat.ErrInvalidViewOnce {
				t.Fatalf("error = %v, want %v", err, chat.ErrInvalidViewOnce)
			}
		})
	}
}

func TestSanitizeMessagesKeepsOtherRecipientHiddenAfterGlobalOpen(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &ChatRepository{db: db}
	viewerID := uuid.New()
	authorID := uuid.New()
	duration := 60
	openedAt := time.Now().UTC()
	message := post.Post{
		ID: uuid.New(), AuthorID: authorID, ExpiresInSeconds: &duration, OpenedAt: &openedAt,
		Content: modelutils.MakeLocalizedString("en", "secret caption"),
		Attachments: []*media.Media{{ID: uuid.New(), File: modelutils.FileMetadata{
			URL: "/static/secret.jpg", StoragePath: "./static/secret.jpg",
		}}},
	}
	if err := repo.sanitizeMessagesForViewer([]*post.Post{&message}, viewerID); err != nil {
		t.Fatalf("sanitizeMessagesForViewer() error = %v", err)
	}
	if !message.ContentHidden || message.Content != nil || len(message.Attachments) != 0 {
		t.Fatalf("unopened recipient received content: %#v", message)
	}
	payload, err := json.Marshal(message)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"secret caption", "/static/secret.jpg", "storage_path", "variants"} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("sanitized payload leaked %q: %s", forbidden, payload)
		}
	}
}

func TestActiveParticipantQueryRequiresCurrentMembership(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &ChatRepository{db: db}
	chatID := uuid.New()
	userID := uuid.New()
	var participants []chat.ChatParticipant

	tx := repo.activeParticipantQuery(context.Background(), chatID, userID).Find(&participants)
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}
	sql := tx.Statement.SQL.String()
	for _, fragment := range []string{"chat_id", "user_id", "left_at IS NULL"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected membership SQL to contain %q, got %s", fragment, sql)
		}
	}
}

func TestDueMessagesQueryUsesExpiryLockAndMessageScope(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &ChatRepository{db: db}
	now := time.Now().UTC()
	var messages []post.Post

	tx := repo.dueMessagesQuery(db, now, 100).Find(&messages)
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}
	sql := tx.Statement.SQL.String()
	for _, fragment := range []string{"post_kind", "contentable_type", "opened_at", "expires_at", "deleted_at", "FOR UPDATE SKIP LOCKED", "LIMIT"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected expiry SQL to contain %q, got %s", fragment, sql)
		}
	}
}

func TestExpiredMessageUnreadUpdateNeverDropsBelowZero(t *testing.T) {
	db := newDryRunTaxonomyDB(t).Session(&gorm.Session{SkipDefaultTransaction: true})
	tx := decrementExpiredMessageUnread(db, uuid.New(), uuid.New(), time.Now())
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}
	sql := tx.Statement.SQL.String()
	for _, fragment := range []string{"chat_participants", "user_id <> ", "left_at IS NULL", "last_read_at IS NULL OR last_read_at <", "cleared_at IS NULL OR cleared_at <", "GREATEST(unread_count - 1, 0)"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected unread SQL to contain %q, got %s", fragment, sql)
		}
	}
}

func TestLiveUsersWithoutLocationQueryAppliesLiveCursorAndOrder(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &UserRepository{db: db}
	isLive := true
	cursor := int64(1001)

	var users []models.User
	tx := repo.liveUsersWithoutLocationQuery(types.Filter{IsLive: &isLive, Cursor: &cursor, Limit: 20}).
		Find(&users)
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	for _, fragment := range []string{
		"FROM \"users\"",
		"is_live =",
		"public_id >",
		"ORDER BY public_id ASC",
		"LIMIT",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected SQL to contain %q, got %s", fragment, sql)
		}
	}
}

func TestPublicProfileQueryPreloadsCountsWithoutPrivateViewDetails(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &UserRepository{db: db}
	query := repo.publicProfileQuery("profile-user")

	if _, ok := query.Statement.Preloads["Engagements"]; !ok {
		t.Fatal("public profile query must retain aggregate engagement counts")
	}
	for association := range query.Statement.Preloads {
		if strings.HasPrefix(association, "Engagements.EngagementDetails") {
			t.Fatalf("public profile query exposes private engagement details through preload %q", association)
		}
	}
}

func TestDeleteUserIDUsesAuthenticatedUserUUID(t *testing.T) {
	authID := uuid.New()
	submittedID := uuid.New()

	got, err := deleteUserID(types.Filter{
		AuthUser: &types.Actor{ID: authID},
		UserUUID: submittedID,
		UserID:   0,
	})
	if err != nil {
		t.Fatalf("deleteUserID() error = %v", err)
	}
	if got != authID {
		t.Fatalf("expected authenticated user id %s, got %s", authID, got)
	}
}

func TestDeleteUserIDAcceptsExplicitUUIDWithoutAuthenticatedUser(t *testing.T) {
	userID := uuid.New()

	got, err := deleteUserID(types.Filter{UserUUID: userID})
	if err != nil {
		t.Fatalf("deleteUserID() error = %v", err)
	}
	if got != userID {
		t.Fatalf("expected explicit user uuid %s, got %s", userID, got)
	}
}

func TestDeleteUserIDRejectsMissingUUID(t *testing.T) {
	if _, err := deleteUserID(types.Filter{UserID: 0}); err == nil {
		t.Fatalf("expected missing uuid error")
	}
}

func stringifyVars(values []interface{}) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprint(value))
	}
	return strings.Join(parts, " ")
}
