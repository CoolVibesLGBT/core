package services

import (
	"coolvibes/models/chat"
	"coolvibes/models/post"
	"coolvibes/models/user"
	"coolvibes/repositories"
	"errors"
	"mime/multipart"

	"github.com/google/uuid"
)

type ChatService struct {
	mediaRepo        *repositories.MediaRepository
	userRepo         *repositories.UserRepository
	postRepo         *repositories.PostRepository
	matchesRepo      *repositories.MatchesRepository
	chatRepo         *repositories.ChatRepository
	notificationRepo *repositories.NotificationRepository
}

func NewChatService(
	userRepo *repositories.UserRepository,
	postRepo *repositories.PostRepository,
	mediaRepo *repositories.MediaRepository,
	matchesRepo *repositories.MatchesRepository,
	chatRepo *repositories.ChatRepository,
	notificationRepo *repositories.NotificationRepository) *ChatService {
	return &ChatService{postRepo: postRepo, mediaRepo: mediaRepo, userRepo: userRepo, matchesRepo: matchesRepo, chatRepo: chatRepo, notificationRepo: notificationRepo}
}

func (s *ChatService) UserRepo() *repositories.UserRepository {
	return s.userRepo
}

func (s *ChatService) SendTypingEvent(chatID, userID uuid.UUID, typing bool) error {
	return s.chatRepo.SendTypingEvent(chatID, userID, typing)
}

func (s *ChatService) CreateChat(participantUserId, userID uuid.UUID, chatType string) (*chat.Chat, error) {
	participantUser, err := s.userRepo.GetUserByUUIDdWithoutRelations(participantUserId)
	if err != nil {
		return nil, errors.New("user does not exist")
	}

	if participantUser.ID == userID {
		return nil, errors.New("you cannot create a chat with yourself")
	}

	if chatType == string(chat.ChatTypePrivate) {
		chat, err := s.chatRepo.GetPrivateChatBetweenUsers(participantUserId, userID)
		if err != nil {
			// Eğer private chat bulunamazsa yeni oluştur
			chat, err := s.chatRepo.CreatePrivateChat(userID, participantUserId)
			if err != nil {
				return nil, errors.New("failed to create chat")
			}
			return chat, nil
		}
		return chat, nil
	}

	// Diğer chat tipleri için farklı işlemler olabilir (şimdilik hata döndürelim)
	return nil, errors.New("unsupported chat type")
}

func (s *ChatService) GetChatsByUserID(userID uuid.UUID) ([]chat.Chat, error) {
	return s.chatRepo.GetChatsByUserID(userID)
}

func (s *ChatService) AddMessageToChat(request map[string][]string, files []*multipart.FileHeader, author *user.User) (*post.Post, error) {
	_post, err := s.chatRepo.AddMessageToChat(request, files, author)
	if err != nil {
		return nil, err
	}
	return _post, nil
}

func (s *ChatService) GetMessagesByChatID(userID uuid.UUID, chatID uuid.UUID) ([]post.Post, error) {
	return s.chatRepo.GetMessagesByChatID(userID, chatID)
}
