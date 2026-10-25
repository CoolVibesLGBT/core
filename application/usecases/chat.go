package usecases

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/constants"
	domainmedia "core/domain/media"
	domainuser "core/domain/user"
	"core/models"
	"core/models/chat"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ChatService struct {
	socketService    ports.RealtimeNotifier
	mediaRepo        ports.MediaRepository
	userRepo         ports.UserRepository
	postRepo         ports.PostRepository
	matchesRepo      ports.MatchesRepository
	chatRepo         ports.ChatRepository
	notificationRepo ports.NotificationRepository
}

var _ ports.ChatUseCase = (*ChatService)(nil)

func NewChatService(
	socketService ports.RealtimeNotifier,
	userRepo ports.UserRepository,
	postRepo ports.PostRepository,
	mediaRepo ports.MediaRepository,
	matchesRepo ports.MatchesRepository,
	chatRepo ports.ChatRepository,
	notificationRepo ports.NotificationRepository) *ChatService {
	return &ChatService{
		socketService: socketService, postRepo: postRepo, mediaRepo: mediaRepo, userRepo: userRepo, matchesRepo: matchesRepo, chatRepo: chatRepo, notificationRepo: notificationRepo}
}

func (s *ChatService) SendTypingEvent(chatID uuid.UUID, user *models.User, typing bool) error {
	if user == nil || user.ID == uuid.Nil {
		return chat.ErrNotParticipant
	}
	if err := s.chatRepo.SendTypingEvent(chatID, user.ID, typing); err != nil {
		return err
	}
	message := types.ChatTypingEvent{
		Action: constants.CMD_TYPING,
		ChatID: chatID,
		UserID: types.SnowflakeID(user.PublicID),
		Typing: typing,
	}
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshalling typing event: %v", err)
		return err
	}
	log.Printf("User %s is typing in chat %s: %v", user.ID, chatID, typing)
	err = s.socketService.BroadcastToRoom("/", chatID.String(), "chat", string(jsonMessage))

	if err != nil {
		log.Printf("Error broadcasting typing event: %v", err)
		return err
	}
	return nil
}

func (s *ChatService) CreateChat(context context.Context, participantUserId, userID uuid.UUID, chatType string) (*chat.Chat, error) {
	participantUser, err := s.userRepo.GetUserByUUIDdWithoutRelations(types.Filter{Context: context, UserUUID: participantUserId})
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, err
	}
	return s.createChatForResolvedParticipant(participantUser, userID, chatType)
}

// CreateChatFromIdentifier accepts the public snowflake ID used by API
// projections while retaining UUID compatibility for already-deployed
// clients. Resolution stays inside the application boundary; handlers never
// treat an arbitrary client value as an internal persistence identifier.
func (s *ChatService) CreateChatFromIdentifier(ctx context.Context, participantIdentifier string, actor *models.User, chatType string) (*types.Chat, error) {
	if actor == nil || actor.ID == uuid.Nil {
		return nil, chat.ErrNotParticipant
	}
	participantUser, err := s.resolveChatParticipant(ctx, participantIdentifier)
	if err != nil {
		return nil, err
	}
	entity, err := s.createChatForResolvedParticipant(participantUser, actor.ID, chatType)
	if err != nil {
		return nil, err
	}
	return chatProjection(entity, actor.ID, actor, participantUser), nil
}

func (s *ChatService) resolveChatParticipant(ctx context.Context, participantIdentifier string) (*models.User, error) {
	participantIdentifier = strings.TrimSpace(participantIdentifier)
	if participantIdentifier == "" {
		return nil, constants.ErrInvalidParticipantID
	}

	var (
		participantUser *models.User
		err             error
	)
	if participantUUID, parseErr := uuid.Parse(participantIdentifier); parseErr == nil {
		participantUser, err = s.userRepo.GetUserByUUIDdWithoutRelations(types.Filter{Context: ctx, UserUUID: participantUUID})
	} else {
		publicID, parseErr := strconv.ParseInt(participantIdentifier, 10, 64)
		if parseErr != nil || publicID <= 0 {
			return nil, constants.ErrInvalidParticipantID
		}
		participantUser, err = s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: publicID})
	}
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, err
	}
	if participantUser == nil || participantUser.ID == uuid.Nil {
		return nil, constants.ErrUserNotFound
	}
	return participantUser, nil
}

