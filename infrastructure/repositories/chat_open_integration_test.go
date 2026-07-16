package repositories

import (
	"context"
	"core/constants"
	"core/models"
	"core/models/chat"
	"core/models/media"
	"core/models/post"
	modelutils "core/models/utils"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func createChatAttachment(t *testing.T, db *gorm.DB, messageID, userID uuid.UUID, publicID int64, data []byte) media.Media {
	t.Helper()
	path := filepath.Join(t.TempDir(), uuid.NewString()+".jpg")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	file := modelutils.FileMetadata{
		ID: uuid.New(), URL: "/static/unguessable/" + uuid.NewString() + ".jpg", StoragePath: path,
		MimeType: "image/jpeg", Size: int64(len(data)), Name: "photo.jpg", CreatedAt: time.Now().UTC(),
	}
	attachment := media.Media{
		ID: uuid.New(), PublicID: publicID, FileID: file.ID, OwnerID: messageID,
		OwnerType: media.OwnerPost, UserID: userID, Role: media.RoleChatMedia,
		ProcessingStatus: media.ProcessingStatusReady, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := db.Omit(clause.Associations).Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	if err := db.Omit(clause.Associations).Create(&attachment).Error; err != nil {
		t.Fatalf("create media: %v", err)
	}
	attachment.File = file
	return attachment
}

func findMessage(messages []post.Post, id uuid.UUID) *post.Post {
	for i := range messages {
		if messages[i].ID == id {
			return &messages[i]
		}
	}
	return nil
}

func TestOpenMessageStartsOnOpenAndAuthorizesEachRecipient(t *testing.T) {
	db := expiryIntegrationDB(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	baseID := now.UnixNano()
	author := models.User{ID: uuid.New(), PublicID: baseID, Domain: models.CoolVibes, UserName: "open-author-" + uuid.NewString(), DisplayName: "Author", UserRole: constants.UserRoleUser}
	recipientA := models.User{ID: uuid.New(), PublicID: baseID + 1, Domain: models.CoolVibes, UserName: "open-a-" + uuid.NewString(), DisplayName: "A", UserRole: constants.UserRoleUser}
	recipientB := models.User{ID: uuid.New(), PublicID: baseID + 2, Domain: models.CoolVibes, UserName: "open-b-" + uuid.NewString(), DisplayName: "B", UserRole: constants.UserRoleUser}
	outsider := models.User{ID: uuid.New(), PublicID: baseID + 3, Domain: models.CoolVibes, UserName: "open-out-" + uuid.NewString(), DisplayName: "Out", UserRole: constants.UserRoleUser}
	if err := db.Omit(clause.Associations).Create(&[]models.User{author, recipientA, recipientB, outsider}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	chatID := uuid.New()
	chatEntity := chat.Chat{ID: chatID, Type: chat.ChatTypePrivate, CreatorID: author.ID, CreatedAt: now, UpdatedAt: now}
	if err := db.Omit(clause.Associations).Create(&chatEntity).Error; err != nil {
		t.Fatalf("create chat: %v", err)
	}
	participants := []chat.ChatParticipant{
		{ID: uuid.New(), ChatID: chatID, UserID: author.ID, JoinedAt: now},
		{ID: uuid.New(), ChatID: chatID, UserID: recipientA.ID, JoinedAt: now},
		{ID: uuid.New(), ChatID: chatID, UserID: recipientB.ID, JoinedAt: now},
	}
	if err := db.Omit(clause.Associations).Create(&participants).Error; err != nil {
		t.Fatalf("create participants: %v", err)
	}

	contentableType := string(post.PostKindChat)
	duration := 60
	timed := post.Post{
		ID: uuid.New(), PublicID: baseID + 10, PostKind: post.PostKindMessage, Domain: models.CoolVibes,
		ContentableID: &chatID, ContentableType: &contentableType, AuthorID: author.ID, Published: true,
		Content: modelutils.MakeLocalizedString("en", "top secret"), ExpiresInSeconds: &duration,
		CreatedAt: now, UpdatedAt: now,
	}
	permanent := post.Post{
		ID: uuid.New(), PublicID: baseID + 11, PostKind: post.PostKindMessage, Domain: models.CoolVibes,
		ContentableID: &chatID, ContentableType: &contentableType, AuthorID: author.ID, Published: true,
		Content: modelutils.MakeLocalizedString("en", "permanent"), CreatedAt: now.Add(time.Second), UpdatedAt: now,
	}
	viewOnce := post.Post{
		ID: uuid.New(), PublicID: baseID + 12, PostKind: post.PostKindMessage, Domain: models.CoolVibes,
		ContentableID: &chatID, ContentableType: &contentableType, AuthorID: author.ID, Published: true,
		Content: modelutils.MakeLocalizedString("en", "caption"), ViewOnce: true,
		CreatedAt: now.Add(2 * time.Second), UpdatedAt: now,
	}
	if err := db.Omit(clause.Associations).Create(&[]post.Post{timed, permanent, viewOnce}).Error; err != nil {
		t.Fatalf("create messages: %v", err)
	}
	timedAttachment := createChatAttachment(t, db, timed.ID, author.ID, baseID+20, []byte("timed-image"))
	viewOnceAttachment := createChatAttachment(t, db, viewOnce.ID, author.ID, baseID+21, []byte("one-view-image"))

	repo := &ChatRepository{db: db}
	before, err := repo.GetMessagesByChatID(recipientA.ID, chatID)
	if err != nil {
		t.Fatalf("fetch before open: %v", err)
	}
	beforeTimed := findMessage(before, timed.ID)
	beforeOnce := findMessage(before, viewOnce.ID)
	if beforeTimed == nil || !beforeTimed.ContentHidden || beforeTimed.Content != nil || len(beforeTimed.Attachments) != 0 {
		t.Fatalf("timed message leaked before open: %#v", beforeTimed)
	}
	if beforeOnce == nil || !beforeOnce.ContentHidden || len(beforeOnce.Attachments) != 0 {
		t.Fatalf("view-once message leaked before open: %#v", beforeOnce)
	}
	privateChat, err := repo.GetPrivateChatBetweenUsers(author.ID, recipientB.ID)
	if err != nil {
		t.Fatalf("private chat fetch: %v", err)
	}
	if privateTimed := findMessage(privateChat.Messages, timed.ID); privateTimed == nil || !privateTimed.ContentHidden || len(privateTimed.Attachments) != 0 {
		t.Fatalf("private chat creation response leaked unopened content: %#v", privateTimed)
	}

	openedA, err := repo.OpenMessage(context.Background(), &recipientA, chatID, timed.ID, now.Add(3*time.Second))
	if err != nil {
		t.Fatalf("recipient A open: %v", err)
	}
	if openedA.Message.OpenedAt == nil || openedA.Message.ExpiresAt == nil || !openedA.Message.ExpiresAt.Equal(now.Add(63*time.Second)) {
		t.Fatalf("timer was not started from open: %#v", openedA.Message)
	}
	if openedA.Message.Content == nil || len(openedA.Message.Attachments) != 1 || openedA.Message.Attachments[0].ID != timedAttachment.ID {
		t.Fatalf("opened regular message not full: %#v", openedA.Message)
	}

	reopenedA, err := repo.OpenMessage(context.Background(), &recipientA, chatID, timed.ID, now.Add(4*time.Second))
	if err != nil || reopenedA.Message.ExpiresAt == nil || !reopenedA.Message.ExpiresAt.Equal(*openedA.Message.ExpiresAt) {
		t.Fatalf("regular reopen must be idempotent: result=%#v error=%v", reopenedA, err)
	}
	fetchedA, err := repo.GetMessagesByChatID(recipientA.ID, chatID)
	if err != nil || findMessage(fetchedA, timed.ID) == nil || findMessage(fetchedA, timed.ID).ContentHidden {
		t.Fatalf("recipient A should retain full timed DTO before expiry: %#v error=%v", findMessage(fetchedA, timed.ID), err)
	}
	fetchedB, err := repo.GetMessagesByChatID(recipientB.ID, chatID)
	if err != nil {
		t.Fatal(err)
	}
	if messageB := findMessage(fetchedB, timed.ID); messageB == nil || !messageB.ContentHidden || len(messageB.Attachments) != 0 {
		t.Fatalf("recipient B leaked content after A opened: %#v", messageB)
	}
	openedB, err := repo.OpenMessage(context.Background(), &recipientB, chatID, timed.ID, now.Add(5*time.Second))
	if err != nil || openedB.Message.Content == nil || openedB.Message.ExpiresAt == nil || !openedB.Message.ExpiresAt.Equal(*openedA.Message.ExpiresAt) {
		t.Fatalf("recipient B could not open within global TTL: result=%#v error=%v", openedB, err)
	}

	if _, err := repo.OpenMessage(context.Background(), &author, chatID, timed.ID, now.Add(6*time.Second)); !errors.Is(err, chat.ErrAuthorCannotOpen) {
		t.Fatalf("sender error = %v", err)
	}
	if _, err := repo.OpenMessage(context.Background(), &outsider, chatID, timed.ID, now.Add(6*time.Second)); !errors.Is(err, chat.ErrNotParticipant) {
		t.Fatalf("outsider error = %v", err)
	}
	if _, err := repo.OpenMessage(context.Background(), &recipientA, chatID, permanent.ID, now.Add(6*time.Second)); !errors.Is(err, chat.ErrNotDisappearing) {
		t.Fatalf("ordinary message error = %v", err)
	}
	if _, err := repo.OpenMessage(context.Background(), &recipientA, chatID, timed.ID, now.Add(64*time.Second)); !errors.Is(err, chat.ErrMessageExpired) {
		t.Fatalf("expired message error = %v", err)
	}

	openedOnce, err := repo.OpenMessage(context.Background(), &recipientA, chatID, viewOnce.ID, now.Add(7*time.Second))
	if err != nil {
		t.Fatalf("first view-once open: %v", err)
	}
	if openedOnce.Media == nil || openedOnce.Media.ID != viewOnceAttachment.ID || len(openedOnce.Message.Attachments) != 0 {
		t.Fatalf("invalid view-once DTO: %#v", openedOnce)
	}
	decoded, err := base64.StdEncoding.DecodeString(openedOnce.Media.DataBase64)
	if err != nil || string(decoded) != "one-view-image" {
		t.Fatalf("invalid base64 media: %q error=%v", decoded, err)
	}
	payload, err := json.Marshal(openedOnce)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{viewOnceAttachment.File.URL, viewOnceAttachment.File.StoragePath, "storage_path", "variants"} {
		if forbidden != "" && strings.Contains(string(payload), forbidden) {
			t.Fatalf("view-once open leaked %q: %s", forbidden, payload)
		}
	}
	if _, err := repo.OpenMessage(context.Background(), &recipientA, chatID, viewOnce.ID, now.Add(8*time.Second)); !errors.Is(err, chat.ErrMessageAlreadySeen) {
		t.Fatalf("second view-once error = %v", err)
	}

	var mediaCount, fileCount int64
	if err := db.Model(&media.Media{}).Where("id = ?", viewOnceAttachment.ID).Count(&mediaCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&modelutils.FileMetadata{}).Where("id = ?", viewOnceAttachment.FileID).Count(&fileCount).Error; err != nil {
		t.Fatal(err)
	}
	if mediaCount != 1 || fileCount != 1 {
		t.Fatalf("view consumption changed stored media rows: media=%d file=%d", mediaCount, fileCount)
	}
	if _, err := os.Stat(viewOnceAttachment.File.StoragePath); err != nil {
		t.Fatalf("view consumption removed physical media: %v", err)
	}
}
