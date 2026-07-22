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

func (r *ChatRepository) GetChatByIDWithoutRelations(chatID uuid.UUID) (*chat.Chat, error) {
	var chatObj chat.Chat
	err := r.db.Where("id = ?", chatID).First(&chatObj).Error
	if err != nil {
		return nil, err
	}
	return &chatObj, nil
}

func (r *ChatRepository) GetChatsByUserIDW(userID uuid.UUID) ([]chat.Chat, error) {
	return r.GetChatsByUserID(userID)
}

func (r *ChatRepository) GetChatsByUserID(userID uuid.UUID) ([]chat.Chat, error) {
	page, err := r.ListChats(context.Background(), ports.ChatListQuery{
		UserID: userID,
		Limit:  constants.DEFAULT_LIMIT,
	})
	return page.Chats, err
}

const chatListActivityExpression = "COALESCE(chats.last_message_timestamp, chats.created_at)"

func chatUserReadScope(db *gorm.DB) *gorm.DB {
	return db.Select("id", "public_id", "user_name", "display_name", "avatar_id")
}

func chatMediaReadScope(db *gorm.DB) *gorm.DB {
	return db.Select("id", "public_id", "file_id")
}

func chatAttachmentMediaReadScope(db *gorm.DB) *gorm.DB {
	// owner columns are required by GORM to attach polymorphic media rows to
	// their posts; the application DTO still omits them from every response.
	return db.Select("id", "public_id", "file_id", "owner_id", "owner_type")
}

func chatAvatarFileReadScope(db *gorm.DB) *gorm.DB {
	return db.Select("id", "url", "variants")
}

func chatAttachmentFileReadScope(db *gorm.DB) *gorm.DB {
	return db.Select("id", "url", "mime_type", "name", "variants")
}

func (r *ChatRepository) chatsByUserIDQuery(ctx context.Context, query ports.ChatListQuery) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now()
	db := r.db.WithContext(ctx).
		Model(&chat.Chat{}).
		Where("chats.deleted_at IS NULL").
		Where(`
			EXISTS (
				SELECT 1
				FROM chat_participants cp
				WHERE cp.chat_id = chats.id
				AND cp.user_id = ?
				AND cp.left_at IS NULL
				AND (
					cp.cleared_at IS NULL
					OR chats.last_message_timestamp IS NULL
					OR chats.last_message_timestamp > cp.cleared_at
				)
			)
		`, query.UserID).
		Where(`
			NOT EXISTS (
				SELECT 1
				FROM engagements e
				JOIN engagement_details ed ON ed.engagement_id = e.id
				WHERE e.contentable_id = chats.id
				AND e.contentable_type = ?
				AND (
					(ed.kind = ? AND ed.engager_id = ?)
					OR ed.kind = ?
				)
			)
		`,
			models.EngagementContentableTypeChat,
			models.EngagementKindChatDeletedForMe,
			query.UserID,
			models.EngagementKindChatDeletedForAll,
		).
		Preload("Participants", "left_at IS NULL").
		Preload("Participants.User", chatUserReadScope).
		Preload("Participants.User.Avatar", chatMediaReadScope).
		Preload("Participants.User.Avatar.File", chatAvatarFileReadScope).
		Preload("LastMessage", func(db *gorm.DB) *gorm.DB {
			return activeMessageScope(now)(db.Where("posts.deleted_at IS NULL"))
		}).
		Preload("LastMessage.Author", chatUserReadScope).
		Preload("LastMessage.Attachments", chatAttachmentMediaReadScope).
		Preload("LastMessage.Attachments.File", chatAttachmentFileReadScope).
		Preload("LastMessage.Author.Avatar", chatMediaReadScope).
		Preload("LastMessage.Author.Avatar.File", chatAvatarFileReadScope)

	if query.Cursor != nil {
		db = db.Where(
			"("+chatListActivityExpression+" < ?) OR ("+chatListActivityExpression+" = ? AND chats.id < ?)",
			query.Cursor.ActivityAt,
			query.Cursor.ActivityAt,
			query.Cursor.ChatID,
		)
	}
	return db
}

