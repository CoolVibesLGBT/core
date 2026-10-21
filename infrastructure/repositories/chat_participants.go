package repositories

import (
	"context"
	"core/constants"
	domainmedia "core/domain/media"
	domainuser "core/domain/user"
	"core/models"
	"core/models/chat"
	"core/models/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *ChatRepository) CreateGroupChat(ctx context.Context, creatorID uuid.UUID, participantIDs []uuid.UUID, title, description *utils.LocalizedString) (*chat.Chat, error) {
	return r.createGroupWithAvatar(ctx, creatorID, participantIDs, title, description, nil)
}
func (r *ChatRepository) CreateGroupChatWithAvatar(ctx context.Context, creatorID uuid.UUID, participantIDs []uuid.UUID, title, description *utils.LocalizedString, avatar *domainmedia.UploadedFile, options ...chat.GroupOptions) (*chat.Chat, error) {
	return r.createGroupWithAvatar(ctx, creatorID, participantIDs, title, description, avatar, options...)
}
func (r *ChatRepository) createGroupWithAvatar(ctx context.Context, creatorID uuid.UUID, participantIDs []uuid.UUID, title, description *utils.LocalizedString, avatar *domainmedia.UploadedFile, options ...chat.GroupOptions) (*chat.Chat, error) {
	if avatar != nil {
		if err := validateGroupAvatar(avatar); err != nil {
			return nil, err
		}
	}
	storedPath := ""
	committed := false
	defer func() {
		if !committed && storedPath != "" {
			_ = removeStoredUpload(storedPath)
		}
	}()

	config := chat.GroupOptions{Visibility: "private"}
	if len(options) > 0 {
		config = options[0]
	}
	visibility, username, validationErr := config.Normalize()
	if validationErr != nil {
		return nil, validationErr
	}
	entity := &chat.Chat{ID: uuid.New(), Type: chat.ChatTypeGroup, CreatorID: creatorID, Title: title, Description: description, Visibility: visibility, PublicUsername: username}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		seen := map[uuid.UUID]bool{creatorID: true}
		entity.Participants = []chat.ChatParticipant{{ID: uuid.New(), ChatID: entity.ID, UserID: creatorID, Role: chat.ParticipantRoleOwner, JoinedAt: now}}
		for _, id := range participantIDs {
			if seen[id] {
				continue
			}
			seen[id] = true
			if err := ensureSocialInteractionAllowed(tx, creatorID, id, domainuser.InteractionChat); err != nil {
				return err
			}
			entity.Participants = append(entity.Participants, chat.ChatParticipant{ID: uuid.New(), ChatID: entity.ID, UserID: id, Role: chat.ParticipantRoleMember, JoinedAt: now})
		}
		if len(entity.Participants) < 2 {
			return constants.ErrInvalidParticipantsLength
		}
		if err := tx.Create(entity).Error; err != nil {
			return err
		}
		if avatar != nil {
			item, err := r.saveGroupAvatar(tx, entity.ID, creatorID, avatar)
			if err != nil {
				return err
			}
			storedPath = item.File.StoragePath
			entity.Avatar, entity.AvatarID = item, &item.ID
			if err := tx.Model(entity).Update("avatar_id", item.ID).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	committed = true
	return entity, nil
}

// Lock the chat before its participants so membership changes serialize without
// two moderators locking one another's participant rows in opposite order.
func loadGroupParticipantScope(tx *gorm.DB, actor *models.User, chatID uuid.UUID) (*chat.Chat, *chat.ChatParticipant, error) {
	entity, participant, err := loadChatMutationScope(tx, actor, chatID)
	if err != nil {
		return nil, nil, err
	}
	if entity.Type != chat.ChatTypeGroup {
		return nil, nil, constants.ErrUnsupportedChatType
	}
	var deleted int64
	err = tx.Model(&chat.Chat{}).Where("id = ? AND id IN (?)", chatID, deletedChatContentIDs(tx, models.EngagementContentableTypeChat, uuid.Nil, models.EngagementKindChatDeletedForMe, models.EngagementKindChatDeletedForAll)).Count(&deleted).Error
	if err != nil {
		return nil, nil, err
	}
	if deleted > 0 {
		return nil, nil, chat.ErrChatNotFound
	}
	return entity, participant, nil
}

func (r *ChatRepository) AddParticipant(ctx context.Context, actor *models.User, chatID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		entity, participant, err := loadGroupParticipantScope(tx, actor, chatID)
		if err != nil {
			return err
		}
		if !entity.CanModerate(actor.ID, actor.UserRole, participant.Role) {
			return chat.ErrPermissionDenied
		}
		if userID == actor.ID {
			return nil
		}
		if err := ensureSocialInteractionAllowed(tx, actor.ID, userID, domainuser.InteractionChat); err != nil {
			return err
		}
		now := time.Now().UTC()
		member := chat.ChatParticipant{ID: uuid.New(), ChatID: chatID, UserID: userID, Role: chat.ParticipantRoleMember, JoinedAt: now}
		// A departed member reuses the same identity. Active membership is an
		// idempotent no-op, preserving roles, read state, and the original join time.
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "chat_id"}, {Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{"left_at": nil, "joined_at": now, "role": chat.ParticipantRoleMember, "unread_count": 0, "updated_at": now}),
			Where:     clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "chat_participants.left_at IS NOT NULL"}}},
		}).Omit(clause.Associations).Create(&member).Error; err != nil {
			return err
		}
		return tx.Model(entity).Update("updated_at", now).Error
	})
}

func (r *ChatRepository) RemoveParticipant(ctx context.Context, actor *models.User, chatID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		entity, participant, err := loadGroupParticipantScope(tx, actor, chatID)
		if err != nil {
			return err
		}
		if actor.ID != userID && (!entity.CanModerate(actor.ID, actor.UserRole, participant.Role) || userID == entity.CreatorID) {
			return chat.ErrPermissionDenied
		}
		if userID == entity.CreatorID {
			var others int64
			if err := tx.Model(&chat.ChatParticipant{}).Where("chat_id = ? AND user_id <> ? AND left_at IS NULL", chatID, userID).Count(&others).Error; err != nil {
				return err
			}
			if others > 0 {
				return constants.ErrGroupOwnerCannotLeave
			}
		}
		now := time.Now().UTC()
		if err := tx.Model(&chat.ChatParticipant{}).Where("chat_id = ? AND user_id = ? AND left_at IS NULL", chatID, userID).Updates(map[string]any{"left_at": now, "unread_count": 0, "updated_at": now}).Error; err != nil {
			return err
		}
		if userID == entity.CreatorID {
			return tx.Delete(entity).Error
		}
		return tx.Model(entity).Update("updated_at", now).Error
	})
}

func (r *ChatRepository) GetParticipants(ctx context.Context, actor *models.User, chatID uuid.UUID) (participants []chat.ChatParticipant, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, _, err := loadGroupParticipantScope(tx, actor, chatID); err != nil {
			return err
		}
		return tx.Where("chat_id = ? AND left_at IS NULL", chatID).Order("joined_at ASC, id ASC").
			Preload("User", chatUserReadScope).Preload("User.Avatar", chatMediaReadScope).Preload("User.Avatar.File", chatAvatarFileReadScope).Find(&participants).Error
	})
	return participants, err
}
