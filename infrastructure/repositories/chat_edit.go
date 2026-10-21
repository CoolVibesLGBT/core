package repositories

import (
	"context"
	"core/application/ports"
	"core/models"
	"core/models/chat"
	"core/models/post"
	modelutils "core/models/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"strings"
	"time"
	"unicode/utf8"
)

var _ ports.ChatMessageEditorRepository = (*ChatRepository)(nil)

func (r *ChatRepository) EditMessage(ctx context.Context, actor *models.User, chatID, messageID uuid.UUID, text string) (*post.Post, error) {
	if actor == nil {
		return nil, chat.ErrNotParticipant
	}
	if strings.TrimSpace(text) == "" || utf8.RuneCountInString(text) > 4096 {
		return nil, chat.ErrEmptyMessage
	}
	var edited post.Post
	var clearedAt *time.Time
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_, participant, err := loadChatMutationScope(tx, actor, chatID)
		if err != nil {
			return err
		}
		clearedAt = participant.ClearedAt
		message, err := loadChatMutationMessage(tx, chatID, messageID)
		if err != nil {
			return err
		}
		if message.AuthorID != actor.ID || message.ViewOnce || message.ExpiresInSeconds != nil {
			return chat.ErrPermissionDenied
		}
		if err := visibleChatMessages(tx.Where("posts.id = ?", messageID), actor.ID, participant.ClearedAt).Take(&edited).Error; err != nil {
			return chat.ErrMessageNotFound
		}
		language := actor.DefaultLanguage
		if language == "" {
			language = "en"
		}
		content := modelutils.LocalizedString{language: text}
		return tx.Model(&post.Post{}).Where("id = ?", messageID).Updates(map[string]interface{}{"content": &content, "updated_at": time.Now().UTC()}).Error
	})
	if err != nil {
		return nil, err
	}
	if err := preloadChatMessageRelations(r.db.WithContext(ctx), time.Now(), actor.ID, clearedAt).Where("posts.id = ?", messageID).Take(&edited).Error; err != nil {
		return nil, err
	}
	messages := []post.Post{edited}
	if err := r.sanitizeMessagesForViewer(postPointers(messages), actor.ID); err != nil {
		return nil, err
	}
	return &messages[0], nil
}