func (r *ChatRepository) ListChats(ctx context.Context, query ports.ChatListQuery) (ports.ChatListPage, error) {
	limit := boundedRepositoryChatLimit(query.Limit)
	query.Limit = limit

	var chats []chat.Chat
	err := r.chatsByUserIDQuery(ctx, query).
		Order(chatListActivityExpression + " DESC, chats.id DESC").
		Limit(limit + 1).
		Find(&chats).Error
	if err != nil {
		return ports.ChatListPage{}, err
	}

	hasMore := len(chats) > limit
	if hasMore {
		chats = chats[:limit]
	}
	lastMessages := make([]*post.Post, 0, len(chats))
	for i := range chats {
		if chats[i].LastMessage != nil {
			lastMessages = append(lastMessages, chats[i].LastMessage)
		}
	}
	if err := r.sanitizeMessagesForViewer(lastMessages, query.UserID); err != nil {
		return ports.ChatListPage{}, err
	}

	return ports.ChatListPage{Chats: chats, HasMore: hasMore}, nil
}

// GetChatsByUserIDWithCursor is retained for internal callers while sharing
// the same bounded implementation used by the production action.
func (r *ChatRepository) GetChatsByUserIDWithCursor(ctx context.Context, query ports.ChatListQuery) (ports.ChatListPage, error) {
	return r.ListChats(ctx, query)
}

func boundedRepositoryChatLimit(limit int) int {
	if limit <= 0 {
		return constants.DEFAULT_LIMIT
	}
	if limit > constants.MAXIMUM_LIMIT {
		return constants.MAXIMUM_LIMIT
	}
	return limit
}

func (r *ChatRepository) GetPrivateChatBetweenUsers(fromUser, toUser uuid.UUID) (*chat.Chat, error) {
	var chatObj chat.Chat
	err := r.privateChatBetweenUsersQuery(fromUser, toUser, time.Now()).
		First(&chatObj).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, chat.ErrChatNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := r.sanitizeMessagesForViewer(postPointers(chatObj.Messages), toUser); err != nil {
		return nil, err
	}
	return &chatObj, nil
}

func (r *ChatRepository) privateChatBetweenUsersQuery(fromUser, toUser uuid.UUID, now time.Time) *gorm.DB {
	return r.db.
		Joins("JOIN chat_participants cp1 ON cp1.chat_id = chats.id AND cp1.left_at IS NULL").
		Joins("JOIN chat_participants cp2 ON cp2.chat_id = chats.id AND cp2.left_at IS NULL").
		Where("chats.type = ?", chat.ChatTypePrivate).
		Where("cp1.user_id = ?", fromUser).
		Where("cp2.user_id = ?", toUser).
		Where("chats.deleted_at IS NULL").
		Preload("Participants", "left_at IS NULL").
		Preload("Participants.User", chatUserReadScope).
		Preload("Participants.User.Avatar", chatMediaReadScope).
		Preload("Participants.User.Avatar.File", chatAvatarFileReadScope).
		Preload("Messages", activeMessageScope(now)).
		Preload("Messages.Author", chatUserReadScope).
		Preload("Messages.Author.Avatar", chatMediaReadScope).
		Preload("Messages.Author.Avatar.File", chatAvatarFileReadScope).
		Preload("Messages.Attachments", chatAttachmentMediaReadScope).
		Preload("Messages.Attachments.File", chatAttachmentFileReadScope)
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

func (r *ChatRepository) SendTypingEvent(chatID, userID uuid.UUID, _ bool) error {
	if err := r.ensureActiveParticipant(context.Background(), chatID, userID); err != nil {
		return err
	}
	return nil
}

func (r *ChatRepository) AddMessageToChat(ctx context.Context, formData ports.FormData, author *models.User) (createdPost *post.Post, retErr error) {
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

	chatID, err := uuid.Parse(postForm.ChatID)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	createdMediaPaths := make([]string, 0, len(formData.Files))
	committed := false
	defer func() {
		if committed {
			return
		}
		cleanupErr := cleanupStoredUploads(createdMediaPaths)
		if recovered := recover(); recovered != nil {
			if cleanupErr != nil {
				helpers.Error("chat message rollback media cleanup error: %v", cleanupErr)
			}
			panic(recovered)
		}
		retErr = errors.Join(retErr, cleanupErr)
	}()

	var creation *contentablePostCreation
	retErr = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, _, err := loadChatMutationScope(tx, author, chatID); err != nil {
			return err
		}

		var err error
		creation, err = r.postRepo.transactionScoped(tx).createContentablePostInTransaction(
			ctx,
			formData,
			author,
			string(post.PostKindChat),
			&chatID,
			&createdMediaPaths,
		)
		if err != nil {
			return err
		}
		message := creation.post
		if result := tx.Model(&chat.Chat{}).
			Where("id = ?", chatID).
			Updates(map[string]interface{}{
				"last_message_id":        message.ID,
				"last_message_timestamp": message.CreatedAt,
			}); result.Error != nil {
			return result.Error
		} else if result.RowsAffected != 1 {
			return chat.ErrChatNotFound
		}

		return tx.Model(&chat.ChatParticipant{}).
			Where("chat_id = ? AND user_id <> ? AND left_at IS NULL", chatID, author.ID).
			Update("unread_count", gorm.Expr("unread_count + ?", 1)).Error
	})
	if retErr != nil {
		return nil, retErr
	}
	committed = true

	chatPost := creation.post
	chatPost.Author = *author
	chatPost.ClientID = formData.ClientID
	if hydrated, err := r.postRepo.GetPostByIDIncludingUnpublished(chatPost.ID); err == nil {
		chatPost = hydrated
		chatPost.ClientID = formData.ClientID
	} else {
		helpers.Error("hydrate committed chat message %s: %v", chatPost.ID, err)
	}

	newMessageNotification := fmt.Sprintf("You received a new message from %s. Click to read.", author.UserName)
	err = r.NotifyChatParticipants(chatID, *author, "New Message", newMessageNotification)
	if err != nil {
		fmt.Println("Error notifying chat participants:", err)
	}
	// The message and its chat metadata are already committed. Notification is
	// an external best-effort side effect and must not make the client retry a
	// successful send (which would duplicate the message).
	return chatPost, nil
}

