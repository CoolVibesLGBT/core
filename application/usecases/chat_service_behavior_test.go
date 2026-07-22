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
	"errors"
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

func TestChatServiceCreatePrivateChatDoesNotTreatRepositoryFailureAsMissing(t *testing.T) {
	authID := uuid.New()
	participantID := uuid.New()
	userRepo := &fakeUserRepository{byUUID: map[uuid.UUID]*models.User{
		participantID: {ID: participantID, PublicID: 2},
	}}
	repositoryFailure := errors.New("database unavailable")
	chatRepo := &fakeChatRepository{privateChatErr: repositoryFailure}
	service := NewChatService(&fakeRealtimeNotifier{}, userRepo, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, chatRepo, &fakeNotificationRepository{})

	_, err := service.CreateChat(context.Background(), participantID, authID, string(chat.ChatTypePrivate))
	if !errors.Is(err, repositoryFailure) {
		t.Fatalf("CreateChat() error = %v, want repository failure", err)
	}
	if chatRepo.createdPrivate != nil {
		t.Fatal("repository failure was mistaken for a missing chat")
	}
}

func TestChatServiceCreateChatPreservesParticipantLookupFailures(t *testing.T) {
	authID := uuid.New()
	authUser := &models.User{ID: authID, PublicID: 1}
	lookupFailure := errors.New("user database unavailable")
	tests := []struct {
		name       string
		identifier string
		userRepo   *fakeUserRepository
	}{
		{name: "legacy uuid", identifier: uuid.NewString(), userRepo: &fakeUserRepository{byUUIDErr: lookupFailure}},
		{name: "public id", identifier: "987654321", userRepo: &fakeUserRepository{byPublicIDErr: lookupFailure}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatRepo := &fakeChatRepository{}
			service := NewChatService(&fakeRealtimeNotifier{}, tt.userRepo, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, chatRepo, &fakeNotificationRepository{})
			_, err := service.CreateChatFromIdentifier(context.Background(), tt.identifier, authUser, string(chat.ChatTypePrivate))
			if !errors.Is(err, lookupFailure) {
				t.Fatalf("error = %v, want lookup failure", err)
			}
			if chatRepo.createdPrivate != nil {
				t.Fatal("participant lookup failure triggered chat creation")
			}
		})
	}
}

func TestChatServiceCreateChatFromIdentifierSupportsPublicIDAndLegacyUUID(t *testing.T) {
	authID := uuid.New()
	authUser := &models.User{ID: authID, PublicID: 1}
	publicParticipant := &models.User{ID: uuid.New(), PublicID: 987654321}
	legacyParticipant := &models.User{ID: uuid.New(), PublicID: 987654322}
	userRepo := &fakeUserRepository{
		byPublicID: map[int64]*models.User{publicParticipant.PublicID: publicParticipant},
		byUUID:     map[uuid.UUID]*models.User{legacyParticipant.ID: legacyParticipant},
	}

	tests := []struct {
		name       string
		identifier string
		wantID     uuid.UUID
	}{
		{name: "public snowflake", identifier: "987654321", wantID: publicParticipant.ID},
		{name: "legacy uuid", identifier: legacyParticipant.ID.String(), wantID: legacyParticipant.ID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatRepo := &fakeChatRepository{}
			service := NewChatService(&fakeRealtimeNotifier{}, userRepo, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, chatRepo, &fakeNotificationRepository{})
			if _, err := service.CreateChatFromIdentifier(context.Background(), tt.identifier, authUser, string(chat.ChatTypePrivate)); err != nil {
				t.Fatalf("CreateChatFromIdentifier() error = %v", err)
			}
			if chatRepo.createdPrivate == nil || chatRepo.createdPrivate.CreatorID != authID {
				t.Fatalf("chat was not created by auth user: %#v", chatRepo.createdPrivate)
			}
			if chatRepo.privateFrom != tt.wantID || chatRepo.privateTo != authID {
				t.Fatalf("resolved participant lookup = %s/%s, want %s/%s", chatRepo.privateFrom, chatRepo.privateTo, tt.wantID, authID)
			}
		})
	}
}

func TestChatServiceCreateChatFromIdentifierRejectsInvalidAndSelfPublicID(t *testing.T) {
	authID := uuid.New()
	authUser := &models.User{ID: authID, PublicID: 123456789}
	userRepo := &fakeUserRepository{byPublicID: map[int64]*models.User{authUser.PublicID: authUser}}
	service := NewChatService(&fakeRealtimeNotifier{}, userRepo, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, &fakeChatRepository{}, &fakeNotificationRepository{})

	for _, identifier := range []string{"0", "-1", "not-an-id", "9223372036854775808"} {
		if _, err := service.CreateChatFromIdentifier(context.Background(), identifier, authUser, string(chat.ChatTypePrivate)); err == nil {
			t.Fatalf("identifier %q should be rejected", identifier)
		}
	}
	if _, err := service.CreateChatFromIdentifier(context.Background(), "123456789", authUser, string(chat.ChatTypePrivate)); err == nil {
		t.Fatal("self chat via public ID should be rejected")
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

func TestChatServiceAddMessageDoesNotFailCommittedSendWhenRealtimeDeliveryFails(t *testing.T) {
	chatID := uuid.New()
	author := &models.User{ID: uuid.New(), PublicID: 1}
	message := &post.Post{ID: uuid.New(), AuthorID: author.ID, ContentableID: &chatID, PostKind: post.PostKindMessage}
	notifier := &fakeRealtimeNotifier{err: errors.New("socket unavailable")}
	service := NewChatService(notifier, &fakeUserRepository{}, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, &fakeChatRepository{message: message}, &fakeNotificationRepository{})

	created, err := service.AddMessageToChat(context.Background(), ports.FormData{}, author)
	if err != nil {
		t.Fatalf("committed send returned realtime error: %v", err)
	}
	if created == nil || created.ID != message.ID {
		t.Fatalf("created message = %#v, want %s", created, message.ID)
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
	viewer := &models.User{ID: uuid.New(), PublicID: 778899}
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
	for _, expected := range []string{constants.CMD_CHAT_MESSAGE_OPENED, chatID.String(), messageID.String(), "778899", "expires_at"} {
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
	user := &models.User{ID: userID, PublicID: 445566}
	notifier := &fakeRealtimeNotifier{}
	service := NewChatService(notifier, &fakeUserRepository{}, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, &fakeChatRepository{}, &fakeNotificationRepository{})

	if err := service.SendTypingEvent(chatID, user, true); err != nil {
		t.Fatalf("SendTypingEvent() error = %v", err)
	}
	if notifier.room != chatID.String() {
		t.Fatalf("expected room %s, got %q", chatID, notifier.room)
	}
	if !strings.Contains(notifier.msg, "445566") || strings.Contains(notifier.msg, userID.String()) {
		t.Fatalf("expected typing payload to contain only the public user id, got %s", notifier.msg)
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
