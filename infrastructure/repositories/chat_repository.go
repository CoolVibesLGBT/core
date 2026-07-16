package repositories

import (
	"context"
	"core/application/ports"
	"core/constants"
	"core/helpers"
	"core/models"
	"core/models/chat"
	"core/models/notifications"
	"core/models/utils"
	"encoding/base64"
	"errors"
	"fmt"

	"core/models/post"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-playground/form/v4"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func activeMessageScope(now time.Time) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("(posts.expires_at IS NULL OR posts.expires_at > ?)", now)
	}
}

func hideMessageContent(message *post.Post, viewedOnce bool) {
	message.Content = nil
	message.Title = nil
	message.Summary = nil
	message.Attachments = nil
	message.ContentHidden = true
	message.ViewedOnce = viewedOnce
}

func (r *ChatRepository) consumedViewOnceIDs(userID uuid.UUID, messageIDs []uuid.UUID) (map[uuid.UUID]struct{}, error) {
	consumed := make(map[uuid.UUID]struct{})
	if userID == uuid.Nil || len(messageIDs) == 0 {
		return consumed, nil
	}
	var ids []uuid.UUID
	if err := r.db.Model(&chat.MessageView{}).
		Where("user_id = ? AND message_id IN ?", userID, messageIDs).
		Pluck("message_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		consumed[id] = struct{}{}
	}
	return consumed, nil
}

// sanitizeMessagesForViewer removes every media locator (including nested
// storage_path and variants) simply by omitting attachments until the viewer
// is authorized to receive the opened payload.
func (r *ChatRepository) sanitizeMessagesForViewer(messages []*post.Post, userID uuid.UUID) error {
	disappearingIDs := make([]uuid.UUID, 0)
	for _, message := range messages {
		if message != nil && message.AuthorID != userID && (message.ViewOnce || message.ExpiresInSeconds != nil) {
			disappearingIDs = append(disappearingIDs, message.ID)
		}
	}
	viewedByUser, err := r.consumedViewOnceIDs(userID, disappearingIDs)
	if err != nil {
		return err
	}
	for _, message := range messages {
		if message == nil || message.AuthorID == userID {
			continue
		}
		_, viewed := viewedByUser[message.ID]
		if message.ViewOnce || (message.ExpiresInSeconds != nil && !viewed) {
			hideMessageContent(message, viewed)
		}
	}
	return nil
}

func postPointers(messages []post.Post) []*post.Post {
	result := make([]*post.Post, 0, len(messages))
	seen := make(map[uuid.UUID]struct{})
	var appendMessage func(*post.Post)
	appendMessage = func(message *post.Post) {
		if message == nil {
			return
		}
		if _, ok := seen[message.ID]; ok {
			return
		}
		seen[message.ID] = struct{}{}
		result = append(result, message)
		appendMessage(message.Parent)
	}
	for i := range messages {
		appendMessage(&messages[i])
	}
	return result
}

func validateMessageForm(formData ports.FormData) error {
	if formData.ViewOnce {
		if formData.ImageCount != 1 || formData.VideoCount != 0 || len(formData.Files) != 1 || !isImageUpload(formData.Files[0]) {
			return chat.ErrInvalidViewOnce
		}
	}
	for _, value := range formData.Values["content"] {
		if strings.TrimSpace(value) != "" {
			return nil
		}
	}
	if len(formData.Files) > 0 {
		return nil
	}
	return chat.ErrEmptyMessage
}

func isImageUpload(file ports.UploadedFile) bool {
	if file == nil {
		return false
	}
	return isImageFile(file.ContentType(), file.Filename())
}

func isImageFile(contentType, filename string) bool {
	mimeType := strings.ToLower(strings.TrimSpace(contentType))
	if strings.HasPrefix(mimeType, "image/") {
		return true
	}
	if mimeType != "" && mimeType != "application/octet-stream" {
		return false
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".heic", ".heif", ".avif":
		return true
	default:
		return false
	}
}

func (r *ChatRepository) activeParticipantQuery(ctx context.Context, chatID, userID uuid.UUID) *gorm.DB {
	return r.db.WithContext(ctx).
		Model(&chat.ChatParticipant{}).
		Where("chat_id = ? AND user_id = ? AND left_at IS NULL", chatID, userID)
}