func authenticatedChatUser(authUser *models.User, userID uuid.UUID) bool {
	return authUser != nil && authUser.ID != uuid.Nil && authUser.ID == userID
}

func loadChatMutationScope(tx *gorm.DB, authUser *models.User, chatID uuid.UUID) (*chat.Chat, *chat.ChatParticipant, error) {
	if tx == nil || authUser == nil || authUser.ID == uuid.Nil || chatID == uuid.Nil {
		return nil, nil, chat.ErrNotParticipant
	}

	var participant chat.ChatParticipant
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("chat_id = ? AND user_id = ? AND left_at IS NULL", chatID, authUser.ID).
		Take(&participant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, chat.ErrNotParticipant
	}
	if err != nil {
		return nil, nil, err
	}

	var chatEntity chat.Chat
	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", chatID).
		Take(&chatEntity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, chat.ErrChatNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	return &chatEntity, &participant, nil
}

func loadChatMutationMessage(tx *gorm.DB, chatID, messageID uuid.UUID) (*post.Post, error) {
	if tx == nil || chatID == uuid.Nil || messageID == uuid.Nil {
		return nil, chat.ErrMessageNotFound
	}
	var message post.Post
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(
			"id = ? AND post_kind = ? AND contentable_type = ? AND contentable_id = ?",
			messageID,
			post.PostKindMessage,
			post.PostKindChat,
			chatID,
		).
		Take(&message).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, chat.ErrMessageNotFound
	}
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *ChatRepository) withChatVisibilityMutation(ctx context.Context, mutation func(*gorm.DB) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if r.db.Name() != "postgres" {
		fallbackEngagementViewLock.Lock()
		defer fallbackEngagementViewLock.Unlock()
	}
	return r.db.WithContext(ctx).Transaction(mutation)
}

func chatVisibilityDedupeKey(kind models.EngagementKind, contentableType models.EngagementContentableType, contentableID, actorID uuid.UUID, global bool) string {
	scope := actorID.String()
	if global {
		scope = "global"
	}
	return fmt.Sprintf("chat-visibility:%s:%s:%s:%s", contentableType, contentableID, kind, scope)
}

