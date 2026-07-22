package handlers

import (
	"context"
	"core/application/ports"
	"core/application/types"
	usecases "core/application/usecases"
	"core/models"
	"core/models/chat"
	"core/models/post"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type chatHandlerUserRepo struct {
	ports.UserRepository
	byUUID     map[uuid.UUID]*models.User
	byPublicID map[int64]*models.User
}

func (r *chatHandlerUserRepo) GetUserByUUIDdWithoutRelations(filters types.Filter) (*models.User, error) {
	if user, ok := r.byUUID[filters.UserUUID]; ok {
		return user, nil
	}
	return nil, ports.ErrNotFound
}

func (r *chatHandlerUserRepo) GetUserByPublicIdWithoutRelations(filters types.Filter) (*models.User, error) {
	if user, ok := r.byPublicID[filters.UserID]; ok {
		return user, nil
	}
	return nil, ports.ErrNotFound
}

type chatHandlerRepo struct {
	ports.ChatRepository
	typingChatID   uuid.UUID
	typingUserID   uuid.UUID
	typingValue    bool
	privateFrom    uuid.UUID
	privateTo      uuid.UUID
	privateErr     error
	createdFrom    uuid.UUID
	createdTo      uuid.UUID
	chatListQuery  ports.ChatListQuery
	chatListPage   ports.ChatListPage
	messageAuthor  *models.User
	messageForm    ports.FormData
	messageQuery   ports.ChatMessageListQuery
	messagePage    ports.ChatMessageListPage
	messageErr     error
	markedChatID   uuid.UUID
	markedMessages []uuid.UUID
	openResult     *chat.OpenMessageResult
	openErr        error
	openAuthUser   *models.User
	openChatID     uuid.UUID
	openMessageID  uuid.UUID
	mutationAction string
	mutationAuth   *models.User
	mutationChatID uuid.UUID
	mutationUserID uuid.UUID
	mutationMsgID  uuid.UUID
	mutationErr    error
}

func (r *chatHandlerRepo) SendTypingEvent(chatID, userID uuid.UUID, typing bool) error {
	r.typingChatID = chatID
	r.typingUserID = userID
	r.typingValue = typing
	return nil
}

func (r *chatHandlerRepo) GetPrivateChatBetweenUsers(fromUser, toUser uuid.UUID) (*chat.Chat, error) {
	r.privateFrom = fromUser
	r.privateTo = toUser
	if r.privateErr != nil {
		return nil, r.privateErr
	}
	return nil, chat.ErrChatNotFound
}

func (r *chatHandlerRepo) CreatePrivateChat(fromUser, toUser uuid.UUID) (*chat.Chat, error) {
	r.createdFrom = fromUser
	r.createdTo = toUser
	return &chat.Chat{ID: uuid.New(), Type: chat.ChatTypePrivate, CreatorID: fromUser}, nil
}

func (r *chatHandlerRepo) ListChats(_ context.Context, query ports.ChatListQuery) (ports.ChatListPage, error) {
	r.chatListQuery = query
	return r.chatListPage, nil
}

func (r *chatHandlerRepo) AddMessageToChat(ctx context.Context, form ports.FormData, author *models.User) (*post.Post, error) {
	r.messageAuthor = author
	r.messageForm = form
	chatID := uuid.New()
	return &post.Post{ID: uuid.New(), PublicID: 9001, AuthorID: author.ID, ContentableID: &chatID, PostKind: post.PostKindMessage}, nil
}

func (r *chatHandlerRepo) ListChatMessages(_ context.Context, query ports.ChatMessageListQuery) (ports.ChatMessageListPage, error) {
	r.messageQuery = query
	return r.messagePage, r.messageErr
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

func (r *chatHandlerRepo) recordMutation(action string, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	r.mutationAction = action
	r.mutationAuth = authUser
	r.mutationChatID = chatID
	r.mutationUserID = userID
	r.mutationMsgID = messageID
	return r.mutationErr
}

func (r *chatHandlerRepo) PinMessage(_ context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	return r.recordMutation("pin", authUser, chatID, userID, messageID)
}

func (r *chatHandlerRepo) UnpinMessage(_ context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	return r.recordMutation("unpin", authUser, chatID, userID, messageID)
}

func (r *chatHandlerRepo) DeleteMessageForAll(_ context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	return r.recordMutation("delete_message_for_all", authUser, chatID, userID, messageID)
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

func TestHandleCreateChatResolvesPublicSnowflakeParticipantID(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	participant := &models.User{ID: uuid.New(), PublicID: 912345678}
	chatRepo := &chatHandlerRepo{}
	service := newChatHandlerService(&chatHandlerUserRepo{
		byPublicID: map[int64]*models.User{participant.PublicID: participant},
	}, chatRepo, &chatHandlerNotifier{})

	resp := performMultipartHandlerRequest(t, HandleCreateChat(service), authUser, map[string]string{
		"type":            string(chat.ChatTypePrivate),
		"participant_ids": `[912345678]`,
	}, nil)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	if chatRepo.privateFrom != participant.ID || chatRepo.privateTo != authUser.ID {
		t.Fatalf("public participant was not resolved to internal identity: from=%s to=%s", chatRepo.privateFrom, chatRepo.privateTo)
	}
	if chatRepo.createdFrom != authUser.ID || chatRepo.createdTo != participant.ID {
		t.Fatalf("created private chat IDs = %s/%s", chatRepo.createdFrom, chatRepo.createdTo)
	}
}

func TestHandleCreateChatRejectsMultipleParticipantsInSingleParticipantCommand(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	chatRepo := &chatHandlerRepo{}
	service := newChatHandlerService(&chatHandlerUserRepo{}, chatRepo, &chatHandlerNotifier{})

	resp := performMultipartHandlerRequest(t, HandleCreateChat(service), authUser, map[string]string{
		"type":            string(chat.ChatTypePrivate),
		"participant_ids": `["123","456"]`,
	}, nil)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
	if chatRepo.createdFrom != uuid.Nil || chatRepo.createdTo != uuid.Nil {
		t.Fatal("multi-participant payload reached private chat creation")
	}
}

func TestHandleCreateChatRejectsAmbiguousParticipantFields(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	chatRepo := &chatHandlerRepo{}
	service := newChatHandlerService(&chatHandlerUserRepo{}, chatRepo, &chatHandlerNotifier{})

	resp := performMultipartHandlerRequest(t, HandleCreateChat(service), authUser, map[string]string{
		"type":              string(chat.ChatTypePrivate),
		"participant_ids":   `["123"]`,
		"participant_ids[]": "456",
	}, nil)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
	if chatRepo.createdFrom != uuid.Nil || chatRepo.createdTo != uuid.Nil {
		t.Fatal("ambiguous participant fields reached private chat creation")
	}
}

func TestHandleCreateChatRejectsMalformedParticipantIdentifier(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	chatRepo := &chatHandlerRepo{}
	service := newChatHandlerService(&chatHandlerUserRepo{}, chatRepo, &chatHandlerNotifier{})

	resp := performMultipartHandlerRequest(t, HandleCreateChat(service), authUser, map[string]string{
		"type":              string(chat.ChatTypePrivate),
		"participant_ids[]": "-12-not-an-id",
	}, nil)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
	if chatRepo.createdFrom != uuid.Nil || chatRepo.createdTo != uuid.Nil {
		t.Fatal("malformed participant identifier reached chat creation")
	}
}

func TestHandleCreateChatDoesNotMaskRepositoryFailureAsClientInput(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	participant := &models.User{ID: uuid.New(), PublicID: 11}
	chatRepo := &chatHandlerRepo{privateErr: errors.New("database unavailable")}
	service := newChatHandlerService(&chatHandlerUserRepo{
		byUUID: map[uuid.UUID]*models.User{participant.ID: participant},
	}, chatRepo, &chatHandlerNotifier{})

	resp := performMultipartHandlerRequest(t, HandleCreateChat(service), authUser, map[string]string{
		"type":              string(chat.ChatTypePrivate),
		"participant_ids[]": participant.ID.String(),
	}, nil)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusInternalServerError)
	}
	if chatRepo.createdFrom != uuid.Nil || chatRepo.createdTo != uuid.Nil {
		t.Fatal("repository lookup failure triggered chat creation")
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
	if notifier.room != chatID.String() || notifier.event != "chat" || !strings.Contains(notifier.msg, `"user_id":"10"`) || strings.Contains(notifier.msg, authUser.ID.String()) {
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

func TestChatMutationHandlersPreserveChatUserMessageParameterOrder(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	chatID := uuid.New()
	messageID := uuid.New()

	tests := []struct {
		name   string
		action string
		handle func(ports.ChatUseCase) fiber.Handler
	}{
		{name: "pin", action: "pin", handle: HandlePinMessage},
		{name: "unpin", action: "unpin", handle: HandleUnpinMessage},
		{name: "delete for all", action: "delete_message_for_all", handle: HandleDeleteMessageForAll},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &chatHandlerRepo{}
			service := newChatHandlerService(&chatHandlerUserRepo{}, repo, &chatHandlerNotifier{})
			resp := performMultipartHandlerRequest(t, tt.handle(service), authUser, map[string]string{
				"chat_id": chatID.String(), "message_id": messageID.String(),
			}, nil)

			if resp.StatusCode != fiber.StatusOK {
				t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
			}
			if repo.mutationAction != tt.action || repo.mutationAuth != authUser || repo.mutationChatID != chatID || repo.mutationUserID != authUser.ID || repo.mutationMsgID != messageID {
				t.Fatalf("mutation delegation = action=%q auth=%p chat=%s user=%s message=%s", repo.mutationAction, repo.mutationAuth, repo.mutationChatID, repo.mutationUserID, repo.mutationMsgID)
			}
		})
	}
}

func TestChatMutationHandlersMapAuthorizationAndTargetErrors(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "outsider", err: chat.ErrNotParticipant, want: fiber.StatusForbidden},
		{name: "ordinary participant", err: chat.ErrPermissionDenied, want: fiber.StatusForbidden},
		{name: "message from another chat", err: chat.ErrMessageNotFound, want: fiber.StatusNotFound},
		{name: "missing chat", err: chat.ErrChatNotFound, want: fiber.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &chatHandlerRepo{mutationErr: tt.err}
			service := newChatHandlerService(&chatHandlerUserRepo{}, repo, &chatHandlerNotifier{})
			resp := performMultipartHandlerRequest(t, HandleDeleteMessageForAll(service), authUser, map[string]string{
				"chat_id": uuid.New().String(), "message_id": uuid.New().String(),
			}, nil)
			if resp.StatusCode != tt.want {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.want)
			}
		})
	}
}

