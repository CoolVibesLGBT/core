package repositories

import (
	"context"
	"core/constants"
	"core/helpers"
	"core/models"
	"core/models/chat"
	"core/models/notifications"
	"core/models/utils"
	"fmt"

	"core/models/post"
	"log"
	"mime/multipart"
	"time"

	"github.com/go-playground/form/v4"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatRepository struct {
	db               *gorm.DB
	snowFlakeNode    *helpers.Node
	postRepo         *PostRepository
	userRepo         *UserRepository
	notificationRepo *NotificationRepository
}

func (r *ChatRepository) DB() *gorm.DB {
	return r.db
}

func (r *ChatRepository) Node() *helpers.Node {
	return r.snowFlakeNode
}

func NewChatRepository(db *gorm.DB, snowFlakeNode *helpers.Node, postRepo *PostRepository, userRepo *UserRepository, notificationRepo *NotificationRepository) *ChatRepository {
	return &ChatRepository{db: db, snowFlakeNode: snowFlakeNode, postRepo: postRepo, userRepo: userRepo, notificationRepo: notificationRepo}
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
		Preload("Participants.User.Avatar.File").
		Preload("Participants.User.Cover.File").
		Preload("LastMessage").
		Preload("LastMessage.Author").
		Preload("LastMessage.Author.Avatar.File").
		Preload("LastMessage.Author.Cover.File").
		Order("last_message_timestamp DESC").
		Find(&chats).Error

	if err != nil {
		return nil, err
	}

	return chats, nil
}

func (r *ChatRepository) GetChatsByUserIDWithCursor(userID uuid.UUID, cursor *time.Time, limit int) ([]chat.Chat, error) {
	var chats []chat.Chat

	db := r.db.
		Joins("JOIN chat_participants ON chat_participants.chat_id = chats.id").
		Where("chat_participants.user_id = ?", userID).
		Preload("Participants.User.Avatar.File").
		Preload("Participants.User.Cover.File").
		Preload("LastMessage").
		Preload("LastMessage.Author").
		Preload("LastMessage.Author.Avatar.File").
		Preload("LastMessage.Author.Cover.File").
		Order("last_message_timestamp DESC").
		Limit(limit)

	if cursor != nil {
		// Cursor varsa, last_message_timestamp değeri cursor'dan küçük olanları getir (eski mesajlar)
		db = db.Where("last_message_timestamp < ?", *cursor)
	}

	err := db.Find(&chats).Error
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
		Preload("Participants.User.Avatar.File").
		Preload("Participants.User.Cover.File").
		Preload("Participants.Chat").
		Preload("Messages").
		First(&chatObj).Error

	if err != nil {
		return nil, err
	}
	return &chatObj, nil
}

func (r *ChatRepository) CreatePrivateChat(fromUser, toUser uuid.UUID) (*chat.Chat, error) {
	newChat := &chat.Chat{
		ID:          uuid.New(),
		Type:        chat.ChatTypePrivate,
		CreatorID:   fromUser,
		Title:       &utils.LocalizedString{"en": "Private Chat"},
		Description: &utils.LocalizedString{"en": "A private chat is a secure, invite-only conversation between selected participants."},
		Participants: []chat.ChatParticipant{
			{ID: uuid.New(), UserID: fromUser},
			{ID: uuid.New(), UserID: toUser},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := r.db.Create(newChat).Error
	if err != nil {
		fmt.Println(err)
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

func (r *ChatRepository) CreateGroupChat(creatorID uuid.UUID, participantIDs []uuid.UUID, title *utils.LocalizedString, description *utils.LocalizedString) (*chat.Chat, error) {
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

func (r *ChatRepository) SendTypingEvent(chatID, userID uuid.UUID, typing bool) (map[string]interface{}, error) {
	message := map[string]interface{}{
		"action":  constants.CMD_TYPING,
		"chat_id": chatID.String(),
		"user_id": userID.String(),
		"typing":  typing,
	}
	return message, nil
}

func (r *ChatRepository) AddMessageToChat(context context.Context, request map[string][]string, files []*multipart.FileHeader, author *models.User) (*post.Post, error) {

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

	_createdPost, err := r.postRepo.CreateContentablePost(context, request, files, author, "chat", &chatObj.ID)
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

	newMessageNotification := fmt.Sprintf("You received a new message from %s. Click to read.", author.UserName)
	err = r.NotifyChatParticipants(chatObj.ID, *author, "New Message", newMessageNotification)
	if err != nil {
		fmt.Println("Error notifying chat participants:", err)
		//return nil, err
	}
	return chatPost, err
}

func (r *ChatRepository) PinMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	chat, err := r.GetChatByIDWithoutRelations(chatID)
	if err != nil {
		return err
	}
	message, err := r.postRepo.GetPostByID(messageID)
	if err != nil {
		return err
	}
	chat.PinnedMsgID = &message.ID
	chat.PinnedByID = &userID
	err = r.db.Save(chat).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *ChatRepository) UnpinMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	chat, err := r.GetChatByIDWithoutRelations(chatID)
	if err != nil {
		return err
	}
	chat.PinnedMsgID = nil
	chat.PinnedByID = nil
	err = r.db.Save(chat).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *ChatRepository) DeleteMessageForUser(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	post, err := r.postRepo.GetPostByID(messageID)
	if err != nil {
		return err
	}
	if post != nil {
		_, err = r.userRepo.engagementRepo.ToggleEngagement(ctx, authUser.ID, post.AuthorID, models.EngagementKindDeletedForMe, post.ID, models.EngagementContentableTypeMessage)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *ChatRepository) DeleteMessageForAll(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	post, err := r.postRepo.GetPostByID(messageID)
	if err != nil {
		return err
	}
	if post != nil {
		_, err = r.userRepo.engagementRepo.ToggleEngagement(ctx, authUser.ID, post.AuthorID, models.EngagementKindDeletedForAll, post.ID, models.EngagementContentableTypeMessage)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *ChatRepository) DeleteChatForUser(ctx context.Context, authUser *models.User, chatID, userID uuid.UUID) error {
	chat, err := r.GetChatByID(chatID)
	if err != nil {
		return err
	}
	if chat != nil {
		_, err = r.userRepo.engagementRepo.ToggleEngagement(ctx, authUser.ID, chat.CreatorID, models.EngagementKindDeletedForMe, chat.ID, models.EngagementContentableTypeChat)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *ChatRepository) DeleteChatForAll(ctx context.Context, authUser *models.User, chatID uuid.UUID) error {
	chat, err := r.GetChatByID(chatID)
	if err != nil {
		return err
	}
	if chat != nil {
		_, err = r.userRepo.engagementRepo.ToggleEngagement(ctx, authUser.ID, chat.CreatorID, models.EngagementKindDeletedForAll, chat.ID, models.EngagementContentableTypeChat)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *ChatRepository) DeleteChat(ctx context.Context, authUser *models.User, chatID, userID uuid.UUID) error {
	return r.db.Delete(&chat.Chat{}, "id = ? AND user_id = ?", chatID).Error
}

func (r *ChatRepository) DeleteMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	return r.db.Delete(&post.Post{}, "id = ? AND contentable_type = ? AND contentable_id = ?", messageID, post.PostKindChat, chatID).Error
}

func (r *ChatRepository) NotifyChatParticipants(chatId uuid.UUID, author models.User, messageTitle, messageText string) error {

	// Katılımcıları ve user ilişkisini preload ile çek
	var participants []chat.ChatParticipant
	err := r.db.Preload("User").
		Where("chat_id = ? AND user_id <> ?", chatId, author.ID).
		Find(&participants).Error
	if err != nil {
		return err
	}

	// Her katılımcıya bildirim gönder
	for _, participant := range participants {
		user := participant.User
		if user.ID == uuid.Nil {
			fmt.Printf("User data missing for participant %s\n", participant.ID)
			continue
		}

		payload := notifications.NotificationPayload{
			Title: messageTitle,
			Body:  messageText,
			// diğer alanlar eklenecekse ekle
		}

		err := r.notificationRepo.SendNotificationToUser(author, user, notifications.NotificationTypeChatMessage, messageTitle, messageText, payload)
		if err != nil {
			fmt.Printf("Bildirim gönderilemedi user %s: %v\n", user.ID, err)
		}
	}

	return nil
}

func (r *ChatRepository) GetMessagesByChatID(userID uuid.UUID, chatID uuid.UUID) ([]post.Post, error) {
	var messages []post.Post
	err := r.db.
		Where("contentable_type = ? AND contentable_id = ?", post.PostKindChat, chatID).
		Order("created_at ASC").
		Preload("Author").
		Preload("Parent").
		Preload("Parent.Author.Avatar.File").
		Preload("Parent.Author.Cover.File").
		Preload("Parent.Attachments").
		Preload("Parent.Attachments.File").
		Preload("Author.Avatar.File").
		Preload("Author.Cover.File").
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

func (r *ChatRepository) GetMessagesByChatIDWithCursor(userID uuid.UUID, chatID uuid.UUID, limit int, cursor *int64) ([]post.Post, error) {
	var messages []post.Post
	err := r.db.
		Where("contentable_type = ? AND contentable_id = ?", post.PostKindChat, chatID).
		Order("created_at ASC").
		Preload("Author").
		Preload("Parent").
		Preload("Location").
		Preload("Parent.Author.Avatar.File").
		Preload("Parent.Author.Cover.File").
		Preload("Parent.Attachments").
		Preload("Parent.Attachments.File").
		Preload("Author.Avatar.File").
		Preload("Author.Cover.File").
		Preload("Attachments").
		Preload("Attachments.File").
		Limit(limit).
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