func addChatVisibilityFlag(
	tx *gorm.DB,
	actorID, ownerID uuid.UUID,
	kind models.EngagementKind,
	contentableID uuid.UUID,
	contentableType models.EngagementContentableType,
	global bool,
) (bool, error) {
	if tx == nil || actorID == uuid.Nil || ownerID == uuid.Nil || contentableID == uuid.Nil {
		return false, errors.New("chat visibility identifiers are required")
	}
	if tx.Name() == "postgres" {
		if err := lockViewAggregate(tx, engagementAggregateLockKey(contentableType, contentableID)).Error; err != nil {
			return false, err
		}
	}
	aggregate, err := loadOrCreateEngagementAggregate(tx, contentableID, contentableType)
	if err != nil {
		return false, err
	}

	existing := tx.Model(&models.EngagementDetail{}).
		Where("engagement_id = ? AND kind = ?", aggregate.ID, kind)
	if !global {
		existing = existing.Where("engager_id = ?", actorID)
	}
	var count int64
	if err := existing.Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}

	now := time.Now().UTC()
	dedupeKey := chatVisibilityDedupeKey(kind, contentableType, contentableID, actorID, global)
	detail := models.EngagementDetail{
		ID: uuid.New(), EngagementID: aggregate.ID, DedupeKey: &dedupeKey,
		EngagerID: actorID, EngageeID: ownerID, Kind: kind,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := createEngagementDetailInTransaction(tx, &detail); err != nil {
		if errors.Is(err, errEngagementDetailAlreadyExists) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func clearPinnedMessage(tx *gorm.DB, chatID, messageID uuid.UUID) error {
	return tx.Model(&chat.Chat{}).
		Where("id = ? AND pinned_msg_id = ?", chatID, messageID).
		Updates(map[string]interface{}{
			"pinned_msg_id": nil,
			"pinned_by_id":  nil,
		}).Error
}

func hasGlobalMessageDeletionFlag(tx *gorm.DB, messageID uuid.UUID) (bool, error) {
	var count int64
	err := tx.Model(&models.EngagementDetail{}).
		Joins("JOIN engagements e ON e.id = engagement_details.engagement_id").
		Where(
			"e.contentable_id = ? AND e.contentable_type = ? AND engagement_details.kind = ?",
			messageID,
			models.EngagementContentableTypeMessage,
			models.EngagementKindMessageDeletedForAll,
		).
		Count(&count).Error
	return count > 0, err
}

func (r *ChatRepository) PinMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	if !authenticatedChatUser(authUser, userID) {
		return chat.ErrNotParticipant
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, _, err := loadChatMutationScope(tx, authUser, chatID); err != nil {
			return err
		}
		if _, err := loadChatMutationMessage(tx, chatID, messageID); err != nil {
			return err
		}
		return tx.Model(&chat.Chat{}).
			Where("id = ?", chatID).
			Updates(map[string]interface{}{
				"pinned_msg_id": messageID,
				"pinned_by_id":  authUser.ID,
			}).Error
	})
}

func (r *ChatRepository) UnpinMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	if !authenticatedChatUser(authUser, userID) {
		return chat.ErrNotParticipant
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, _, err := loadChatMutationScope(tx, authUser, chatID); err != nil {
			return err
		}
		if _, err := loadChatMutationMessage(tx, chatID, messageID); err != nil {
			return err
		}
		// The message ID is part of the command contract. It must never clear a
		// newer/different pin if an old unpin request arrives late.
		return tx.Model(&chat.Chat{}).
			Where("id = ? AND pinned_msg_id = ?", chatID, messageID).
			Updates(map[string]interface{}{
				"pinned_msg_id": nil,
				"pinned_by_id":  nil,
			}).Error
	})
}

func (r *ChatRepository) DeleteMessageForUser(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	if !authenticatedChatUser(authUser, userID) {
		return chat.ErrNotParticipant
	}
	return r.withChatVisibilityMutation(ctx, func(tx *gorm.DB) error {
		if _, _, err := loadChatMutationScope(tx, authUser, chatID); err != nil {
			return err
		}
		message, err := loadChatMutationMessage(tx, chatID, messageID)
		if err != nil {
			return err
		}
		_, err = addChatVisibilityFlag(tx, authUser.ID, message.AuthorID, models.EngagementKindMessageDeletedForMe, message.ID, models.EngagementContentableTypeMessage, false)
		return err
	})
}