func (s *ChatService) createChatForResolvedParticipant(participantUser *models.User, userID uuid.UUID, chatType string) (*chat.Chat, error) {
	if participantUser == nil || participantUser.ID == uuid.Nil {
		return nil, constants.ErrUserNotFound
	}

	if err := domainuser.EnsureDifferentUsers(participantUser.ID.String(), userID.String(), domainuser.InteractionChat); err != nil {
		return nil, constants.ErrSelfChatNotAllowed
	}

	if chatType == string(chat.ChatTypePrivate) {
		existingChat, err := s.chatRepo.GetPrivateChatBetweenUsers(participantUser.ID, userID)
		if err != nil {
			if !errors.Is(err, chat.ErrChatNotFound) {
				return nil, err
			}
			createdChat, err := s.chatRepo.CreatePrivateChat(userID, participantUser.ID)
			if err != nil {
				return nil, err
			}
			return createdChat, nil
		}
		return existingChat, nil
	}

	// Diğer chat tipleri için farklı işlemler olabilir (şimdilik hata döndürelim)
	return nil, constants.ErrUnsupportedChatType
}

func (s *ChatService) GetChatsByUserID(userID uuid.UUID) ([]types.Chat, error) {
	chats, _, err := s.FetchChats(context.Background(), userID, nil, constants.DEFAULT_LIMIT)
	return chats, err
}

func (s *ChatService) FetchChats(ctx context.Context, userID uuid.UUID, cursor *ports.ChatListCursor, limit int) ([]types.Chat, *string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	limit = boundedChatPageLimit(limit)
	page, err := s.chatRepo.ListChats(ctx, ports.ChatListQuery{
		UserID: userID,
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		return nil, nil, err
	}

	var nextCursor *string
	if page.HasMore && len(page.Chats) > 0 {
		last := page.Chats[len(page.Chats)-1]
		activityAt := last.CreatedAt
		if last.LastMessageTimestamp != nil {
			activityAt = *last.LastMessageTimestamp
		}
		nextCursor, err = types.NewTimeUUIDCursor(activityAt, last.ID)
		if err != nil {
			return nil, nil, err
		}
	}

	return chatListProjection(page.Chats, userID), nextCursor, nil
}

func (s *ChatService) AddMessageToChat(context context.Context, values map[string][]string, files []*domainmedia.UploadedFile, author *models.User) (*types.ChatMessage, error) {
	_post, err := s.chatRepo.AddMessageToChat(context, values, files, author)
	if err != nil {
		return nil, err
	}
	if _post.ScheduledAt != nil {
		// Only the author knows about it until PublishScheduledMessages.
		return chatMessageProjection(_post, map[uuid.UUID]*models.User{author.ID: author}, 1), nil
	}

	// Room broadcasts cannot be personalized. Unopened disappearing content is
	// therefore always a placeholder; the authenticated open action is the
	// only path that returns it to a recipient.
	broadcastPost := *_post
	if broadcastPost.ViewOnce || broadcastPost.ExpiresInSeconds != nil {
		broadcastPost.Content = nil
		broadcastPost.Title = nil
		broadcastPost.Summary = nil
		broadcastPost.Attachments = nil
		broadcastPost.ContentHidden = true
	}
	broadcastView := chatMessageProjection(&broadcastPost, map[uuid.UUID]*models.User{author.ID: author}, 1)
	message := types.ChatMessageEvent{
		Action:  constants.CMD_SEND_MESSAGE,
		Message: broadcastView,
	}
	jsonMessage, _ := json.Marshal(message)
	if _post.ContentableID != nil {
		err = s.socketService.BroadcastToRoom("/", _post.ContentableID.String(), "chat", string(jsonMessage))
	}
	if err != nil {
		log.Printf("Failed to broadcast message: %v", err)
		// Persistence already succeeded. Treat realtime delivery as a
		// best-effort side effect so the client does not retry and duplicate the
		// committed message.
	}
	if _post.ViewOnce {
		// A view-once send response must not publish a reusable static URL/path,
		// even to the sender. The caption remains available for the placeholder.
		senderPost := *_post
		senderPost.Attachments = nil
		senderPost.ContentHidden = true
		return chatMessageProjection(&senderPost, map[uuid.UUID]*models.User{author.ID: author}, 1), nil
	}
	return chatMessageProjection(_post, map[uuid.UUID]*models.User{author.ID: author}, 1), nil
}

func (s *ChatService) GetMessagesByChatID(userID uuid.UUID, chatID uuid.UUID) ([]types.ChatMessage, error) {
	messages, _, err := s.FetchChatMessages(context.Background(), userID, chatID, nil, constants.DEFAULT_LIMIT)
	return messages, err
}

func (s *ChatService) FetchChatMessages(ctx context.Context, userID, chatID uuid.UUID, cursor *ports.ChatMessageListCursor, limit int) ([]types.ChatMessage, *string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	limit = boundedChatPageLimit(limit)
	page, err := s.chatRepo.ListChatMessages(ctx, ports.ChatMessageListQuery{
		UserID: userID,
		ChatID: chatID,
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		return nil, nil, err
	}

	var nextCursor *string
	if page.HasMore && len(page.Messages) > 0 {
		// Repository pages preserve the historical chronological response
		// order, so the first item is the oldest item in this page.
		nextCursor, err = types.NewPublicIDCursor(page.Messages[0].PublicID)
		if err != nil {
			return nil, nil, err
		}
	}

	return chatMessagesProjection(page.Messages), nextCursor, nil
}

func boundedChatPageLimit(limit int) int {
	if limit <= 0 {
		return constants.DEFAULT_LIMIT
	}
	if limit > constants.MAXIMUM_LIMIT {
		return constants.MAXIMUM_LIMIT
	}
	return limit
}

func (s *ChatService) PinMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	if err := s.chatRepo.PinMessage(ctx, authUser, chatID, userID, messageID); err != nil {
		return err
	}
	s.broadcastPinChange(constants.CMD_PIN_MESSAGE, chatID, messageID)
	return nil
}

func (s *ChatService) UnpinMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	if err := s.chatRepo.UnpinMessage(ctx, authUser, chatID, userID, messageID); err != nil {
		return err
	}
	s.broadcastPinChange(constants.CMD_UNPIN_MESSAGE, chatID, messageID)
	return nil
}

