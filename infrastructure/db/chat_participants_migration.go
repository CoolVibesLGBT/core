package db

import (
	"core/models/chat"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repair legacy duplicate identities before AutoMigrate installs the unique
// (chat,user) index. The newest membership state wins; retain the furthest
// history boundary so merging records cannot reveal cleared messages.
func MigrateChatParticipants(database *gorm.DB) error {
	if !database.Migrator().HasTable(&chat.ChatParticipant{}) || database.Migrator().HasIndex(&chat.ChatParticipant{}, "idx_chat_participants_chat_user") {
		return nil
	}
	return database.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("LOCK TABLE chat_participants IN ACCESS EXCLUSIVE MODE").Error; err != nil {
			return err
		}
		var members []chat.ChatParticipant
		if err := tx.Order("chat_id ASC, user_id ASC, updated_at DESC, created_at DESC, id DESC").Find(&members).Error; err != nil {
			return err
		}
		for start := 0; start < len(members); {
			end := start + 1
			winner := members[start]
			for end < len(members) && members[end].ChatID == winner.ChatID && members[end].UserID == winner.UserID {
				other := members[end]
				if other.ClearedAt != nil && (winner.ClearedAt == nil || other.ClearedAt.After(*winner.ClearedAt)) {
					winner.ClearedAt = other.ClearedAt
				}
				end++
			}
			for i := start + 1; i < end; i++ {
				if err := tx.Delete(&chat.ChatParticipant{}, "id = ?", members[i].ID).Error; err != nil {
					return err
				}
			}
			changes := map[string]any{"cleared_at": winner.ClearedAt}
			if winner.ID == uuid.Nil {
				changes["id"] = uuid.New()
			}
			if winner.JoinedAt.IsZero() {
				changes["joined_at"] = winner.CreatedAt
			}
			if err := tx.Model(&chat.ChatParticipant{}).Where("id = ?", winner.ID).Updates(changes).Error; err != nil {
				return err
			}
			start = end
		}
		return tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_chat_participants_chat_user ON chat_participants (chat_id, user_id)").Error
	})
}