func (r *ChatRepository) DeleteMessageForAll(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	if !authenticatedChatUser(authUser, userID) {
		return chat.ErrNotParticipant
	}
	return r.withChatVisibilityMutation(ctx, func(tx *gorm.DB) error {
		chatEntity, participant, err := loadChatMutationScope(tx, authUser, chatID)
		if err != nil {
			return err
		}
		message, err := loadChatMutationMessage(tx, chatID, messageID)
		if err != nil {
			return err
		}
		if !chatEntity.CanDeleteMessage(authUser.ID, message.AuthorID, authUser.UserRole, participant.Role) {
			return chat.ErrPermissionDenied
		}
		created, err := addChatVisibilityFlag(tx, authUser.ID, message.AuthorID, models.EngagementKindMessageDeletedForAll, message.ID, models.EngagementContentableTypeMessage, true)
		if err != nil {
			return err
		}
		if err := clearPinnedMessage(tx, chatID, message.ID); err != nil {
			return err
		}
		if created {
			if err := decrementExpiredMessageUnread(tx, chatID, message.AuthorID, message.CreatedAt).Error; err != nil {
				return err
			}
		}
		return repairChatLastMessage(tx, chatID, time.Now().UTC())
	})
}

func (r *ChatRepository) DeleteChatForUser(ctx context.Context, authUser *models.User, chatID, userID uuid.UUID) error {
	if !authenticatedChatUser(authUser, userID) {
		return chat.ErrNotParticipant
	}
	return r.withChatVisibilityMutation(ctx, func(tx *gorm.DB) error {
		chatEntity, _, err := loadChatMutationScope(tx, authUser, chatID)
		if err != nil {
			return err
		}
		_, err = addChatVisibilityFlag(tx, authUser.ID, chatEntity.CreatorID, models.EngagementKindChatDeletedForMe, chatEntity.ID, models.EngagementContentableTypeChat, false)
		return err
	})
}

func (r *ChatRepository) DeleteChatForAll(ctx context.Context, authUser *models.User, chatID uuid.UUID) error {
	if authUser == nil {
		return chat.ErrNotParticipant
	}
	return r.withChatVisibilityMutation(ctx, func(tx *gorm.DB) error {
		chatEntity, participant, err := loadChatMutationScope(tx, authUser, chatID)
		if err != nil {
			return err
		}
		if !chatEntity.CanModerate(authUser.ID, authUser.UserRole, participant.Role) {
			return chat.ErrPermissionDenied
		}
		_, err = addChatVisibilityFlag(tx, authUser.ID, chatEntity.CreatorID, models.EngagementKindChatDeletedForAll, chatEntity.ID, models.EngagementContentableTypeChat, true)
		return err
	})
}

func (r *ChatRepository) DeleteChat(ctx context.Context, authUser *models.User, chatID, userID uuid.UUID) error {
	// The public action is documented as deleting a chat for the authenticated
	// user; keep it as an idempotent visibility mutation, not a physical delete.
	return r.DeleteChatForUser(ctx, authUser, chatID, userID)
}

func (r *ChatRepository) DeleteMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	if !authenticatedChatUser(authUser, userID) {
		return chat.ErrNotParticipant
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		chatEntity, participant, err := loadChatMutationScope(tx, authUser, chatID)
		if err != nil {
			return err
		}
		message, err := loadChatMutationMessage(tx, chatID, messageID)
		if err != nil {
			return err
		}
		if !chatEntity.CanDeleteMessage(authUser.ID, message.AuthorID, authUser.UserRole, participant.Role) {
			return chat.ErrPermissionDenied
		}
		alreadyHiddenForAll, err := hasGlobalMessageDeletionFlag(tx, message.ID)
		if err != nil {
			return err
		}
		result := tx.
			Where("id = ? AND post_kind = ? AND contentable_type = ? AND contentable_id = ?", messageID, post.PostKindMessage, post.PostKindChat, chatID).
			Delete(&post.Post{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return chat.ErrMessageNotFound
		}
		if err := clearPinnedMessage(tx, chatID, message.ID); err != nil {
			return err
		}
		if !alreadyHiddenForAll {
			if err := decrementExpiredMessageUnread(tx, chatID, message.AuthorID, message.CreatedAt).Error; err != nil {
				return err
			}
		}
		return repairChatLastMessage(tx, chatID, time.Now().UTC())
	})
}