// broadcastPinChange lets every open client refresh its pinned list.
func (s *ChatService) broadcastPinChange(action string, chatID, messageID uuid.UUID) {
	if s.socketService == nil {
		return
	}
	payload, err := json.Marshal(map[string]string{
		"action":     action,
		"chat_id":    chatID.String(),
		"message_id": messageID.String(),
	})
	if err == nil {
		_ = s.socketService.BroadcastToRoom("/", chatID.String(), "chat", string(payload))
	}
}

// FetchPinnedMessages returns the viewer-visible pinned messages, oldest pin
// first, for the Telegram pinned-message bar.
func (s *ChatService) FetchPinnedMessages(ctx context.Context, userID, chatID uuid.UUID) ([]types.ChatMessage, error) {
	repository, ok := s.chatRepo.(ports.ChatPinnedMessagesRepository)
	if !ok {
		return []types.ChatMessage{}, nil
	}
	messages, err := repository.ListPinnedMessages(ctx, userID, chatID)
	if err != nil {
		return nil, err
	}
	return chatMessagesProjection(messages), nil
}

func (s *ChatService) DeleteMessageForUser(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	return s.chatRepo.DeleteMessageForUser(ctx, authUser, chatID, userID, messageID)
}

func (s *ChatService) DeleteMessageForAll(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	return s.chatRepo.DeleteMessageForAll(ctx, authUser, chatID, userID, messageID)
}

func (s *ChatService) DeleteChatForUser(ctx context.Context, authUser *models.User, chatID, userID uuid.UUID) error {
	return s.chatRepo.DeleteChatForUser(ctx, authUser, chatID, userID)
}

func (s *ChatService) DeleteChatForAll(ctx context.Context, authUser *models.User, chatID uuid.UUID) error {
	return s.chatRepo.DeleteChatForAll(ctx, authUser, chatID)
}

func (s *ChatService) DeleteChat(ctx context.Context, authUser *models.User, chatID, userID uuid.UUID) error {
	return s.chatRepo.DeleteChat(ctx, authUser, chatID, userID)
}

func (s *ChatService) DeleteMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	return s.chatRepo.DeleteMessage(ctx, authUser, chatID, userID, messageID)
}

func (s *ChatService) DeleteChatHistoryForUser(ctx context.Context, authUser *models.User, chatID uuid.UUID) error {
	return s.chatRepo.DeleteChatHistoryForUser(ctx, authUser, chatID)
}

func (s *ChatService) DeleteChatHistoryForAll(ctx context.Context, authUser *models.User, chatID uuid.UUID) error {
	return s.chatRepo.DeleteChatHistoryForAll(ctx, authUser, chatID)
}

func (s *ChatService) MarkChatMessageRead(ctx context.Context, authUser *models.User, chatID uuid.UUID, messageIDs []uuid.UUID) error {
	return s.chatRepo.MarkChatMessageRead(ctx, authUser, chatID, messageIDs)
}

