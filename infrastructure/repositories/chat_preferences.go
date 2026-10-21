package repositories

import (
	"context"
	"core/models"
	"core/models/chat"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (r *ChatRepository) SetChatMuted(ctx context.Context, actor *models.User, chatID uuid.UUID, muted bool) error {
	if actor == nil {
		return chat.ErrNotParticipant
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, _, err := loadChatMutationScope(tx, actor, chatID); err != nil {
			return err
		}
		return tx.Model(&chat.ChatParticipant{}).Where("chat_id = ? AND user_id = ? AND left_at IS NULL", chatID, actor.ID).Update("is_muted", muted).Error
	})
}
