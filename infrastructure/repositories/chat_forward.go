package repositories

import (
	"context"
	"core/application/ports"
	"core/models"
	"core/models/chat"
	"core/models/media"
	"core/models/post"
	modelutils "core/models/utils"
	"encoding/json"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var _ ports.ChatMessageForwarderRepository = (*ChatRepository)(nil)

// A batch commits in full or not at all. Forwarding copies message/media ownership,
// while retaining the underlying file rather than downloading a protected URL.
func (r *ChatRepository) ForwardMessages(ctx context.Context, actor *models.User, sourceID, targetID uuid.UUID, ids []uuid.UUID, comment string) ([]post.Post, error) {
	if actor == nil {
		return nil, chat.ErrNotParticipant
	}
	comment = strings.TrimSpace(comment)
	if utf8.RuneCountInString(comment) > 4096 {
		return nil, chat.ErrEmptyMessage
	}
	if len(ids) == 0 || len(ids) > 100 {
		return nil, chat.ErrMessageNotFound
	}
	seen := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		if id == uuid.Nil || seen[id] {
			return nil, chat.ErrMessageNotFound
		}
		seen[id] = true
	}
	var result []post.Post
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Use the same chat-before-participant lock order as all other mutations.
		rooms := []uuid.UUID{sourceID}
		if sourceID != targetID {
			rooms = append(rooms, targetID)
		}
		sort.Slice(rooms, func(i, j int) bool { return rooms[i].String() < rooms[j].String() })
		var sourceMember *chat.ChatParticipant
		for _, roomID := range rooms {
			_, member, err := loadChatMutationScope(tx, actor, roomID)
			if err != nil {
				return err
			}
			var deleted int64
			if err := tx.Model(&chat.Chat{}).Where("id = ? AND id IN (?)", roomID, deletedChatContentIDs(tx, models.EngagementContentableTypeChat, actor.ID, models.EngagementKindChatDeletedForMe, models.EngagementKindChatDeletedForAll)).Count(&deleted).Error; err != nil {
				return err
			}
			if deleted > 0 {
				return chat.ErrChatNotFound
			}
			if roomID == sourceID {
				sourceMember = member
			}
		}
		var originals []post.Post
		query := visibleChatMessages(tx, actor.ID, sourceMember.ClearedAt).
			Where("posts.id IN ? AND posts.contentable_id = ? AND posts.contentable_type = ? AND posts.post_kind = ?", ids, sourceID, post.PostKindChat, post.PostKindMessage).
			Where("posts.expires_at IS NULL OR posts.expires_at > ?", time.Now().UTC()).
			Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Author").Preload("Attachments").Order("posts.public_id ASC")
		if err := query.Find(&originals).Error; err != nil {
			return err
		}
		if len(originals) != len(ids) {
			return chat.ErrMessageNotFound
		}
		for _, original := range originals {
			if original.ViewOnce || original.ExpiresInSeconds != nil || original.ExpiresAt != nil {
				return chat.ErrPermissionDenied
			}
		}
		if comment != "" {
			language := actor.DefaultLanguage
			if language == "" {
				language = "en"
			}
			content := modelutils.LocalizedString{language: comment}
			kind := string(post.PostKindChat)
			audience := "private"
			now := time.Now().UTC()
			caption := post.Post{ID: uuid.New(), PublicID: r.Node().Generate().Int64(), AuthorID: actor.ID, Domain: actor.Domain, PostKind: post.PostKindMessage, ContentableID: &targetID, ContentableType: &kind, Content: &content, Audience: &audience, CreatedAt: now, UpdatedAt: now}
			if err := tx.Omit(clause.Associations).Create(&caption).Error; err != nil {
				return err
			}
			result = append(result, caption)
		}
		for _, original := range originals {
			origin := chat.ForwardOriginFromExtras(original.Extras)
			if origin == nil {
				name := strings.TrimSpace(original.Author.DisplayName)
				if name == "" {
					name = original.Author.UserName
				}
				origin = &chat.ForwardOrigin{Name: name, Date: original.CreatedAt}
			}
			extras := map[string]interface{}{"forwarded_from": origin}
			if emoticon, err := chat.ParseEmoticonExtras([]string{string(original.Extras)}); err == nil && emoticon != nil {
				extras["emoticon"] = emoticon
			}
			encoded, err := json.Marshal(extras)
			if err != nil {
				return err
			}
			now := time.Now().UTC()
			contentType := string(post.PostKindChat)
			audience := "private"
			forwarded := post.Post{ID: uuid.New(), PublicID: r.Node().Generate().Int64(), AuthorID: actor.ID, Domain: actor.Domain, PostKind: post.PostKindMessage, ContentableID: &targetID, ContentableType: &contentType, Content: original.Content, ContentCategory: original.ContentCategory, Audience: &audience, Extras: datatypes.JSON(encoded), Published: original.Published, CreatedAt: now, UpdatedAt: now}
			if err := tx.Omit(clause.Associations).Create(&forwarded).Error; err != nil {
				return err
			}
			for _, attachment := range original.Attachments {
				if attachment == nil {
					continue
				}
				clone := media.Media{ID: uuid.New(), PublicID: r.Node().Generate().Int64(), FileID: attachment.FileID, OwnerID: forwarded.ID, OwnerType: media.OwnerPost, UserID: actor.ID, Role: attachment.Role, IsPublic: false, ProcessingStatus: attachment.ProcessingStatus, CreatedAt: now, UpdatedAt: now}
				if err := tx.Omit(clause.Associations).Create(&clone).Error; err != nil {
					return err
				}
			}
			result = append(result, forwarded)
		}
		last := result[len(result)-1]
		if err := tx.Model(&chat.Chat{}).Where("id = ?", targetID).Updates(map[string]interface{}{"last_message_id": last.ID, "last_message_timestamp": last.CreatedAt, "updated_at": last.CreatedAt}).Error; err != nil {
			return err
		}
		if err := tx.Model(&chat.ChatParticipant{}).Where("chat_id = ? AND user_id <> ? AND left_at IS NULL", targetID, actor.ID).Update("unread_count", gorm.Expr("unread_count + ?", len(result))).Error; err != nil {
			return err
		}
		// Hydrate before commit so a response failure cannot leave an unreported committed batch.
		for i := range result {
			if err := preloadChatMessageRelations(tx, time.Now(), actor.ID, nil).Where("posts.id = ?", result[i].ID).Take(&result[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if r.notificationRepo != nil {
		_ = r.NotifyChatParticipants(ctx, targetID, *actor, "New Message", "You received a forwarded message.")
	}
	return result, nil
}
