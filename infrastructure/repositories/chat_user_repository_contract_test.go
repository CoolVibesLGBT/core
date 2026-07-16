package repositories

import (
	"context"
	"core/application/ports"
	"core/models"
	"core/models/chat"
	"core/models/media"
	"core/models/post"
	modelutils "core/models/utils"
	"core/types"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type chatContractUploadedFile struct{ mime string }

func (f chatContractUploadedFile) Filename() string    { return "image.jpg" }
func (f chatContractUploadedFile) Size() int64         { return 5 }
func (f chatContractUploadedFile) ContentType() string { return f.mime }
func (f chatContractUploadedFile) Open() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("image")), nil
}

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
		"posts.contentable_type",
		"posts.contentable_id",
		"LEFT JOIN engagements",
		"LEFT JOIN engagement_details",
		"ed.id IS NULL",
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

func TestValidateMessageFormRequiresTextOrAttachment(t *testing.T) {
	if err := validateMessageForm(ports.FormData{Values: map[string][]string{"content": {"  "}}}); err != chat.ErrEmptyMessage {
		t.Fatalf("validateMessageForm() error = %v, want %v", err, chat.ErrEmptyMessage)
	}
	if err := validateMessageForm(ports.FormData{Values: map[string][]string{"content": {"hello"}}}); err != nil {
		t.Fatalf("text message should be valid: %v", err)
	}
	if err := validateMessageForm(ports.FormData{Files: []ports.UploadedFile{nil}}); err != nil {
		t.Fatalf("attachment message should be valid: %v", err)
	}
}

func TestValidateMessageFormRequiresExactlyOneImageForViewOnce(t *testing.T) {
	image := chatContractUploadedFile{mime: "image/jpeg"}
	valid := ports.FormData{ViewOnce: true, ImageCount: 1, Files: []ports.UploadedFile{image}}
	if err := validateMessageForm(valid); err != nil {
		t.Fatalf("valid view-once image rejected: %v", err)
	}
	unknownMIME := ports.FormData{ViewOnce: true, ImageCount: 1, Files: []ports.UploadedFile{chatContractUploadedFile{mime: "application/octet-stream"}}}
	if err := validateMessageForm(unknownMIME); err != nil {
		t.Fatalf("image extension with generic browser MIME rejected: %v", err)
	}
	for name, form := range map[string]ports.FormData{
		"no image":   {ViewOnce: true},
		"two images": {ViewOnce: true, ImageCount: 2, Files: []ports.UploadedFile{image, image}},
		"video":      {ViewOnce: true, VideoCount: 1, Files: []ports.UploadedFile{chatContractUploadedFile{mime: "video/mp4"}}},
		"wrong mime": {ViewOnce: true, ImageCount: 1, Files: []ports.UploadedFile{chatContractUploadedFile{mime: "video/mp4"}}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateMessageForm(form); err != chat.ErrInvalidViewOnce {
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
		AuthUser: &models.User{ID: authID},
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