func (s *ChatService) OpenMessage(ctx context.Context, authUser *models.User, chatID, messageID uuid.UUID, now time.Time) (*types.ChatOpenMessageResult, error) {
	if authUser == nil || authUser.ID == uuid.Nil {
		return nil, chat.ErrNotParticipant
	}
	result, err := s.chatRepo.OpenMessage(ctx, authUser, chatID, messageID, now)
	if err != nil {
		return nil, err
	}
	if result == nil || result.Message == nil {
		return nil, constants.ErrChatOpenMessageResultEmpty
	}

	safeResult := &types.ChatOpenMessageResult{
		Message: chatMessageProjection(result.Message, map[uuid.UUID]*models.User{authUser.ID: authUser}, 1),
	}
	if result.Media != nil {
		safeResult.Media = &types.ChatOpenedMedia{
			Name:       result.Media.Name,
			MimeType:   result.Media.MimeType,
			DataBase64: result.Media.DataBase64,
		}
	}

	if result.Message.ViewOnce && result.Message.AuthorID == authUser.ID {
		return safeResult, nil
	}

	event := types.ChatMessageOpenedEvent{
		Action:    constants.CMD_CHAT_MESSAGE_OPENED,
		ChatID:    chatID,
		MessageID: messageID,
		OpenedBy:  types.SnowflakeID(authUser.PublicID),
		OpenedAt:  result.Message.OpenedAt,
		ExpiresAt: result.Message.ExpiresAt,
		ViewOnce:  result.Message.ViewOnce,
	}
	jsonMessage, marshalErr := json.Marshal(event)
	if marshalErr != nil {
		log.Printf("Failed to marshal opened message event: %v", marshalErr)
		return safeResult, nil
	}
	if broadcastErr := s.socketService.BroadcastToRoom("/", chatID.String(), "chat", string(jsonMessage)); broadcastErr != nil {
		// Consumption already committed. A transient socket failure must never
		// turn a successful first view into a failed HTTP response.
		log.Printf("Failed to broadcast opened message %s: %v", messageID, broadcastErr)
	}
	return safeResult, nil
}

func (s *ChatService) ExpireMessages(ctx context.Context, now time.Time, limit int) (int, error) {
	// The one-second chat sweep also delivers due scheduled messages.
	if err := s.PublishScheduledMessages(ctx, now, limit); err != nil {
		log.Printf("Failed to publish scheduled messages: %v", err)
	}
	expired, err := s.chatRepo.ExpireMessages(ctx, now, limit)
	if err != nil {
		return 0, err
	}

	for _, item := range expired {
		message := types.ChatMessageExpiredEvent{
			Action:    constants.CMD_CHAT_MESSAGE_EXPIRED,
			ChatID:    item.ChatID,
			MessageID: item.MessageID,
			ExpiredAt: item.ExpiredAt,
		}
		jsonMessage, marshalErr := json.Marshal(message)
		if marshalErr != nil {
			log.Printf("Failed to marshal expired message event: %v", marshalErr)
			continue
		}
		if broadcastErr := s.socketService.BroadcastToRoom("/", item.ChatID.String(), "chat", string(jsonMessage)); broadcastErr != nil {
			// expires_at is also returned with each message, so clients can still
			// hide it on time if a transient socket broadcast fails.
			log.Printf("Failed to broadcast expired message %s: %v", item.MessageID, broadcastErr)
		}
	}

	return len(expired), nil
}

func (s *ChatService) FetchScheduledMessages(ctx context.Context, userID, chatID uuid.UUID) ([]types.ChatMessage, error) {
	repository, ok := s.chatRepo.(ports.ChatScheduledMessagesRepository)
	if !ok {
		return []types.ChatMessage{}, nil
	}
	messages, err := repository.ListScheduledMessages(ctx, userID, chatID)
	if err != nil {
		return nil, err
	}
	return chatMessagesProjection(messages), nil
}

func (s *ChatService) SendScheduledNow(ctx context.Context, author *models.User, chatID, messageID uuid.UUID) error {
	repository, ok := s.chatRepo.(ports.ChatScheduledMessagesRepository)
	if !ok {
		return chat.ErrPermissionDenied
	}
	return repository.SendScheduledNow(ctx, author, chatID, messageID)
}

// PublishScheduledMessages delivers due scheduled messages exactly like a
// fresh send: the repository notifies participants, this broadcasts to the room.
func (s *ChatService) PublishScheduledMessages(ctx context.Context, now time.Time, limit int) error {
	repository, ok := s.chatRepo.(ports.ChatScheduledMessagesRepository)
	if !ok {
		return nil
	}
	published, err := repository.PublishScheduledMessages(ctx, now, limit)
	if err != nil {
		return err
	}
	for i := range published {
		message := &published[i]
		if message.ContentableID == nil {
			continue
		}
		author := message.Author
		broadcast := *message
		if broadcast.ViewOnce || broadcast.ExpiresInSeconds != nil {
			broadcast.Content = nil
			broadcast.Title = nil
			broadcast.Summary = nil
			broadcast.Attachments = nil
			broadcast.ContentHidden = true
		}
		view := chatMessageProjection(&broadcast, map[uuid.UUID]*models.User{author.ID: &author}, 1)
		payload, _ := json.Marshal(types.ChatMessageEvent{Action: constants.CMD_SEND_MESSAGE, Message: view})
		if s.socketService != nil {
			if err := s.socketService.BroadcastToRoom("/", message.ContentableID.String(), "chat", string(payload)); err != nil {
				log.Printf("Failed to broadcast scheduled message %s: %v", message.ID, err)
			}
		}
	}
	return nil
}
