package repositories

import (
	"context"
	"core/constants"
	"core/models"
	"core/models/chat"
	"core/models/media"
	"core/models/post"
	modelutils "core/models/utils"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func expiryIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" && os.Getenv("ENV") == "test" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin test transaction: %v", tx.Error)
	}
	t.Cleanup(func() { tx.Rollback() })
	return tx
}

func TestExpireMessagesSoftDeletesAndRepairsChatWithoutDeletingMedia(t *testing.T) {
	db := expiryIntegrationDB(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	basePublicID := now.UnixNano()
	authorID := uuid.New()
	receiverID := uuid.New()

	users := []models.User{
		{ID: authorID, PublicID: basePublicID, Domain: models.CoolVibes, UserName: "expiry-author-" + uuid.NewString(), DisplayName: "Expiry Author", UserRole: constants.UserRoleUser},
		{ID: receiverID, PublicID: basePublicID + 1, Domain: models.CoolVibes, UserName: "expiry-receiver-" + uuid.NewString(), DisplayName: "Expiry Receiver", UserRole: constants.UserRoleUser},
	}
	if err := db.Omit(clause.Associations).Create(&users).Error; err != nil {
		t.Fatalf("create test users: %v", err)
	}

	chatID := uuid.New()
	contentableType := string(post.PostKindChat)
	persistentMessage := post.Post{
		ID: uuid.New(), PublicID: basePublicID + 2, PostKind: post.PostKindMessage,
		Domain: models.CoolVibes, ContentableID: &chatID, ContentableType: &contentableType,
		AuthorID: authorID, Published: true, CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now.Add(-2 * time.Minute),
	}
	expiresAt := now.Add(-time.Second)
	openedAt := now.Add(-time.Minute)
	expiredMessage := post.Post{
		ID: uuid.New(), PublicID: basePublicID + 3, PostKind: post.PostKindMessage,
		Domain: models.CoolVibes, ContentableID: &chatID, ContentableType: &contentableType,
		AuthorID: authorID, Published: true, OpenedAt: &openedAt, ExpiresAt: &expiresAt,
		CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute),
	}
	unopenedDuration := 60
	unopenedMessage := post.Post{
		ID: uuid.New(), PublicID: basePublicID + 5, PostKind: post.PostKindMessage,
		Domain: models.CoolVibes, ContentableID: &chatID, ContentableType: &contentableType,
		AuthorID: authorID, Published: true, ExpiresInSeconds: &unopenedDuration, ExpiresAt: &expiresAt,
		CreatedAt: now.Add(-4 * time.Minute), UpdatedAt: now.Add(-4 * time.Minute),
	}
	if err := db.Omit(clause.Associations).Create(&[]post.Post{persistentMessage, expiredMessage, unopenedMessage}).Error; err != nil {
		t.Fatalf("create test messages: %v", err)
	}

	chatEntity := chat.Chat{
		ID: chatID, Type: chat.ChatTypePrivate, CreatorID: authorID,
		PinnedMsgID: &expiredMessage.ID, PinnedByID: &authorID,
		LastMessageID: &expiredMessage.ID, LastMessageTimestamp: &expiredMessage.CreatedAt,
		CreatedAt: now.Add(-3 * time.Minute), UpdatedAt: now.Add(-time.Minute),
	}
	if err := db.Omit(clause.Associations).Create(&chatEntity).Error; err != nil {
		t.Fatalf("create test chat: %v", err)
	}
	participants := []chat.ChatParticipant{
		{ID: uuid.New(), ChatID: chatID, UserID: authorID, JoinedAt: now.Add(-3 * time.Minute)},
		{ID: uuid.New(), ChatID: chatID, UserID: receiverID, JoinedAt: now.Add(-3 * time.Minute), UnreadCount: 1},
	}
	if err := db.Omit(clause.Associations).Create(&participants).Error; err != nil {
		t.Fatalf("create test participants: %v", err)
	}

	// In a second chat, the receiver has already read/cleared the expiring
	// message but still has one newer unread message. Expiry must not blindly
	// decrement that newer unread count.
	readChatID := uuid.New()
	readContentableType := string(post.PostKindChat)
	readExpiredAt := now.Add(-time.Second)
	readOpenedAt := now.Add(-3 * time.Minute)
	readExpiredMessage := post.Post{
		ID: uuid.New(), PublicID: basePublicID + 10, PostKind: post.PostKindMessage,
		Domain: models.CoolVibes, ContentableID: &readChatID, ContentableType: &readContentableType,
		AuthorID: authorID, Published: true, OpenedAt: &readOpenedAt, ExpiresAt: &readExpiredAt,
		CreatedAt: now.Add(-3 * time.Minute), UpdatedAt: now.Add(-3 * time.Minute),
	}
	newerUnreadMessage := post.Post{
		ID: uuid.New(), PublicID: basePublicID + 11, PostKind: post.PostKindMessage,
		Domain: models.CoolVibes, ContentableID: &readChatID, ContentableType: &readContentableType,
		AuthorID: authorID, Published: true,
		CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute),
	}
	if err := db.Omit(clause.Associations).Create(&[]post.Post{readExpiredMessage, newerUnreadMessage}).Error; err != nil {
		t.Fatalf("create read-boundary messages: %v", err)
	}
	readChat := chat.Chat{
		ID: readChatID, Type: chat.ChatTypePrivate, CreatorID: authorID,
		LastMessageID: &newerUnreadMessage.ID, LastMessageTimestamp: &newerUnreadMessage.CreatedAt,
		CreatedAt: now.Add(-4 * time.Minute), UpdatedAt: now.Add(-time.Minute),
	}
	if err := db.Omit(clause.Associations).Create(&readChat).Error; err != nil {
		t.Fatalf("create read-boundary chat: %v", err)
	}
	readBoundary := now.Add(-2 * time.Minute)
	readParticipants := []chat.ChatParticipant{
		{ID: uuid.New(), ChatID: readChatID, UserID: authorID, JoinedAt: now.Add(-4 * time.Minute)},
		{
			ID: uuid.New(), ChatID: readChatID, UserID: receiverID, JoinedAt: now.Add(-4 * time.Minute),
			UnreadCount: 1, LastReadAt: &readBoundary, ClearedAt: &readBoundary,
		},
	}
	if err := db.Omit(clause.Associations).Create(&readParticipants).Error; err != nil {
		t.Fatalf("create read-boundary participants: %v", err)
	}

	physicalPath := filepath.Join(t.TempDir(), "preserved-chat-image.jpg")
	if err := os.WriteFile(physicalPath, []byte("image"), 0o600); err != nil {
		t.Fatalf("create physical media fixture: %v", err)
	}
	fileID := uuid.New()
	file := modelutils.FileMetadata{
		ID: fileID, URL: "/static/test/preserved-chat-image.jpg", StoragePath: physicalPath,
		MimeType: "image/jpeg", Size: 5, Name: "preserved-chat-image.jpg", CreatedAt: now.Add(-time.Minute),
	}
	attachment := media.Media{
		ID: uuid.New(), PublicID: basePublicID + 4, FileID: fileID,
		OwnerID: expiredMessage.ID, OwnerType: media.OwnerPost, UserID: authorID,
		Role: media.RoleChatMedia, ProcessingStatus: media.ProcessingStatusReady,
		CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute),
	}
	if err := db.Omit(clause.Associations).Create(&file).Error; err != nil {
		t.Fatalf("create file metadata: %v", err)
	}
	if err := db.Omit(clause.Associations).Create(&attachment).Error; err != nil {
		t.Fatalf("create media attachment: %v", err)
	}

	repo := &ChatRepository{db: db}
	expired, err := repo.ExpireMessages(context.Background(), now, 100)
	if err != nil {
		t.Fatalf("ExpireMessages() error = %v", err)
	}
	if len(expired) != 2 {
		t.Fatalf("ExpireMessages() = %#v, want two messages", expired)
	}
	expiredIDs := map[uuid.UUID]bool{}
	for _, item := range expired {
		expiredIDs[item.MessageID] = true
	}
	if !expiredIDs[expiredMessage.ID] || !expiredIDs[readExpiredMessage.ID] {
		t.Fatalf("ExpireMessages() = %#v, missing expected message", expired)
	}

	var deleted post.Post
	if err := db.Unscoped().First(&deleted, "id = ?", expiredMessage.ID).Error; err != nil || !deleted.DeletedAt.Valid {
		t.Fatalf("expected soft-deleted post, post=%#v error=%v", deleted, err)
	}
	var unopened post.Post
	if err := db.Unscoped().First(&unopened, "id = ?", unopenedMessage.ID).Error; err != nil || unopened.DeletedAt.Valid {
		t.Fatalf("unopened disappearing message must not expire, post=%#v error=%v", unopened, err)
	}
	var repaired chat.Chat
	if err := db.First(&repaired, "id = ?", chatID).Error; err != nil {
		t.Fatalf("load repaired chat: %v", err)
	}
	if repaired.PinnedMsgID != nil || repaired.PinnedByID != nil {
		t.Fatalf("expected expired pin to be cleared, got message=%v by=%v", repaired.PinnedMsgID, repaired.PinnedByID)
	}
	if repaired.LastMessageID == nil || *repaired.LastMessageID != persistentMessage.ID || repaired.LastMessageTimestamp == nil || !repaired.LastMessageTimestamp.Equal(persistentMessage.CreatedAt) {
		t.Fatalf("unexpected repaired last message: id=%v timestamp=%v", repaired.LastMessageID, repaired.LastMessageTimestamp)
	}
	var receiver chat.ChatParticipant
	if err := db.Where("chat_id = ? AND user_id = ?", chatID, receiverID).First(&receiver).Error; err != nil {
		t.Fatalf("load receiver participant: %v", err)
	}
	if receiver.UnreadCount != 0 {
		t.Fatalf("expected unread count 0 after expiry, got %d", receiver.UnreadCount)
	}
	var readReceiver chat.ChatParticipant
	if err := db.Where("chat_id = ? AND user_id = ?", readChatID, receiverID).First(&readReceiver).Error; err != nil {
		t.Fatalf("load read-boundary receiver: %v", err)
	}
	if readReceiver.UnreadCount != 1 {
		t.Fatalf("expiry of an already read/cleared message changed newer unread count to %d", readReceiver.UnreadCount)
	}
	var repairedReadChat chat.Chat
	if err := db.First(&repairedReadChat, "id = ?", readChatID).Error; err != nil {
		t.Fatalf("load repaired read-boundary chat: %v", err)
	}
	if repairedReadChat.LastMessageID == nil || *repairedReadChat.LastMessageID != newerUnreadMessage.ID {
		t.Fatalf("expected newer unread message to remain last, got %v", repairedReadChat.LastMessageID)
	}

	var mediaCount, fileCount int64
	db.Model(&media.Media{}).Where("id = ?", attachment.ID).Count(&mediaCount)
	db.Model(&modelutils.FileMetadata{}).Where("id = ?", fileID).Count(&fileCount)
	if mediaCount != 1 || fileCount != 1 {
		t.Fatalf("expiry must preserve media rows, media=%d file=%d", mediaCount, fileCount)
	}
	if _, err := os.Stat(physicalPath); err != nil {
		t.Fatalf("expiry must preserve physical media: %v", err)
	}

	second, err := repo.ExpireMessages(context.Background(), now, 100)
	if err != nil || len(second) != 0 {
		t.Fatalf("second sweep should be idempotent, expired=%#v error=%v", second, err)
	}
}