func (r *ChatRepository) ensureActiveParticipant(ctx context.Context, chatID, userID uuid.UUID) error {
	var participant chat.ChatParticipant
	err := r.activeParticipantQuery(ctx, chatID, userID).
		Take(&participant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return chat.ErrNotParticipant
	}
	return err
}

func (r *ChatRepository) CreateChat(chat *chat.Chat) error {
	return r.db.Create(chat).Error
}

func (r *ChatRepository) GetChatByID(id uuid.UUID) (*chat.Chat, error) {
	var chatObj chat.Chat
	now := time.Now()
	err := r.db.Preload("Participants").Preload("Messages", activeMessageScope(now)).Where("id = ?", id).First(&chatObj).Error
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

func (r *ChatRepository) GetChatsByUserIDW(userID uuid.UUID) ([]chat.Chat, error) {
	var chats []chat.Chat
	now := time.Now()

	err := r.db.
		Joins("JOIN chat_participants ON chat_participants.chat_id = chats.id").
		Where("chat_participants.user_id = ?", userID).
		Preload("Participants.User.Avatar.File").
		Preload("Participants.User.Cover.File").
		Preload("LastMessage", activeMessageScope(now)).
		Preload("LastMessage.Author").
		Preload("LastMessage.Attachments").
		Preload("LastMessage.Attachments.File").
		Preload("LastMessage.Author.Avatar.File").
		Preload("LastMessage.Author.Cover.File").
		Order("last_message_timestamp DESC").
		Find(&chats).Error

	if err != nil {
		return nil, err
	}
	lastMessages := make([]*post.Post, 0, len(chats))
	for i := range chats {
		if chats[i].LastMessage != nil {
			lastMessages = append(lastMessages, chats[i].LastMessage)
		}
	}
	if err := r.sanitizeMessagesForViewer(lastMessages, userID); err != nil {
		return nil, err
	}

	return chats, nil
}

func (r *ChatRepository) GetChatsByUserID(userID uuid.UUID) ([]chat.Chat, error) {
	var chats []chat.Chat
	now := time.Now()

	err := r.db.
		Model(&chat.Chat{}).
		Where("chats.deleted_at IS NULL").
		Where(`
			EXISTS (
				SELECT 1
				FROM chat_participants cp
				WHERE cp.chat_id = chats.id
				AND cp.user_id = ?
				AND (
					cp.cleared_at IS NULL
					OR chats.last_message_timestamp IS NULL
					OR chats.last_message_timestamp > cp.cleared_at
				)
			)
		`, userID).
		Where(`
			NOT EXISTS (
				SELECT 1
				FROM engagements e
				JOIN engagement_details ed ON ed.engagement_id = e.id
				WHERE e.contentable_id = chats.id
				AND e.contentable_type = ?
				AND ed.kind = ?
			)
		`,
			post.PostKindChat,
			models.EngagementKindChatDeletedForAll,
		).
		Preload("Participants.User.Avatar.File").
		Preload("Participants.User.Cover.File").
		Preload("LastMessage", func(db *gorm.DB) *gorm.DB {
			return activeMessageScope(now)(db.Where("posts.deleted_at IS NULL"))
		}).
		Preload("LastMessage.Author").
		Preload("LastMessage.Attachments").
		Preload("LastMessage.Attachments.File").
		Preload("LastMessage.Author.Avatar.File").
		Preload("LastMessage.Author.Cover.File").
		Order("chats.last_message_timestamp DESC").
		Find(&chats).Error

	if err != nil {
		return nil, err
	}
	lastMessages := make([]*post.Post, 0, len(chats))
	for i := range chats {
		if chats[i].LastMessage != nil {
			lastMessages = append(lastMessages, chats[i].LastMessage)
		}
	}
	if err := r.sanitizeMessagesForViewer(lastMessages, userID); err != nil {
		return nil, err
	}

	return chats, nil
}

func (r *ChatRepository) GetChatsByUserIDWithCursor(userID uuid.UUID, cursor *time.Time, limit int) ([]chat.Chat, error) {
	var chats []chat.Chat
	now := time.Now()

	db := r.db.
		Joins("JOIN chat_participants ON chat_participants.chat_id = chats.id").
		Where("chat_participants.user_id = ?", userID).
		Preload("Participants.User.Avatar.File").
		Preload("Participants.User.Cover.File").
		Preload("LastMessage", activeMessageScope(now)).
		Preload("LastMessage.Author").
		Preload("LastMessage.Attachments").
		Preload("LastMessage.Attachments.File").
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
	lastMessages := make([]*post.Post, 0, len(chats))
	for i := range chats {
		if chats[i].LastMessage != nil {
			lastMessages = append(lastMessages, chats[i].LastMessage)
		}
	}
	if err := r.sanitizeMessagesForViewer(lastMessages, userID); err != nil {
		return nil, err
	}

	return chats, nil
}

func (r *ChatRepository) GetPrivateChatBetweenUsers(fromUser, toUser uuid.UUID) (*chat.Chat, error) {
	var chatObj chat.Chat
	now := time.Now()

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
		Preload("Messages", activeMessageScope(now)).
		Preload("Messages.Author").
		Preload("Messages.Author.Avatar.File").
		Preload("Messages.Author.Cover.File").
		Preload("Messages.Attachments").
		Preload("Messages.Attachments.File").
		First(&chatObj).Error

	if err != nil {
		return nil, err
	}
	if err := r.sanitizeMessagesForViewer(postPointers(chatObj.Messages), toUser); err != nil {
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

func (r *ChatRepository) AddMessageToChat(context context.Context, formData ports.FormData, author *models.User) (*post.Post, error) {
	if author == nil {
		return nil, chat.ErrNotParticipant
	}
	if err := validateMessageForm(formData); err != nil {
		return nil, err
	}

	type PostForm struct {
		ChatID string `form:"chat_id"`
	}
	decoder := form.NewDecoder()
	postForm := PostForm{}

	if err := decoder.Decode(&postForm, formData.Values); err != nil {
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
	if err := r.ensureActiveParticipant(context, chatObj.ID, author.ID); err != nil {
		return nil, err
	}

	_createdPost, err := r.postRepo.CreateContentablePost(context, formData, author, string(post.PostKindChat), &chatObj.ID)
	if err != nil {
		return nil, err
	}
	chatPost, err := r.postRepo.GetPostByID(_createdPost.ID)
	if err != nil {
		return nil, err
	}
	chatPost.ClientID = formData.ClientID

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
	fmt.Println("HERE", authUser.ID, chatID, userID, messageID)

	post, err := r.postRepo.GetPostByID(messageID)
	if err != nil {
		fmt.Println("ERROR", err.Error())
		return err
	}
	if post != nil {

		err = r.userRepo.engagementRepo.AddEngagement(ctx, authUser.ID, post.AuthorID, models.EngagementKindMessageDeletedForMe, post.ID, models.EngagementContentableTypeMessage)
		if err != nil {
			fmt.Println("ERROR", err.Error())

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
		_, err = r.userRepo.engagementRepo.ToggleEngagement(ctx, authUser.ID, post.AuthorID, models.EngagementKindMessageDeletedForAll, post.ID, models.EngagementContentableTypeMessage)
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
		_, err = r.userRepo.engagementRepo.ToggleEngagement(ctx, authUser.ID, chat.CreatorID, models.EngagementKindChatDeletedForMe, chat.ID, models.EngagementContentableTypeChat)
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
		_, err = r.userRepo.engagementRepo.ToggleEngagement(ctx, authUser.ID, chat.CreatorID, models.EngagementKindChatDeletedForAll, chat.ID, models.EngagementContentableTypeChat)
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

	var participant chat.ChatParticipant
	if err := r.db.
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		First(&participant).Error; err != nil {
		return nil, err
	}

	query := r.messagesByChatIDQuery(userID, chatID, participant.ClearedAt)

	err := query.
		Order("posts.created_at ASC").
		Preload("Author").
		Preload("Parent", activeMessageScope(time.Now())).
		Preload("Parent.Author.Avatar.File").
		Preload("Parent.Author.Cover.File").
		Preload("Parent.Attachments").
		Preload("Engagements", func(db *gorm.DB) *gorm.DB {
			return db.Preload("EngagementDetails", func(db2 *gorm.DB) *gorm.DB {
				return db2.Where("kind NOT IN ?", []models.EngagementKind{
					models.EngagementKindMessageDeletedForMe,
					models.EngagementKindMessageDeletedForAll,
				})
			})
		}).
		Preload("Engagements.EngagementDetails.Engager").
		Preload("Engagements.EngagementDetails.Engagee").
		Preload("Parent.Attachments.File").
		Preload("Author.Avatar.File").
		Preload("Author.Cover.File").
		Preload("Attachments").
		Preload("Attachments.File").
		Find(&messages).Error

	if err != nil {
		return nil, err
	}
	if err := r.sanitizeMessagesForViewer(postPointers(messages), userID); err != nil {
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

func (r *ChatRepository) messagesByChatIDQuery(userID uuid.UUID, chatID uuid.UUID, clearedAt *time.Time) *gorm.DB {
	query := r.db.Model(&post.Post{}).
		Where("posts.contentable_type = ? AND posts.contentable_id = ?", post.PostKindChat, chatID).
		Where("(posts.expires_at IS NULL OR posts.expires_at > ?)", time.Now()).
		Joins(`LEFT JOIN engagements e 
			ON e.contentable_id = posts.id AND e.contentable_type = ?`, post.PostKindMessage).
		Joins(`LEFT JOIN engagement_details ed 
			ON ed.engagement_id = e.id 
			AND ((ed.kind = ? AND ed.engager_id = ?) OR ed.kind = ?)`,
			models.EngagementKindMessageDeletedForMe, userID, models.EngagementKindMessageDeletedForAll).
		Where("ed.id IS NULL")

	if clearedAt != nil {
		query = query.Where("posts.created_at > ?", *clearedAt)
	}

	return query
}

func (r *ChatRepository) GetMessagesByChatIDWithCursor(userID uuid.UUID, chatID uuid.UUID, limit int, cursor *int64) ([]post.Post, error) {
	var messages []post.Post
	if err := r.ensureActiveParticipant(context.Background(), chatID, userID); err != nil {
		return nil, err
	}
	err := r.db.
		Where("contentable_type = ? AND contentable_id = ?", post.PostKindChat, chatID).
		Where("(expires_at IS NULL OR expires_at > ?)", time.Now()).
		Order("created_at ASC").
		Preload("Author").
		Preload("Parent", activeMessageScope(time.Now())).
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
	if err := r.sanitizeMessagesForViewer(postPointers(messages), userID); err != nil {
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

// OpenMessage atomically starts the global expiry window on the first open
// and records authorization per recipient. View-once media is returned as
// base64 exactly once per recipient; its media row and physical file remain.
func (r *ChatRepository) OpenMessage(ctx context.Context, authUser *models.User, chatID, messageID uuid.UUID, now time.Time) (*chat.OpenMessageResult, error) {
	if authUser == nil {
		return nil, chat.ErrNotParticipant
	}

	var result *chat.OpenMessageResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var participant chat.ChatParticipant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("chat_id = ? AND user_id = ? AND left_at IS NULL", chatID, authUser.ID).
			Take(&participant).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return chat.ErrNotParticipant
		} else if err != nil {
			return err
		}

		var message post.Post
		err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND post_kind = ? AND contentable_type = ? AND contentable_id = ?", messageID, post.PostKindMessage, post.PostKindChat, chatID).
			Preload("Author").
			Preload("Attachments").
			Preload("Attachments.File").
			First(&message).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return chat.ErrMessageNotFound
		}
		if err != nil {
			return err
		}
		if message.AuthorID == authUser.ID {
			return chat.ErrAuthorCannotOpen
		}
		if message.ExpiresAt != nil && !message.ExpiresAt.After(now) {
			return chat.ErrMessageExpired
		}
		if !message.ViewOnce && message.ExpiresInSeconds == nil {
			return chat.ErrNotDisappearing
		}

		disappearing := message.ViewOnce || message.ExpiresInSeconds != nil
		alreadyViewed := false
		if disappearing {
			var count int64
			if err := tx.Model(&chat.MessageView{}).
				Where("message_id = ? AND user_id = ?", message.ID, authUser.ID).
				Count(&count).Error; err != nil {
				return err
			}
			alreadyViewed = count > 0
			if message.ViewOnce && alreadyViewed {
				return chat.ErrMessageAlreadySeen
			}
		}

		var openedMedia *chat.OpenedMedia
		if message.ViewOnce {
			if len(message.Attachments) != 1 || message.Attachments[0] == nil || !isImageFile(message.Attachments[0].File.MimeType, message.Attachments[0].File.Name) {
				return chat.ErrInvalidViewOnce
			}
			attachment := message.Attachments[0]
			data, err := os.ReadFile(attachment.File.StoragePath)
			if err != nil {
				return fmt.Errorf("read view-once media: %w", err)
			}
			openedMedia = &chat.OpenedMedia{
				ID: attachment.ID, Name: attachment.File.Name,
				MimeType:   attachment.File.MimeType,
				DataBase64: base64.StdEncoding.EncodeToString(data),
			}
		}

		if disappearing && !alreadyViewed {
			view := chat.MessageView{ID: uuid.New(), MessageID: message.ID, UserID: authUser.ID, ViewedAt: now}
			if err := tx.Create(&view).Error; err != nil {
				return err
			}
		}

		if disappearing && message.OpenedAt == nil {
			updates := map[string]interface{}{"opened_at": now}
			message.OpenedAt = &now
			if message.ExpiresInSeconds != nil {
				expiresAt := now.Add(time.Duration(*message.ExpiresInSeconds) * time.Second)
				updates["expires_at"] = expiresAt
				message.ExpiresAt = &expiresAt
			}
			if err := tx.Model(&post.Post{}).Where("id = ?", message.ID).Updates(updates).Error; err != nil {
				return err
			}
		}

		if message.ViewOnce {
			message.Attachments = nil
			message.ViewedOnce = true
		}
		result = &chat.OpenMessageResult{Message: &message, Media: openedMedia}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *ChatRepository) dueMessagesQuery(tx *gorm.DB, now time.Time, limit int) *gorm.DB {
	return tx.Model(&post.Post{}).
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("post_kind = ?", post.PostKindMessage).
		Where("contentable_type = ? AND contentable_id IS NOT NULL", post.PostKindChat).
		Where("opened_at IS NOT NULL").
		Where("expires_at IS NOT NULL AND expires_at <= ?", now).
		Order("expires_at ASC").
		Limit(limit)
}

func repairChatLastMessage(tx *gorm.DB, chatID uuid.UUID, now time.Time) error {
	var latest post.Post
	err := tx.
		Select("id", "created_at").
		Where("contentable_type = ? AND contentable_id = ?", post.PostKindChat, chatID).
		Where("(expires_at IS NULL OR expires_at > ?)", now).
		Order("created_at DESC, public_id DESC").
		First(&latest).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	updates := map[string]interface{}{
		"last_message_id":        nil,
		"last_message_timestamp": nil,
	}
	if err == nil {
		updates["last_message_id"] = latest.ID
		updates["last_message_timestamp"] = latest.CreatedAt
	}

	return tx.Model(&chat.Chat{}).
		Where("id = ?", chatID).
		Updates(updates).Error
}

func decrementExpiredMessageUnread(tx *gorm.DB, chatID, authorID uuid.UUID, messageCreatedAt time.Time) *gorm.DB {
	return tx.Model(&chat.ChatParticipant{}).
		Where("chat_id = ? AND user_id <> ? AND left_at IS NULL", chatID, authorID).
		Where("(last_read_at IS NULL OR last_read_at < ?)", messageCreatedAt).
		Where("(cleared_at IS NULL OR cleared_at < ?)", messageCreatedAt).
		UpdateColumn("unread_count", gorm.Expr("GREATEST(unread_count - 1, 0)"))
}

// ExpireMessages soft-deletes due chat messages in a lock-safe, idempotent
// batch. Attachments and their physical files are intentionally preserved.
func (r *ChatRepository) ExpireMessages(ctx context.Context, now time.Time, limit int) ([]chat.ExpiredMessage, error) {
	if limit <= 0 {
		limit = 100
	}

	expired := make([]chat.ExpiredMessage, 0, limit)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var due []post.Post
		if err := r.dueMessagesQuery(tx, now, limit).
			Select("id", "contentable_id", "author_id", "expires_at", "created_at").
			Find(&due).Error; err != nil {
			return err
		}

		touchedChats := make(map[uuid.UUID]struct{})
		deleteSession := tx.Session(&gorm.Session{NowFunc: func() time.Time { return now }})

		for _, message := range due {
			if message.ContentableID == nil || message.ExpiresAt == nil {
				continue
			}

			result := deleteSession.
				Where("id = ? AND expires_at IS NOT NULL AND expires_at <= ?", message.ID, now).
				Delete(&post.Post{})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				continue
			}

			chatID := *message.ContentableID
			touchedChats[chatID] = struct{}{}
			expired = append(expired, chat.ExpiredMessage{
				ChatID:    chatID,
				MessageID: message.ID,
				ExpiredAt: *message.ExpiresAt,
			})

			if err := tx.Model(&chat.Chat{}).
				Where("id = ? AND pinned_msg_id = ?", chatID, message.ID).
				Updates(map[string]interface{}{
					"pinned_msg_id": nil,
					"pinned_by_id":  nil,
				}).Error; err != nil {
				return err
			}
			if err := decrementExpiredMessageUnread(tx, chatID, message.AuthorID, message.CreatedAt).Error; err != nil {
				return err
			}
		}

		for chatID := range touchedChats {
			if err := repairChatLastMessage(tx, chatID, now); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return expired, nil
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

func (r *ChatRepository) DeleteChatHistoryForUser(ctx context.Context, authUser *models.User, chatID uuid.UUID) error {

	now := time.Now()

	return r.db.Model(&chat.ChatParticipant{}).
		Where("chat_id = ? AND user_id = ?", chatID, authUser.ID).
		Updates(map[string]interface{}{
			"cleared_at":   now,
			"unread_count": 0,
			"last_read_at": now,
		}).Error
}

func (r *ChatRepository) DeleteChatHistoryForAll(ctx context.Context, authUser *models.User, chatID uuid.UUID) error {

	chatEntity, err := r.GetChatByID(chatID)
	if err != nil {
		return err
	}
	if chatEntity == nil {
		return nil
	}

	if chatEntity.CreatorID != authUser.ID {
		return errors.New("unauthorized")
	}

	now := time.Now()

	tx := r.db.Begin()

	err = tx.Model(&chat.ChatParticipant{}).
		Where("chat_id = ?", chatID).
		Updates(map[string]interface{}{
			"cleared_at":   now,
			"unread_count": 0,
			"last_read_at": now,
		}).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *ChatRepository) MarkChatMessageRead(ctx context.Context, authUser *models.User, chatID uuid.UUID, messages []uuid.UUID) error {

	var participant chat.ChatParticipant
	err := r.db.
		Where("chat_id = ? AND user_id = ?", chatID, authUser.ID).
		First(&participant).Error
	if err != nil {
		return err
	}

	now := time.Now()

	for _, messageID := range messages {

		post, err := r.postRepo.GetPostByIDWithoutRelations(messageID)
		if err != nil {
			return err
		}

		if post == nil {
			continue
		}

		err = r.userRepo.engagementRepo.AddEngagement(
			ctx,
			authUser.ID,
			post.AuthorID,
			models.EngagementKindChatMessageRead,
			post.ID,
			models.EngagementContentableTypeMessage,
		)

		if err != nil {
			return err
		}
	}

	err = r.db.Model(&chat.ChatParticipant{}).
		Where("chat_id = ? AND user_id = ?", chatID, authUser.ID).
		Updates(map[string]interface{}{
			"unread_count": 0,
			"last_read_at": now,
		}).Error

	if err != nil {
		return err
	}

	return nil

}
