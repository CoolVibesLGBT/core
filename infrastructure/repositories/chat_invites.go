package repositories

import (
	"context"
	"core/application/ports"
	domainuser "core/domain/user"
	"core/models"
	"core/models/chat"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
	"time"
)

var _ ports.ChatGroupInvitesRepository = (*ChatRepository)(nil)

func (r *ChatRepository) ExportGroupInvite(ctx context.Context, actor *models.User, id uuid.UUID, rotate bool) (*chat.GroupInvite, error) {
	var invitation chat.GroupInvite
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		entity, member, err := loadGroupParticipantScope(tx, actor, id)
		if err != nil {
			return err
		}
		if !entity.CanModerate(actor.ID, actor.UserRole, member.Role) {
			return chat.ErrPermissionDenied
		}
		err = tx.Take(&invitation, "chat_id = ?", id).Error
		if err == nil && !rotate {
			return nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		data := make([]byte, 32)
		if _, err := rand.Read(data); err != nil {
			return err
		}
		token := base64.RawURLEncoding.EncodeToString(data)
		invitation = chat.GroupInvite{ID: uuid.New(), ChatID: id, Token: token, CreatedBy: actor.ID}
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "chat_id"}}, DoUpdates: clause.Assignments(map[string]interface{}{"token": token, "created_by": actor.ID, "uses": 0, "updated_at": time.Now().UTC()})}).Create(&invitation).Error
	})
	if err != nil {
		return nil, err
	}
	return &invitation, nil
}
func findInvitedGroup(tx *gorm.DB, token, username string) (*chat.Chat, error) {
	if (token == "") == (username == "") {
		return nil, chat.ErrGroupLinkInvalid
	}
	var entity chat.Chat
	query := tx.Where("type = ?", chat.ChatTypeGroup)
	if token != "" {
		if len(token) != 43 {
			return nil, chat.ErrGroupLinkInvalid
		}
		query = query.Where("id IN (?)", tx.Model(&chat.GroupInvite{}).Select("chat_id").Where("token = ?", token))
	} else {
		query = query.Where("visibility = ? AND public_username = ?", "public", strings.ToLower(username))
	}
	if err := query.Take(&entity).Error; err != nil {
		return nil, chat.ErrGroupLinkInvalid
	}
	var hidden int64
	if err := tx.Model(&chat.Chat{}).Where("id = ? AND id IN (?)", entity.ID, deletedChatContentIDs(tx, models.EngagementContentableTypeChat, uuid.Nil, models.EngagementKindChatDeletedForMe, models.EngagementKindChatDeletedForAll)).Count(&hidden).Error; err != nil {
		return nil, err
	}
	if hidden > 0 {
		return nil, chat.ErrGroupLinkInvalid
	}
	return &entity, nil
}
func (r *ChatRepository) PreviewGroup(ctx context.Context, token, username string) (*chat.Chat, int64, error) {
	tx := r.db.WithContext(ctx)
	entity, err := findInvitedGroup(tx, token, username)
	if err != nil {
		return nil, 0, err
	}
	var count int64
	if err := tx.Model(&chat.ChatParticipant{}).Where("chat_id = ? AND left_at IS NULL", entity.ID).Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := tx.Preload("Avatar.File").Take(entity, "id = ?", entity.ID).Error; err != nil {
		return nil, 0, err
	}
	return entity, count, nil
}
func (r *ChatRepository) JoinGroup(ctx context.Context, actor *models.User, token, username string) (*chat.Chat, error) {
	if actor == nil {
		return nil, chat.ErrNotParticipant
	}
	var entity *chat.Chat
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		found, err := findInvitedGroup(tx, token, username)
		if err != nil {
			return err
		}
		var locked chat.Chat
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Take(&locked, "id = ?", found.ID).Error; err != nil {
			return err
		}
		// Revalidate after the group lock: a link may have been revoked or made private while waiting.
		current, err := findInvitedGroup(tx, token, username)
		if err != nil {
			return err
		}
		entity = current
		var invitation chat.GroupInvite
		if token != "" {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Take(&invitation, "chat_id = ? AND token = ?", entity.ID, token).Error; err != nil {
				return chat.ErrGroupLinkInvalid
			}
		}
		var existing chat.ChatParticipant
		err = tx.Take(&existing, "chat_id = ? AND user_id = ?", entity.ID, actor.ID).Error
		if err == nil && existing.LeftAt == nil {
			return nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := ensureSocialInteractionAllowed(tx, entity.CreatorID, actor.ID, domainuser.InteractionChat); err != nil {
			return err
		}
		member := chat.ChatParticipant{ID: uuid.New(), ChatID: entity.ID, UserID: actor.ID, Role: chat.ParticipantRoleMember, JoinedAt: time.Now().UTC()}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "chat_id"}, {Name: "user_id"}}, DoUpdates: clause.Assignments(map[string]interface{}{"left_at": nil, "role": chat.ParticipantRoleMember, "joined_at": member.JoinedAt, "unread_count": 0, "updated_at": member.JoinedAt})}).Create(&member).Error; err != nil {
			return err
		}
		if token != "" {
			return tx.Model(&chat.GroupInvite{}).Where("id = ?", invitation.ID).Update("uses", gorm.Expr("uses + 1")).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Preload("Avatar.File").Preload("Participants", "left_at IS NULL").Preload("Participants.User", chatUserReadScope).Preload("Participants.User.Avatar.File").Take(entity, "id = ?", entity.ID).Error; err != nil {
		return nil, err
	}
	return entity, nil
}
