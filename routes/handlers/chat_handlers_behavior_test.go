package handlers

import (
	"context"
	"core/application/ports"
	usecases "core/application/usecases"
	"core/models"
	"core/models/chat"
	"core/models/post"
	"core/types"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type chatHandlerUserRepo struct {
	ports.UserRepository
	byUUID map[uuid.UUID]*models.User
}

func (r *chatHandlerUserRepo) GetUserByUUIDdWithoutRelations(filters types.Filter) (*models.User, error) {
	if user, ok := r.byUUID[filters.UserUUID]; ok {
		return user, nil
	}
	return nil, errors.New("user not found")
}

type chatHandlerRepo struct {
	ports.ChatRepository
	typingChatID   uuid.UUID
	typingUserID   uuid.UUID
	typingValue    bool
	privateFrom    uuid.UUID
	privateTo      uuid.UUID
	createdFrom    uuid.UUID
	createdTo      uuid.UUID
	messageAuthor  *models.User
	messageForm    ports.FormData
	markedChatID   uuid.UUID
	markedMessages []uuid.UUID
	openResult     *chat.OpenMessageResult
	openErr        error
	openAuthUser   *models.User
	openChatID     uuid.UUID
	openMessageID  uuid.UUID
}

func (r *chatHandlerRepo) SendTypingEvent(chatID, userID uuid.UUID, typing bool) (map[string]interface{}, error) {
	r.typingChatID = chatID
	r.typingUserID = userID
	r.typingValue = typing
	return map[string]interface{}{"chat_id": chatID.String(), "user_id": userID.String(), "typing": typing}, nil
}

func (r *chatHandlerRepo) GetPrivateChatBetweenUsers(fromUser, toUser uuid.UUID) (*chat.Chat, error) {
	r.privateFrom = fromUser
	r.privateTo = toUser
	return nil, errors.New("not found")
}

func (r *chatHandlerRepo) CreatePrivateChat(fromUser, toUser uuid.UUID) (*chat.Chat, error) {
	r.createdFrom = fromUser
	r.createdTo = toUser
	return &chat.Chat{ID: uuid.New(), Type: chat.ChatTypePrivate, CreatorID: fromUser}, nil
}

func (r *chatHandlerRepo) AddMessageToChat(ctx context.Context, form ports.FormData, author *models.User) (*post.Post, error) {
	r.messageAuthor = author
	r.messageForm = form
	chatID := uuid.New()
	return &post.Post{ID: uuid.New(), PublicID: 9001, AuthorID: author.ID, ContentableID: &chatID, PostKind: post.PostKindMessage}, nil
}

func (r *chatHandlerRepo) MarkChatMessageRead(ctx context.Context, authUser *models.User, chatID uuid.UUID, messages []uuid.UUID) error {
	r.markedChatID = chatID
	r.markedMessages = messages
	return nil
}

func (r *chatHandlerRepo) OpenMessage(ctx context.Context, authUser *models.User, chatID, messageID uuid.UUID, now time.Time) (*chat.OpenMessageResult, error) {
	r.openAuthUser = authUser
	r.openChatID = chatID
	r.openMessageID = messageID
	return r.openResult, r.openErr
}

type chatHandlerNotifier struct {
	room  string
	event string
	msg   string
}

func (n *chatHandlerNotifier) BroadcastToRoom(namespace string, room string, event string, msg string) error {
	n.room = room
	n.event = event
	n.msg = msg
	return nil
}

func newChatHandlerService(userRepo ports.UserRepository, chatRepo *chatHandlerRepo, notifier *chatHandlerNotifier) *usecases.ChatService {
	return usecases.NewChatService(notifier, userRepo, &handlerPostRepo{}, &handlerMediaRepo{}, nil, chatRepo, &handlerNotificationRepo{})
}

func TestHandleCreateChatParsesParticipantAndCreatesPrivateChat(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	participantID := uuid.New()
	chatRepo := &chatHandlerRepo{}
	service := newChatHandlerService(&chatHandlerUserRepo{
		byUUID: map[uuid.UUID]*models.User{participantID: {ID: participantID, PublicID: 11}},
	}, chatRepo, &chatHandlerNotifier{})

	resp := performMultipartHandlerRequest(t, HandleCreateChat(service), authUser, map[string]string{
		"type":              string(chat.ChatTypePrivate),
		"participant_ids[]": participantID.String(),
	}, nil)

	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if chatRepo.privateFrom != participantID || chatRepo.privateTo != authUser.ID {
		t.Fatalf("expected lookup participant/auth user, got from=%s to=%s", chatRepo.privateFrom, chatRepo.privateTo)
	}
	if chatRepo.createdFrom != authUser.ID || chatRepo.createdTo != participantID {
		t.Fatalf("expected created private chat auth->participant, got from=%s to=%s", chatRepo.createdFrom, chatRepo.createdTo)
	}
}

func TestHandleSendTypingEventParsesChatIDAndBroadcasts(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	chatID := uuid.New()
	chatRepo := &chatHandlerRepo{}
	notifier := &chatHandlerNotifier{}
	service := newChatHandlerService(&chatHandlerUserRepo{}, chatRepo, notifier)

	resp := performMultipartHandlerRequest(t, HandleSendTypingEvent(service), authUser, map[string]string{
		"chat_id": chatID.String(),
	}, nil)

	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if chatRepo.typingChatID != chatID || chatRepo.typingUserID != authUser.ID || !chatRepo.typingValue {
		t.Fatalf("expected typing repo call, got chat=%s user=%s value=%v", chatRepo.typingChatID, chatRepo.typingUserID, chatRepo.typingValue)
	}
	if notifier.room != chatID.String() || notifier.event != "chat" || !strings.Contains(notifier.msg, authUser.ID.String()) {
		t.Fatalf("expected typing broadcast, got room=%q event=%q msg=%q", notifier.room, notifier.event, notifier.msg)
	}
}

func TestHandleSendMessagePassesMultipartFormAndAuthUser(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	chatRepo := &chatHandlerRepo{}
	notifier := &chatHandlerNotifier{}
	service := newChatHandlerService(&chatHandlerUserRepo{}, chatRepo, notifier)

	resp := performMultipartHandlerRequest(t, HandleSendMessage(service), authUser, map[string]string{
		"chat_id":            uuid.New().String(),
		"content":            "hello",
		"expires_in_seconds": "60",
	}, map[string][]byte{
		"images[]": []byte("image"),
	})

	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if chatRepo.messageAuthor == nil || chatRepo.messageAuthor.ID != authUser.ID {
		t.Fatalf("expected auth user to be message author, got %#v", chatRepo.messageAuthor)
	}
	if got := chatRepo.messageForm.Values["content"]; len(got) != 1 || got[0] != "hello" {
		t.Fatalf("expected content form value, got %#v", chatRepo.messageForm.Values)
	}
	if len(chatRepo.messageForm.Files) != 1 {
		t.Fatalf("expected one uploaded file, got %d", len(chatRepo.messageForm.Files))
	}
	if chatRepo.messageForm.ExpiresInSeconds == nil || *chatRepo.messageForm.ExpiresInSeconds != 60 {
		t.Fatalf("expected duration 60 without starting expiry, got %v", chatRepo.messageForm.ExpiresInSeconds)
	}
	if notifier.event != "chat" {
		t.Fatalf("expected chat broadcast, got event=%q", notifier.event)
	}
}

func TestHandleSendMessageRejectsInvalidExpiry(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	chatRepo := &chatHandlerRepo{}
	service := newChatHandlerService(&chatHandlerUserRepo{}, chatRepo, &chatHandlerNotifier{})

	resp := performMultipartHandlerRequest(t, HandleSendMessage(service), authUser, map[string]string{
		"chat_id":            uuid.New().String(),
		"content":            "hello",
		"expires_in_seconds": "9",
	}, nil)

	if resp.StatusCode != 400 {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
	if chatRepo.messageAuthor != nil {
		t.Fatal("repository should not be called for an invalid expiry")
	}
}

func TestHandleSendMessageParsesViewOnceAndClientID(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	chatRepo := &chatHandlerRepo{}
	service := newChatHandlerService(&chatHandlerUserRepo{}, chatRepo, &chatHandlerNotifier{})

	resp := performMultipartHandlerRequest(t, HandleSendMessage(service), authUser, map[string]string{
		"chat_id":   uuid.New().String(),
		"view_once": "on",
		"client_id": "optimistic-123",
	}, map[string][]byte{"images[]": []byte("image")})

	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if !chatRepo.messageForm.ViewOnce || chatRepo.messageForm.ImageCount != 1 || chatRepo.messageForm.VideoCount != 0 {
		t.Fatalf("unexpected view-once form: %#v", chatRepo.messageForm)
	}
	if chatRepo.messageForm.ClientID != "optimistic-123" {
		t.Fatalf("expected client id roundtrip input, got %q", chatRepo.messageForm.ClientID)
	}
}

func TestHandleChatMessageOpenReturnsPrivateNoStoreResponse(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	chatID := uuid.New()
	messageID := uuid.New()
	openedAt := time.Now().UTC()
	chatRepo := &chatHandlerRepo{openResult: &chat.OpenMessageResult{Message: &post.Post{
		ID: messageID, ContentableID: &chatID, OpenedAt: &openedAt,
	}}}
	service := newChatHandlerService(&chatHandlerUserRepo{}, chatRepo, &chatHandlerNotifier{})

	resp := performMultipartHandlerRequest(t, HandleChatMessageOpen(service), authUser, map[string]string{
		"chat_id": chatID.String(), "message_id": messageID.String(),
	}, nil)

	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Cache-Control"); !strings.Contains(got, "private") || !strings.Contains(got, "no-store") {
		t.Fatalf("expected private no-store cache control, got %q", got)
	}
	if chatRepo.openAuthUser != authUser || chatRepo.openChatID != chatID || chatRepo.openMessageID != messageID {
		t.Fatalf("unexpected open delegation: %#v", chatRepo)
	}
}

func TestHandleChatMessageOpenMapsLifecycleErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "sender", err: chat.ErrAuthorCannotOpen, want: 403},
		{name: "outsider", err: chat.ErrNotParticipant, want: 403},
		{name: "missing", err: chat.ErrMessageNotFound, want: 404},
		{name: "expired", err: chat.ErrMessageExpired, want: 410},
		{name: "already viewed", err: chat.ErrMessageAlreadySeen, want: 409},
		{name: "ordinary", err: chat.ErrNotDisappearing, want: 422},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authUser := &models.User{ID: uuid.New()}
			repo := &chatHandlerRepo{openErr: tt.err}
			service := newChatHandlerService(&chatHandlerUserRepo{}, repo, &chatHandlerNotifier{})
			resp := performMultipartHandlerRequest(t, HandleChatMessageOpen(service), authUser, map[string]string{
				"chat_id": uuid.New().String(), "message_id": uuid.New().String(),
			}, nil)
			if resp.StatusCode != tt.want {
				t.Fatalf("expected status %d, got %d", tt.want, resp.StatusCode)
			}
		})
	}
}

