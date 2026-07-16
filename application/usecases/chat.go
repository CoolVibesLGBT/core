package usecases

import (
	"context"
	"core/application/ports"
	"core/constants"
	domainuser "core/domain/user"
	"core/models"
	"core/models/chat"
	"core/models/post"
	"core/types"
	"encoding/json"
	"errors"
	"log"
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

func (s *ChatService) SendTypingEvent(chatID, userID uuid.UUID, typing bool) error {
	message, _ := s.chatRepo.SendTypingEvent(chatID, userID, typing)
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshalling typing event: %v", err)
		return err
	}
	log.Printf("User %s is typing in chat %s: %v", userID, chatID, typing)
	err = s.socketService.BroadcastToRoom("/", chatID.String(), "chat", string(jsonMessage))

	if err != nil {
		log.Printf("Error broadcasting typing event: %v", err)
		return nil
	}
	return err
}

func (s *ChatService) CreateChat(context context.Context, participantUserId, userID uuid.UUID, chatType string) (*chat.Chat, error) {
	participantUser, err := s.userRepo.GetUserByUUIDdWithoutRelations(types.Filter{Context: context, UserUUID: participantUserId})
	if err != nil {
		return nil, errors.New(constants.ErrUserNotFound.String())
	}

	if participantUser != nil {
		if err := domainuser.EnsureDifferentUsers(participantUser.ID.String(), userID.String(), domainuser.InteractionChat); err != nil {
			return nil, errors.New(constants.ErrSelfChatNotAllowed.String())
		}
	}

	if chatType == string(chat.ChatTypePrivate) {
		chat, err := s.chatRepo.GetPrivateChatBetweenUsers(participantUserId, userID)
		if err != nil {
			// Eğer private chat bulunamazsa yeni oluştur
			chat, err := s.chatRepo.CreatePrivateChat(userID, participantUserId)
			if err != nil {
				return nil, err
			}
			return chat, nil
		}
		return chat, nil
	}

	// Diğer chat tipleri için farklı işlemler olabilir (şimdilik hata döndürelim)
	return nil, errors.New(constants.ErrUnsupportedChatType.String())
}

func (s *ChatService) GetChatsByUserID(userID uuid.UUID) ([]chat.Chat, error) {
	return s.chatRepo.GetChatsByUserID(userID)
}

func (s *ChatService) AddMessageToChat(context context.Context, form ports.FormData, author *models.User) (*post.Post, error) {
	_post, err := s.chatRepo.AddMessageToChat(context, form, author)
	if err != nil {
		return nil, err
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
	message := map[string]interface{}{
		"action":  constants.CMD_SEND_MESSAGE,
		"message": &broadcastPost,
	}
	jsonMessage, _ := json.Marshal(message)
	err = s.socketService.BroadcastToRoom("/", _post.ContentableID.String(), "chat", string(jsonMessage))
	if err != nil {
		log.Printf("Failed to broadcast message: %v", err)
		return _post, err
	}
	if _post.ViewOnce {
		// A view-once send response must not publish a reusable static URL/path,
		// even to the sender. The caption remains available for the placeholder.
		senderPost := *_post
		senderPost.Attachments = nil
		senderPost.ContentHidden = true
		return &senderPost, nil
	}
	return _post, nil
}

func (s *ChatService) GetMessagesByChatID(userID uuid.UUID, chatID uuid.UUID) ([]post.Post, error) {
	return s.chatRepo.GetMessagesByChatID(userID, chatID)
}

func (s *ChatService) PinMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	return s.chatRepo.PinMessage(ctx, authUser, chatID, userID, messageID)
}

func (s *ChatService) UnpinMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	return s.chatRepo.UnpinMessage(ctx, authUser, chatID, userID, messageID)
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

func (s *ChatService) OpenMessage(ctx context.Context, authUser *models.User, chatID, messageID uuid.UUID, now time.Time) (*chat.OpenMessageResult, error) {
	result, err := s.chatRepo.OpenMessage(ctx, authUser, chatID, messageID, now)
	if err != nil {
		return nil, err
	}

	event := map[string]interface{}{
		"action":     constants.CMD_CHAT_MESSAGE_OPENED,
		"chat_id":    chatID,
		"message_id": messageID,
		"opened_by":  authUser.ID,
		"opened_at":  result.Message.OpenedAt,
		"expires_at": result.Message.ExpiresAt,
		"view_once":  result.Message.ViewOnce,
	}
	jsonMessage, marshalErr := json.Marshal(event)
	if marshalErr != nil {
		log.Printf("Failed to marshal opened message event: %v", marshalErr)
		return result, nil
	}
	if broadcastErr := s.socketService.BroadcastToRoom("/", chatID.String(), "chat", string(jsonMessage)); broadcastErr != nil {
		// Consumption already committed. A transient socket failure must never
		// turn a successful first view into a failed HTTP response.
		log.Printf("Failed to broadcast opened message %s: %v", messageID, broadcastErr)
	}
	return result, nil
}

func (s *ChatService) ExpireMessages(ctx context.Context, now time.Time, limit int) (int, error) {
	expired, err := s.chatRepo.ExpireMessages(ctx, now, limit)
	if err != nil {
		return 0, err
	}

	for _, item := range expired {
		message := map[string]interface{}{
			"action":     constants.CMD_CHAT_MESSAGE_EXPIRED,
			"chat_id":    item.ChatID,
			"message_id": item.MessageID,
			"expired_at": item.ExpiredAt,
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
