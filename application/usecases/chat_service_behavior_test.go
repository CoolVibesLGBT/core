package usecases

import (
	"context"
	"core/application/ports"
	"core/constants"
	"core/models"
	"core/models/chat"
	"core/models/media"
	"core/models/post"
	modelutils "core/models/utils"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestChatServiceCreatePrivateChatCreatesWhenMissing(t *testing.T) {
	authID := uuid.New()
	participantID := uuid.New()
	userRepo := &fakeUserRepository{
		byUUID: map[uuid.UUID]*models.User{
			participantID: {ID: participantID, PublicID: 2},
		},
	}
	chatRepo := &fakeChatRepository{}
	service := NewChatService(&fakeRealtimeNotifier{}, userRepo, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, chatRepo, &fakeNotificationRepository{})

	created, err := service.CreateChat(context.Background(), participantID, authID, string(chat.ChatTypePrivate))
	if err != nil {
		t.Fatalf("CreateChat() error = %v", err)
	}
	if created == nil || chatRepo.createdPrivate == nil {
		t.Fatalf("expected private chat to be created, got %#v", created)
	}
	if chatRepo.createdPrivate.CreatorID != authID {
		t.Fatalf("expected auth user to be creator, got %s", chatRepo.createdPrivate.CreatorID)
	}
}

func TestChatServiceCreatePrivateChatRejectsSelf(t *testing.T) {
	authID := uuid.New()
	userRepo := &fakeUserRepository{
		byUUID: map[uuid.UUID]*models.User{
			authID: {ID: authID, PublicID: 1},
		},
	}
	service := NewChatService(&fakeRealtimeNotifier{}, userRepo, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, &fakeChatRepository{}, &fakeNotificationRepository{})

	_, err := service.CreateChat(context.Background(), authID, authID, string(chat.ChatTypePrivate))
	if err == nil {
		t.Fatalf("expected self chat to fail")
	}
}

func TestChatServiceAddMessageBroadcastsToChatRoom(t *testing.T) {
	chatID := uuid.New()
	author := &models.User{ID: uuid.New(), PublicID: 1}
	message := &post.Post{ID: uuid.New(), AuthorID: author.ID, ContentableID: &chatID, PostKind: post.PostKindMessage}
	chatRepo := &fakeChatRepository{message: message}
	notifier := &fakeRealtimeNotifier{}
	service := NewChatService(notifier, &fakeUserRepository{}, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, chatRepo, &fakeNotificationRepository{})

	created, err := service.AddMessageToChat(context.Background(), ports.FormData{}, author)
	if err != nil {
		t.Fatalf("AddMessageToChat() error = %v", err)
	}
	if created.ID != message.ID {
		t.Fatalf("expected created message %s, got %s", message.ID, created.ID)
	}
	if notifier.room != chatID.String() {
		t.Fatalf("expected broadcast room %s, got %q", chatID, notifier.room)
	}
	if notifier.event != "chat" {
		t.Fatalf("expected chat event, got %q", notifier.event)
	}
	if !strings.Contains(notifier.msg, message.ID.String()) {
		t.Fatalf("expected broadcast payload to contain message id, got %s", notifier.msg)
	}
}

func TestChatServiceViewOnceSendNeverBroadcastsOrReturnsStaticLocator(t *testing.T) {
	chatID := uuid.New()
	author := &models.User{ID: uuid.New(), PublicID: 1}
	message := &post.Post{
		ID: uuid.New(), AuthorID: author.ID, ContentableID: &chatID,
		PostKind: post.PostKindMessage, ViewOnce: true,
		Attachments: []*media.Media{{ID: uuid.New(), File: modelutils.FileMetadata{
			URL: "/static/secret.jpg", StoragePath: "./static/secret.jpg",
		}}},
	}
	chatRepo := &fakeChatRepository{message: message}
	notifier := &fakeRealtimeNotifier{}
	service := NewChatService(notifier, &fakeUserRepository{}, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, chatRepo, &fakeNotificationRepository{})

	created, err := service.AddMessageToChat(context.Background(), ports.FormData{}, author)
	if err != nil {
		t.Fatalf("AddMessageToChat() error = %v", err)
	}
	if len(created.Attachments) != 0 || !created.ContentHidden {
		t.Fatalf("view-once HTTP response leaked attachments: %#v", created)
	}
	for _, forbidden := range []string{"/static/secret.jpg", "storage_path", "variants"} {
		if strings.Contains(notifier.msg, forbidden) {
			t.Fatalf("view-once broadcast leaked %q: %s", forbidden, notifier.msg)
		}
	}
}

func TestChatServiceOpenMessageBroadcastsMetadataOnly(t *testing.T) {
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	chatID := uuid.New()
	messageID := uuid.New()
	viewer := &models.User{ID: uuid.New()}
	expiresAt := now.Add(time.Minute)
	opened := &post.Post{ID: messageID, ContentableID: &chatID, OpenedAt: &now, ExpiresAt: &expiresAt}
	chatRepo := &fakeChatRepository{openResult: &chat.OpenMessageResult{Message: opened}}
	notifier := &fakeRealtimeNotifier{}
	service := NewChatService(notifier, &fakeUserRepository{}, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, chatRepo, &fakeNotificationRepository{})

	result, err := service.OpenMessage(context.Background(), viewer, chatID, messageID, now)
	if err != nil || result.Message.ID != messageID {
		t.Fatalf("OpenMessage() result=%#v error=%v", result, err)
	}
	if chatRepo.openAuthUser != viewer || chatRepo.openChatID != chatID || chatRepo.openMessageID != messageID || !chatRepo.openNow.Equal(now) {
		t.Fatalf("unexpected open delegation: %#v", chatRepo)
	}
	for _, expected := range []string{constants.CMD_CHAT_MESSAGE_OPENED, chatID.String(), messageID.String(), viewer.ID.String(), "expires_at"} {
		if !strings.Contains(notifier.msg, expected) {
			t.Fatalf("opened event %q does not contain %q", notifier.msg, expected)
		}
	}
	if strings.Contains(notifier.msg, "attachments") || strings.Contains(notifier.msg, "content") {
		t.Fatalf("opened event must be metadata-only: %s", notifier.msg)
	}
}

func TestChatServiceSendTypingEventBroadcastsPayload(t *testing.T) {
	chatID := uuid.New()
	userID := uuid.New()
	notifier := &fakeRealtimeNotifier{}
	service := NewChatService(notifier, &fakeUserRepository{}, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, &fakeChatRepository{}, &fakeNotificationRepository{})

	if err := service.SendTypingEvent(chatID, userID, true); err != nil {
		t.Fatalf("SendTypingEvent() error = %v", err)
	}
	if notifier.room != chatID.String() {
		t.Fatalf("expected room %s, got %q", chatID, notifier.room)
	}
	if !strings.Contains(notifier.msg, userID.String()) {
		t.Fatalf("expected typing payload to contain user id, got %s", notifier.msg)
	}
}

func TestChatServiceExpireMessagesBroadcastsExpiredAction(t *testing.T) {
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	chatID := uuid.New()
	messageID := uuid.New()
	chatRepo := &fakeChatRepository{expiredMessages: []chat.ExpiredMessage{{
		ChatID: chatID, MessageID: messageID, ExpiredAt: now.Add(-time.Second),
	}}}
	notifier := &fakeRealtimeNotifier{}
	service := NewChatService(notifier, &fakeUserRepository{}, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, chatRepo, &fakeNotificationRepository{})

	count, err := service.ExpireMessages(context.Background(), now, 100)
	if err != nil {
		t.Fatalf("ExpireMessages() error = %v", err)
	}
	if count != 1 || !chatRepo.expireNow.Equal(now) || chatRepo.expireLimit != 100 {
		t.Fatalf("unexpected expiry delegation count=%d now=%v limit=%d", count, chatRepo.expireNow, chatRepo.expireLimit)
	}
	if notifier.room != chatID.String() || notifier.event != "chat" {
		t.Fatalf("unexpected expiry broadcast room=%q event=%q", notifier.room, notifier.event)
	}
	for _, expected := range []string{constants.CMD_CHAT_MESSAGE_EXPIRED, chatID.String(), messageID.String(), "expired_at"} {
		if !strings.Contains(notifier.msg, expected) {
			t.Fatalf("expiry payload %q does not contain %q", notifier.msg, expected)
		}
	}
}
