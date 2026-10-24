package repositories

import (
	"context"
	"core/application/ports"
	"core/models"
	"core/models/chat"
	"github.com/google/uuid"
	"strings"
)

func (r *ChatRepository) ListPublicGroups(ctx context.Context, query ports.PublicGroupQuery) (ports.PublicGroupPage, error) {
	limit := boundedRepositoryChatLimit(query.Limit)
	db := r.db.WithContext(ctx).Model(&chat.Chat{}).
		Select("chats.*, (?) AS member_count", r.db.Model(&chat.ChatParticipant{}).Select("COUNT(*)").Where("chat_participants.chat_id = chats.id AND left_at IS NULL")).
		Where("chats.type = ? AND chats.visibility = ?", chat.ChatTypeGroup, "public").
		Where("chats.id NOT IN (?)", deletedChatContentIDs(r.db, models.EngagementContentableTypeChat, uuid.Nil, models.EngagementKindChatDeletedForMe, models.EngagementKindChatDeletedForAll)).
		Preload("Avatar", chatMediaReadScope).Preload("Avatar.File", chatAvatarFileReadScope)
	if query.Cursor != nil {
		db = db.Where("chats.id < ?", *query.Cursor)
	}
	if search := strings.TrimSpace(query.Search); search != "" {
		escaped := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(strings.ToLower(search))
		pattern := "%" + escaped + "%"
		db = db.Where("(LOWER(chats.public_username) LIKE ? OR LOWER(CAST(chats.title AS TEXT)) LIKE ? OR LOWER(CAST(chats.description AS TEXT)) LIKE ?)", pattern, pattern, pattern)
	}
	rows := []ports.PublicGroupRow{}
	if err := db.Order("chats.id DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return ports.PublicGroupPage{}, err
	}
	more := len(rows) > limit
	if more {
		rows = rows[:limit]
	}
	return ports.PublicGroupPage{Groups: rows, HasMore: more}, nil
}
