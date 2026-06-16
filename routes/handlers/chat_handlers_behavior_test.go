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
		"chat_id": uuid.New().String(),
		"content": "hello",
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
	if notifier.event != "chat" {
		t.Fatalf("expected chat broadcast, got event=%q", notifier.event)
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