func (r *ChatRepository) NotifyChatParticipants(chatId uuid.UUID, author models.User, messageTitle, messageText string) error {

	// Katılımcıları ve user ilişkisini preload ile çek
	var participants []chat.ChatParticipant
	err := r.db.Preload("User").
		Where("chat_id = ? AND user_id <> ? AND left_at IS NULL", chatId, author.ID).
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
	page, err := r.ListChatMessages(context.Background(), ports.ChatMessageListQuery{
		UserID: userID,
		ChatID: chatID,
		Limit:  constants.DEFAULT_LIMIT,
	})
	return page.Messages, err
}

func (r *ChatRepository) messagesByChatIDQuery(userID uuid.UUID, chatID uuid.UUID, clearedAt *time.Time) *gorm.DB {
	query := r.db.Model(&post.Post{}).
		Where("posts.post_kind = ? AND posts.contentable_type = ? AND posts.contentable_id = ?", post.PostKindMessage, post.PostKindChat, chatID).
		Where("(posts.expires_at IS NULL OR posts.expires_at > ?)", time.Now()).
		Where(`
			NOT EXISTS (
				SELECT 1
				FROM engagements e
				JOIN engagement_details ed ON ed.engagement_id = e.id
				WHERE e.contentable_id = posts.id
				AND e.contentable_type = ?
				AND (
					(ed.kind = ? AND ed.engager_id = ?)
					OR ed.kind = ?
				)
			)
		`,
			models.EngagementContentableTypeMessage,
			models.EngagementKindMessageDeletedForMe,
			userID,
			models.EngagementKindMessageDeletedForAll,
		)

	if clearedAt != nil {
		query = query.Where("posts.created_at > ?", *clearedAt)
	}

	return query
}

func preloadChatMessageRelations(db *gorm.DB, now time.Time) *gorm.DB {
	return db.
		Preload("Author", chatUserReadScope).
		Preload("Parent", activeMessageScope(now)).
		Preload("Location").
		Preload("Parent.Author", chatUserReadScope).
		Preload("Parent.Author.Avatar", chatMediaReadScope).
		Preload("Parent.Author.Avatar.File", chatAvatarFileReadScope).
		Preload("Parent.Attachments", chatAttachmentMediaReadScope).
		Preload("Parent.Attachments.File", chatAttachmentFileReadScope).
		Preload("Author.Avatar", chatMediaReadScope).
		Preload("Author.Avatar.File", chatAvatarFileReadScope).
		Preload("Attachments", chatAttachmentMediaReadScope).
		Preload("Attachments.File", chatAttachmentFileReadScope)
}

func (r *ChatRepository) chatMessagesPageQuery(ctx context.Context, query ports.ChatMessageListQuery, clearedAt *time.Time) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	db := r.messagesByChatIDQuery(query.UserID, query.ChatID, clearedAt).WithContext(ctx)
	if query.Cursor != nil {
		db = db.Where("posts.public_id < ?", query.Cursor.PublicID)
	}
	return preloadChatMessageRelations(db, time.Now()).
		Order("posts.public_id DESC")
}

func (r *ChatRepository) ListChatMessages(ctx context.Context, query ports.ChatMessageListQuery) (ports.ChatMessageListPage, error) {
	limit := boundedRepositoryChatLimit(query.Limit)
	query.Limit = limit
	if ctx == nil {
		ctx = context.Background()
	}

	var participant chat.ChatParticipant
	if err := r.activeParticipantQuery(ctx, query.ChatID, query.UserID).Take(&participant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.ChatMessageListPage{}, chat.ErrNotParticipant
		}
		return ports.ChatMessageListPage{}, err
	}

	var messages []post.Post
	err := r.chatMessagesPageQuery(ctx, query, participant.ClearedAt).
		Limit(limit + 1).
		Find(&messages).Error

	if err != nil {
		return ports.ChatMessageListPage{}, err
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	if err := r.sanitizeMessagesForViewer(postPointers(messages), query.UserID); err != nil {
		return ports.ChatMessageListPage{}, err
	}

	now := time.Now()
	err = r.db.WithContext(ctx).Model(&chat.ChatParticipant{}).
		Where("chat_id = ? AND user_id = ? AND left_at IS NULL", query.ChatID, query.UserID).
		Updates(map[string]interface{}{
			"unread_count": 0,
			"last_read_at": now,
		}).Error

	if err != nil {
		return ports.ChatMessageListPage{}, err
	}
	return ports.ChatMessageListPage{Messages: messages, HasMore: hasMore}, nil
}

