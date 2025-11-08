package repositories

import (
	"coolvibes/constants"
	"coolvibes/helpers"
	"coolvibes/models"
	"coolvibes/models/chat"
	"coolvibes/services/socket"
	"encoding/json"
	"fmt"

	"coolvibes/models/post"
	"coolvibes/models/post/shared"
	"log"
	"mime/multipart"
	"time"

	"github.com/go-playground/form/v4"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatRepository struct {
	db            *gorm.DB
	snowFlakeNode *helpers.Node
	postRepo      *PostRepository
	socketService *socket.SocketService
}

func (r *ChatRepository) DB() *gorm.DB {
	return r.db
}

func (r *ChatRepository) Node() *helpers.Node {
	return r.snowFlakeNode
}

func NewChatRepository(db *gorm.DB, snowFlakeNode *helpers.Node, postRepo *PostRepository, socketService *socket.SocketService) *ChatRepository {
	return &ChatRepository{db: db, snowFlakeNode: snowFlakeNode, postRepo: postRepo, socketService: socketService}
}

func (r *ChatRepository) CreateChat(chat *chat.Chat) error {
	return r.db.Create(chat).Error
}

func (r *ChatRepository) GetChatByID(id uuid.UUID) (*chat.Chat, error) {
	var chatObj chat.Chat
	err := r.db.Preload("Participants").Preload("Messages").Where("id = ?", id).First(&chatObj).Error
	if err != nil {
		return nil, err
	}
	return &chatObj, nil
}

func (r *ChatRepository) GetChatByIDWithoutRelations(id uuid.UUID) (*chat.Chat, error) {
	var chatObj chat.Chat
	err := r.db.Where("id = ?", id).First(&chatObj).Error
	if err != nil {
		return nil, err
	}
	return &chatObj, nil
}

func (r *ChatRepository) GetChatsByUserID(userID uuid.UUID) ([]chat.Chat, error) {
	var chats []chat.Chat

	err := r.db.
		Joins("JOIN chat_participants ON chat_participants.chat_id = chats.id").
		Where("chat_participants.user_id = ?", userID).
		Preload("Participants.User").
		Preload("LastMessage").
		Preload("LastMessage.Author").
		Order("last_message_timestamp DESC").
		Find(&chats).Error

	if err != nil {
		return nil, err
	}

	return chats, nil
}

func (r *ChatRepository) GetPrivateChatBetweenUsers(fromUser, toUser uuid.UUID) (*chat.Chat, error) {
	var chatObj chat.Chat

	err := r.db.
		Joins("JOIN chat_participants cp1 ON cp1.chat_id = chats.id").
		Joins("JOIN chat_participants cp2 ON cp2.chat_id = chats.id").
		Where("chats.type = ?", chat.ChatTypePrivate).
		Where("cp1.user_id = ?", fromUser).
		Where("cp2.user_id = ?", toUser).
		Where("chats.deleted_at IS NULL").
		Preload("Participants").
		Preload("Participants.User").
		Preload("Participants.Chat").
		Preload("Messages").
		First(&chatObj).Error

	if err != nil {
		return nil, err
	}
	return &chatObj, nil
}

func (r *ChatRepository) CreatePrivateChat(userID1, userID2 uuid.UUID) (*chat.Chat, error) {
	newChat := &chat.Chat{
		ID:          uuid.New(),
		Type:        chat.ChatTypePrivate,
		CreatorID:   userID1,
		Title:       &shared.LocalizedString{"en": "Private Chat"},
		Description: &shared.LocalizedString{"en": "A private chat is a secure, invite-only conversation between selected participants."},
		Participants: []chat.ChatParticipant{
			{ID: uuid.New(), UserID: userID1},
			{ID: uuid.New(), UserID: userID2},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := r.db.Create(newChat).Error
	if err != nil {
		return nil, err
	}
	return newChat, nil
}

func (r *ChatRepository) AddParticipant(chatID, userID uuid.UUID) error {
	participant := chat.ChatParticipant{
		ChatID: chatID,
		UserID: userID,
	}
	return r.db.FirstOrCreate(&participant, "chat_id = ? AND user_id = ?", chatID, userID).Error
}

func (r *ChatRepository) RemoveParticipant(chatID, userID uuid.UUID) error {
	return r.db.Where("chat_id = ? AND user_id = ?", chatID, userID).Delete(&chat.ChatParticipant{}).Error
}

func (r *ChatRepository) GetParticipants(chatID uuid.UUID) ([]chat.ChatParticipant, error) {
	var participants []chat.ChatParticipant
	err := r.db.Where("chat_id = ?", chatID).Find(&participants).Error
	return participants, err
}

func (r *ChatRepository) CreateGroupChat(
	creatorID uuid.UUID,
	participantIDs []uuid.UUID,
	title *shared.LocalizedString,
	description *shared.LocalizedString,
) (*chat.Chat, error) {
	// Katılımcılar içine creatorID mutlaka eklenmeli, eklenmemişse ekle
	hasCreator := false
	for _, id := range participantIDs {
		if id == creatorID {
			hasCreator = true
			break
		}
	}
	if !hasCreator {
		participantIDs = append(participantIDs, creatorID)
	}

	participants := make([]chat.ChatParticipant, len(participantIDs))
	for i, userID := range participantIDs {
		participants[i] = chat.ChatParticipant{UserID: userID}
	}

	newChat := &chat.Chat{
		ID:           uuid.New(),
		Type:         chat.ChatTypeGroup,
		CreatorID:    creatorID,
		Title:        title,
		Description:  description,
		Participants: participants,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := r.db.Create(newChat).Error
	if err != nil {
		return nil, err
	}
	return newChat, nil
}

func (r *ChatRepository) SendTypingEvent(chatID, userID uuid.UUID, typing bool) error {

	message := map[string]interface{}{
		"action":  constants.CMD_TYPING,
		"chat_id": chatID.String(),
		"user_id": userID.String(),
		"typing":  typing,
	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshalling typing event: %v", err)
		return err
	}

	log.Printf("User %s is typing in chat %s: %v", userID, chatID, typing)

	// BroadcastToNamespace(namespace string, room string, message string)
	err = r.socketService.BroadcastToRoom("/", chatID.String(), "chat", string(jsonMessage))

	if err != nil {
		log.Printf("Error broadcasting typing event: %v", err)
		return nil
	}

	return nil
}

func (r *ChatRepository) AddMessageToChat(request map[string][]string, files []*multipart.FileHeader, author *models.User) (*post.Post, error) {

	type PostForm struct {
		ChatID string `form:"chat_id"`
	}
	decoder := form.NewDecoder()
	postForm := PostForm{}

	if err := decoder.Decode(&postForm, request); err != nil {
		fmt.Println("Form decode error:", err)
		return nil, err
	}

	chatId, err := uuid.Parse(postForm.ChatID) // Burada artık string var
	if err != nil {
		return nil, err
	}

	chatObj, err := r.GetChatByIDWithoutRelations(chatId)
	if err != nil {
		return nil, err
	}

	_createdPost, err := r.postRepo.CreateContentablePost(request, files, author, "chat", &chatObj.ID)
	if err != nil {
		return nil, err
	}
	chatPost, err := r.postRepo.GetPostByID(_createdPost.ID)
	if err != nil {
		return nil, err
	}

	r.db.Model(&chatObj).Updates(map[string]interface{}{
		"last_message_id":        chatPost.ID,
		"last_message_timestamp": chatPost.CreatedAt,
	})

	err = r.db.Model(&chat.ChatParticipant{}).
		Where("chat_id = ? AND user_id <> ?", chatId, author.ID).
		Update("unread_count", gorm.Expr("unread_count + ?", 1)).
		Error
	if err != nil {
		return nil, err
	}

	message := map[string]interface{}{
		"action":  constants.CMD_SEND_MESSAGE,
		"message": chatPost,
	}
	jsonMessage, _ := json.Marshal(message)
	//r.socketService.BroadcastToNamespace("/", chatObj.ID.String(), string(jsonMessage))
	err = r.socketService.BroadcastToRoom("/", chatObj.ID.String(), "chat", string(jsonMessage))
	if err != nil {
		log.Printf("Failed to broadcast message: %v", err)
		return chatPost, err
	}

	return chatPost, err
}

func (r *ChatRepository) GetMessagesByChatID(userID uuid.UUID, chatID uuid.UUID) ([]post.Post, error) {
	var messages []post.Post
	err := r.db.
		Where("contentable_type = ? AND contentable_id = ?", "chat", chatID).
		Order("created_at ASC").
		Preload("Author").
		Preload("Attachments").
		Preload("Attachments.File").
		Find(&messages).Error

	if err != nil {
		return nil, err
	}

	now := time.Now()
	err = r.db.Model(&chat.ChatParticipant{}).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		Updates(map[string]interface{}{
			"unread_count": 0,
			"last_read_at": now,
		}).Error

	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *ChatRepository) GetUserChatIDsByUserPublicID(userPublicId int64) ([]uuid.UUID, error) {
	var chatIDs []uuid.UUID

	err := r.db.
		Table("chat_participants AS cp").
		Select("cp.chat_id").
		Joins("JOIN users u ON u.id = cp.user_id").
		Where("u.public_id = ?", userPublicId).
		Order("cp.id ASC").
		Scan(&chatIDs).Error

	if err != nil {
		log.Println("Hata:", err)
		return nil, err
	}

	return chatIDs, nil
}
