package usecases

import (
	"context"
	"core/application/ports"
	"core/models"
	"core/models/chat"
	"core/models/post"
	"strings"
	"testing"

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