func TestHandleChatMessageReadParsesArrayMessageIDs(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	chatID := uuid.New()
	messageID := uuid.New()
	chatRepo := &chatHandlerRepo{}
	service := newChatHandlerService(&chatHandlerUserRepo{}, chatRepo, &chatHandlerNotifier{})

	resp := performMultipartHandlerRequest(t, HandleChatMessageRead(service), authUser, map[string]string{
		"chat_id":       chatID.String(),
		"message_ids[]": messageID.String(),
	}, nil)

	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if chatRepo.markedChatID != chatID || len(chatRepo.markedMessages) != 1 || chatRepo.markedMessages[0] != messageID {
		t.Fatalf("expected read marker call, got chat=%s messages=%#v", chatRepo.markedChatID, chatRepo.markedMessages)
	}
}

func TestHandleChatMessageReadRejectsInvalidMessageID(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	service := newChatHandlerService(&chatHandlerUserRepo{}, &chatHandlerRepo{}, &chatHandlerNotifier{})

	resp := performMultipartHandlerRequest(t, HandleChatMessageRead(service), authUser, map[string]string{
		"chat_id":       uuid.New().String(),
		"message_ids[]": "not-a-uuid",
	}, nil)

	if resp.StatusCode != 400 {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

var _ ports.UserRepository = (*chatHandlerUserRepo)(nil)
var _ ports.ChatRepository = (*chatHandlerRepo)(nil)
