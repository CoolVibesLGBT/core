package repositories

import (
	"context"
	"core/application/ports"
	"core/constants"
	domainmedia "core/domain/media"
	"core/models"
	"core/models/chat"
	"core/models/media"
	modelutils "core/models/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strings"
	"unicode/utf8"
)

var _ ports.ChatGroupMediaRepository = (*ChatRepository)(nil)

func validateGroupAvatar(file *domainmedia.UploadedFile) (retErr error) {
	if file == nil || file.Validate() != nil || file.Size > constants.GroupAvatarMaxBytes {
		return chat.ErrInvalidGroupAvatar
	}
	reader, err := file.Open()
	if err != nil {
		return chat.ErrInvalidGroupAvatar
	}
	defer func() {
		if err := reader.Close(); err != nil {
			retErr = chat.ErrInvalidGroupAvatar
		}
	}()
	config, format, err := image.DecodeConfig(io.LimitReader(reader, constants.GroupAvatarHeaderReadLimit))
	if err != nil {
		return chat.ErrInvalidGroupAvatar
	}
	if !validImageDimensions(config, constants.GroupAvatarMaxDimension, constants.GroupAvatarMaxPixels) {
		return chat.ErrInvalidGroupAvatar
	}
	switch format {
	case constants.ImageFormatJPEG, constants.ImageFormatPNG, constants.ImageFormatAVIF:
	default:
		return chat.ErrInvalidGroupAvatar
	}
	canonicalMIME, _, supported := imageFormatMetadata(format)
	if !supported || file.ContentType != canonicalMIME {
		return chat.ErrInvalidGroupAvatar
	}
	return nil
}
func (r *ChatRepository) saveGroupAvatar(tx *gorm.DB, chatID, actorID uuid.UUID, avatar *domainmedia.UploadedFile) (*media.Media, error) {
	// Private chat-owned media: the existing image pipeline generates all variants.
	return NewMediaRepository(tx, r.Node()).AddMedia(chatID, media.OwnerChat, actorID, media.RoleChatImage, avatar)
}
func (r *ChatRepository) UpdateGroup(ctx context.Context, actor *models.User, chatID uuid.UUID, title, description string, avatar *domainmedia.UploadedFile, remove bool, options ...chat.GroupOptions) (*chat.Chat, error) {
	if actor == nil {
		return nil, chat.ErrNotParticipant
	}
	title = strings.TrimSpace(title)
	if title == "" || utf8.RuneCountInString(title) > 128 || utf8.RuneCountInString(description) > 255 {
		return nil, chat.ErrEmptyMessage
	}
	if avatar != nil {
		if err := validateGroupAvatar(avatar); err != nil {
			return nil, err
		}
	}
	var entity *chat.Chat
	storedPath := ""
	committed := false
	defer func() {
		if !committed && storedPath != "" {
			_ = removeStoredUpload(storedPath)
		}
	}()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, member, err := loadGroupParticipantScope(tx, actor, chatID)
		if err != nil {
			return err
		}
		if !current.CanModerate(actor.ID, actor.UserRole, member.Role) {
			return chat.ErrPermissionDenied
		}
		language := actor.DefaultLanguage
		if language == "" {
			language = "en"
		}
		updates := map[string]interface{}{"title": modelutils.MakeLocalizedString(language, title), "description": modelutils.MakeLocalizedString(language, strings.TrimSpace(description))}
		if len(options) > 0 {
			visibility, username, err := options[0].Normalize()
			if err != nil {
				return err
			}
			updates["visibility"] = visibility
			updates["public_username"] = username
		}
		if remove {
			updates["avatar_id"] = nil
		}
		if avatar != nil {
			item, err := r.saveGroupAvatar(tx, chatID, actor.ID, avatar)
			if err != nil {
				return err
			}
			storedPath = item.File.StoragePath
			updates["avatar_id"] = item.ID
		}
		if err := tx.Model(current).Updates(updates).Error; err != nil {
			return err
		}
		entity = current
		return tx.Preload("Avatar.File").Preload("Participants", "left_at IS NULL").Preload("Participants.User", chatUserReadScope).Preload("Participants.User.Avatar.File").Take(entity, "id = ?", chatID).Error
	})
	if err != nil {
		return nil, err
	}
	committed = true
	return entity, nil
}
