package services

import (
	"context"
	"core/constants"
	"core/models"
	"core/models/chat"
	"core/models/post"
	"core/repositories"
	"core/services/socket"
	"core/types"
	"encoding/json"
	"errors"
	"log"
	"mime/multipart"

	"github.com/google/uuid"
)

type ChatService struct {
	socketService    *socket.SocketService
	mediaRepo        *repositories.MediaRepository
	userRepo         *repositories.UserRepository
	postRepo         *repositories.PostRepository
	matchesRepo      *repositories.MatchesRepository
	chatRepo         *repositories.ChatRepository
	notificationRepo *repositories.NotificationRepository
}

func NewChatService(
	socketService *socket.SocketService,
	userRepo *repositories.UserRepository,
	postRepo *repositories.PostRepository,
	mediaRepo *repositories.MediaRepository,
	matchesRepo *repositories.MatchesRepository,
	chatRepo *repositories.ChatRepository,
	notificationRepo *repositories.NotificationRepository) *ChatService {
	return &ChatService{
		socketService: socketService, postRepo: postRepo, mediaRepo: mediaRepo, userRepo: userRepo, matchesRepo: matchesRepo, chatRepo: chatRepo, notificationRepo: notificationRepo}
}

func (s *ChatService) UserRepo() *repositories.UserRepository {
	return s.userRepo
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
		if participantUser.ID.String() == userID.String() {
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

func (s *ChatService) AddMessageToChat(context context.Context, request map[string][]string, files []*multipart.FileHeader, author *models.User) (*post.Post, error) {
	_post, err := s.chatRepo.AddMessageToChat(context, request, files, author)
	if err != nil {
		return nil, err
	}

	message := map[string]interface{}{
		"action":  constants.CMD_SEND_MESSAGE,
		"message": _post,
	}
	jsonMessage, _ := json.Marshal(message)
	err = s.socketService.BroadcastToRoom("/", _post.ContentableID.String(), "chat", string(jsonMessage))
	if err != nil {
		log.Printf("Failed to broadcast message: %v", err)
		return _post, err
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
