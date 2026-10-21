package repositories

import (
	"context"
	"core/models"
	"core/models/chat"
	"core/models/media"
	"core/models/post"
	modelutils "core/models/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func mediaFileIDsByStoragePrefix(db *gorm.DB, prefix string) *gorm.DB {
	return repositorySubquery(db).Model(&modelutils.FileMetadata{}).Select("id").
		Where("storage_path = ? OR storage_path LIKE ?", prefix, prefix+".%")
}

func (r *MediaRepository) HasVisibleChatFileAccess(ctx context.Context, prefix string, viewer uuid.UUID) (bool, error) {
	if viewer == uuid.Nil {
		return false, nil
	}
	db := r.db.WithContext(ctx)
	attachments := repositorySubquery(db).Model(&media.Media{}).Select("owner_id").
		Where("owner_type = ?", media.OwnerPost).
		Where("file_id IN (?)", mediaFileIDsByStoragePrefix(db, prefix))
	chats := repositorySubquery(db).Model(&chat.Chat{}).Select("id").
		Where("id NOT IN (?)", deletedChatContentIDs(db, models.EngagementContentableTypeChat,
			viewer, models.EngagementKindChatDeletedForMe, models.EngagementKindChatDeletedForAll))
	participants := repositorySubquery(db).Model(&chat.ChatParticipant{}).Select("chat_id").
		Where("user_id = ? AND left_at IS NULL", viewer).
		Where("chat_id IN (?)", chats).
		Where("cleared_at IS NULL OR posts.created_at > cleared_at")
	query := db.Model(&post.Post{}).
		Where("id IN (?)", attachments).
		Where("contentable_id IN (?)", participants).
		Where("post_kind = ? AND contentable_type = ?", post.PostKindMessage, post.PostKindChat).
		Where("view_once = ? AND expires_in_seconds IS NULL", false).
		Scopes(activeMessageScope(time.Now().UTC()))
	var count int64
	err := visibleChatMessages(query, viewer, nil).Count(&count).Error
	return count > 0, err
}

func (r *MediaRepository) IsPublicGroupAvatar(ctx context.Context, prefix string) (bool, error) {
	db := r.db.WithContext(ctx)
	avatars := repositorySubquery(db).Model(&media.Media{}).Select("id").
		Where("owner_type = ?", media.OwnerChat).
		Where("owner_id = chats.id").
		Where("file_id IN (?)", mediaFileIDsByStoragePrefix(db, prefix))
	var count int64
	err := db.Model(&chat.Chat{}).
		Where("type = ? AND visibility = ?", chat.ChatTypeGroup, "public").
		Where("avatar_id IN (?)", avatars).Count(&count).Error
	return count > 0, err
}