// GetMessagesByChatIDWithCursor is retained for internal callers while
// sharing the bounded and deletion-aware production implementation.
func (r *ChatRepository) GetMessagesByChatIDWithCursor(ctx context.Context, query ports.ChatMessageListQuery) (ports.ChatMessageListPage, error) {
	return r.ListChatMessages(ctx, query)
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
		Where("post_kind = ? AND contentable_type = ? AND contentable_id = ?", post.PostKindMessage, post.PostKindChat, chatID).
		Where("(expires_at IS NULL OR expires_at > ?)", now).
		Where(`
			NOT EXISTS (
				SELECT 1
				FROM engagements e
				JOIN engagement_details ed ON ed.engagement_id = e.id
				WHERE e.contentable_id = posts.id
				AND e.contentable_type = ?
				AND ed.kind = ?
			)
		`, models.EngagementContentableTypeMessage, models.EngagementKindMessageDeletedForAll).
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
			alreadyHiddenForAll, err := hasGlobalMessageDeletionFlag(tx, message.ID)
			if err != nil {
				return err
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
			if !alreadyHiddenForAll {
				if err := decrementExpiredMessageUnread(tx, chatID, message.AuthorID, message.CreatedAt).Error; err != nil {
					return err
				}
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
	if authUser == nil {
		return chat.ErrNotParticipant
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, _, err := loadChatMutationScope(tx, authUser, chatID); err != nil {
			return err
		}
		now := time.Now().UTC()
		return tx.Model(&chat.ChatParticipant{}).
			Where("chat_id = ? AND user_id = ? AND left_at IS NULL", chatID, authUser.ID).
			Updates(map[string]interface{}{
				"cleared_at":   now,
				"unread_count": 0,
				"last_read_at": now,
			}).Error
	})
}

func (r *ChatRepository) DeleteChatHistoryForAll(ctx context.Context, authUser *models.User, chatID uuid.UUID) error {
	if authUser == nil {
		return chat.ErrNotParticipant
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		chatEntity, participant, err := loadChatMutationScope(tx, authUser, chatID)
		if err != nil {
			return err
		}
		if !chatEntity.CanModerate(authUser.ID, authUser.UserRole, participant.Role) {
			return chat.ErrPermissionDenied
		}
		now := time.Now().UTC()
		return tx.Model(&chat.ChatParticipant{}).
			Where("chat_id = ?", chatID).
			Updates(map[string]interface{}{
				"cleared_at":   now,
				"unread_count": 0,
				"last_read_at": now,
			}).Error
	})
}

func (r *ChatRepository) MarkChatMessageRead(ctx context.Context, authUser *models.User, chatID uuid.UUID, messages []uuid.UUID) error {
	if authUser == nil {
		return chat.ErrNotParticipant
	}
	return r.withChatVisibilityMutation(ctx, func(tx *gorm.DB) error {
		if _, _, err := loadChatMutationScope(tx, authUser, chatID); err != nil {
			return err
		}

		seen := make(map[uuid.UUID]struct{}, len(messages))
		for _, messageID := range messages {
			if _, duplicate := seen[messageID]; duplicate {
				continue
			}
			seen[messageID] = struct{}{}
			message, err := loadChatMutationMessage(tx, chatID, messageID)
			if err != nil {
				return err
			}
			if _, err := addChatVisibilityFlag(
				tx,
				authUser.ID,
				message.AuthorID,
				models.EngagementKindChatMessageRead,
				message.ID,
				models.EngagementContentableTypeMessage,
				false,
			); err != nil {
				return err
			}
		}

		now := time.Now().UTC()
		return tx.Model(&chat.ChatParticipant{}).
			Where("chat_id = ? AND user_id = ? AND left_at IS NULL", chatID, authUser.ID).
			Updates(map[string]interface{}{
				"unread_count": 0,
				"last_read_at": now,
			}).Error
	})
}