func TestHandleGetChatsUsesBoundedCompositeCursorPage(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	chatID := uuid.New()
	activityAt := time.Now().UTC().Round(0)
	chatRepo := &chatHandlerRepo{chatListPage: ports.ChatListPage{
		Chats:   []chat.Chat{{ID: chatID, CreatedAt: activityAt, LastMessageTimestamp: &activityAt}},
		HasMore: true,
	}}
	service := newChatHandlerService(&chatHandlerUserRepo{}, chatRepo, &chatHandlerNotifier{})

	resp := performMultipartHandlerRequest(t, HandleGetChatsByUserID(service), authUser, map[string]string{
		"limit": "999",
	}, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if chatRepo.chatListQuery.UserID != authUser.ID || chatRepo.chatListQuery.Limit != 100 {
		t.Fatalf("unexpected chat query: %#v", chatRepo.chatListQuery)
	}

	var payload struct {
		Data struct {
			Cursor *string `json:"cursor"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	values, ok := types.DecodePaginationCursor(valueOrEmpty(payload.Data.Cursor))
	if !ok {
		t.Fatalf("expected opaque cursor, got %#v", payload.Data.Cursor)
	}
	gotTime, timeOK := types.CursorCreatedAt(values)
	gotID, idOK := types.CursorUUID(values)
	if !timeOK || !gotTime.Equal(activityAt) || !idOK || gotID != chatID {
		t.Fatalf("unexpected decoded cursor: time=%v timeOK=%v id=%s idOK=%v", gotTime, timeOK, gotID, idOK)
	}
}

func TestHandleGetMessagesUsesDefaultLimitAndPublicIDCursor(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	chatID := uuid.New()
	requestCursor, err := types.NewPublicIDCursor(9001)
	if err != nil {
		t.Fatal(err)
	}
	chatRepo := &chatHandlerRepo{messagePage: ports.ChatMessageListPage{
		Messages: []post.Post{{PublicID: 8001}, {PublicID: 8002}},
		HasMore:  true,
	}}
	service := newChatHandlerService(&chatHandlerUserRepo{}, chatRepo, &chatHandlerNotifier{})

	resp := performMultipartHandlerRequest(t, HandleGetMessagesByChatID(service), authUser, map[string]string{
		"chat_id": chatID.String(),
		"cursor":  *requestCursor,
	}, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if chatRepo.messageQuery.UserID != authUser.ID || chatRepo.messageQuery.ChatID != chatID || chatRepo.messageQuery.Limit != 20 {
		t.Fatalf("unexpected message query: %#v", chatRepo.messageQuery)
	}
	if chatRepo.messageQuery.Cursor == nil || chatRepo.messageQuery.Cursor.PublicID != 9001 {
		t.Fatalf("message cursor did not reach repository: %#v", chatRepo.messageQuery.Cursor)
	}

	var payload struct {
		Data struct {
			Cursor *string `json:"cursor"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	values, ok := types.DecodePaginationCursor(valueOrEmpty(payload.Data.Cursor))
	if !ok {
		t.Fatalf("expected opaque cursor, got %#v", payload.Data.Cursor)
	}
	if publicID, ok := types.CursorPublicID(values); !ok || publicID != 8001 {
		t.Fatalf("unexpected next message cursor: publicID=%d ok=%v", publicID, ok)
	}
}

func TestHandleGetMessagesReturnsEmptyConversationWithoutError(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	service := newChatHandlerService(
		&chatHandlerUserRepo{},
		&chatHandlerRepo{messagePage: ports.ChatMessageListPage{Messages: []post.Post{}}},
		&chatHandlerNotifier{},
	)

	resp := performMultipartHandlerRequest(t, HandleGetMessagesByChatID(service), authUser, map[string]string{
		"chat_id": uuid.New().String(),
	}, nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("empty conversation status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
}

func TestHandleGetMessagesDoesNotLeakRepositoryNotFoundError(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "not participant", err: chat.ErrNotParticipant, wantStatus: fiber.StatusForbidden},
		{name: "unexpected repository error", err: errors.New("record not found"), wantStatus: fiber.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newChatHandlerService(
				&chatHandlerUserRepo{},
				&chatHandlerRepo{messageErr: tt.err},
				&chatHandlerNotifier{},
			)
			resp := performMultipartHandlerRequest(t, HandleGetMessagesByChatID(service), authUser, map[string]string{
				"chat_id": uuid.New().String(),
			}, nil)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Errorf("close response body: %v", err)
				}
			}()
			var payload map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if message, _ := payload["message"].(string); strings.Contains(strings.ToLower(message), "record not found") {
				t.Fatalf("repository error leaked to client: %q", message)
			}
		})
	}
}

func TestChatPaginationRejectsInvalidInput(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	service := newChatHandlerService(&chatHandlerUserRepo{}, &chatHandlerRepo{}, &chatHandlerNotifier{})

	invalidLimit := performMultipartHandlerRequest(t, HandleGetChatsByUserID(service), authUser, map[string]string{"limit": "0"}, nil)
	if invalidLimit.StatusCode != 400 {
		t.Fatalf("invalid limit status = %d, want 400", invalidLimit.StatusCode)
	}
	invalidCursor := performMultipartHandlerRequest(t, HandleGetMessagesByChatID(service), authUser, map[string]string{
		"chat_id": uuid.New().String(), "cursor": "not-a-cursor",
	}, nil)
	if invalidCursor.StatusCode != 400 {
		t.Fatalf("invalid cursor status = %d, want 400", invalidCursor.StatusCode)
	}
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

var _ ports.UserRepository = (*chatHandlerUserRepo)(nil)
var _ ports.ChatRepository = (*chatHandlerRepo)(nil)
