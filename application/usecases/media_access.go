package usecases

import (
	"context"
	"core/application/ports"
	domainmedia "core/domain/media"
	"errors"
	"strings"

	"github.com/google/uuid"
)

type MediaAccessDecision struct {
	Allowed bool
	Public  bool
}

type MediaAccessService struct {
	repository ports.MediaAccessRepository
}

func NewMediaAccessService(repository ports.MediaAccessRepository) *MediaAccessService {
	return &MediaAccessService{repository: repository}
}

func (s *MediaAccessService) Authorize(ctx context.Context, storagePrefix string, userPublicID *int64) (MediaAccessDecision, error) {
	access, err := s.repository.FindMediaFileAccess(ctx, storagePrefix)
	if err != nil {
		return MediaAccessDecision{}, err
	}

	policy := domainmedia.FileAccessPolicy{
		MediaPublic:    access.IsPublic,
		AttachedToPost: access.PostID != nil,
		PostPublished:  access.Published,
		Audience:       stringValue(access.Audience),
		ChatMedia:      mediaAccessIsChat(access),
	}
	if policy.PubliclyAccessible() {
		return MediaAccessDecision{Allowed: true, Public: true}, nil
	}
	if userPublicID == nil {
		return MediaAccessDecision{}, nil
	}

	principal, err := s.repository.FindMediaAccessPrincipal(ctx, *userPublicID)
	if errors.Is(err, ports.ErrNotFound) {
		return MediaAccessDecision{}, nil
	}
	if err != nil {
		return MediaAccessDecision{}, err
	}
	policy.PrivilegedViewer = isMediaModerator(principal.Role)
	policy.OwnerViewer = mediaAccessOwnedBy(access, principal.ID)
	if policy.ChatMedia && access.ChatID != nil {
		policy.ChatParticipant, err = s.repository.IsActiveChatParticipant(ctx, *access.ChatID, principal.ID)
		if err != nil {
			return MediaAccessDecision{}, err
		}
	}

	return MediaAccessDecision{Allowed: policy.Accessible()}, nil
}

func mediaAccessIsChat(access ports.MediaFileAccess) bool {
	switch strings.ToLower(strings.TrimSpace(access.Role)) {
	case "chat_image", "chat_media", "chat_video":
		return true
	}
	switch strings.ToLower(strings.TrimSpace(access.PostKind)) {
	case "chat", "message":
		return true
	}
	return strings.EqualFold(strings.TrimSpace(access.ContentableType), "chat")
}

func mediaAccessOwnedBy(access ports.MediaFileAccess, principalID uuid.UUID) bool {
	if mediaAccessIsChat(access) {
		return false
	}
	if access.PostAuthorID != nil {
		return *access.PostAuthorID == principalID
	}
	return access.PostID == nil && strings.EqualFold(access.OwnerType, "user") && access.OwnerID == principalID
}

func isMediaModerator(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "moderator", "admin", "super_admin":
		return true
	default:
		return false
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
