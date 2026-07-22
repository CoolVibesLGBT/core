package usecases

import (
	"context"
	"core/application/ports"
	domainmedia "core/domain/media"
	"errors"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type MediaAccessDecision struct {
	Allowed bool
	Public  bool
}

type MediaAccessService struct {
	repository             ports.MediaAccessRepository
	privatePhotoAuthorizer ports.PrivatePhotoAccessAuthorizer
}

func NewMediaAccessService(repository ports.MediaAccessRepository, authorizers ...ports.PrivatePhotoAccessAuthorizer) *MediaAccessService {
	service := &MediaAccessService{repository: repository}
	if len(authorizers) > 0 {
		service.privatePhotoAuthorizer = authorizers[0]
	} else if authorizer, ok := repository.(ports.PrivatePhotoAccessAuthorizer); ok {
		service.privatePhotoAuthorizer = authorizer
	}
	return service
}

func (s *MediaAccessService) Authorize(ctx context.Context, storagePrefix string, userPublicID *int64, requestedPaths ...string) (MediaAccessDecision, error) {
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
	if mediaAccessIsPrivatePhoto(access) && !policy.PrivilegedViewer && !policy.OwnerViewer {
		// Album grants intentionally cover only re-encoded variants. The raw
		// upload may retain EXIF/GPS metadata and must remain owner-only even
		// when a viewer can infer its filename from a variant URL.
		if len(requestedPaths) == 0 || !privatePhotoProcessedVariant(requestedPaths[0]) || s.privatePhotoAuthorizer == nil {
			return MediaAccessDecision{}, nil
		}
		policy.PrivatePhotoGrant, err = s.privatePhotoAuthorizer.HasApprovedPrivatePhotoAccess(ctx, access.OwnerID, principal.ID)
		if err != nil {
			return MediaAccessDecision{}, err
		}
	}
	if policy.ChatMedia && access.ChatID != nil {
		policy.ChatParticipant, err = s.repository.IsActiveChatParticipant(ctx, *access.ChatID, principal.ID)
		if err != nil {
			return MediaAccessDecision{}, err
		}
	}

	return MediaAccessDecision{Allowed: policy.Accessible()}, nil
}

func privatePhotoProcessedVariant(requestedPath string) bool {
	extension := strings.ToLower(filepath.Ext(requestedPath))
	if extension != ".webp" {
		return false
	}
	stem := strings.TrimSuffix(filepath.Base(requestedPath), extension)
	for _, aspect := range []string{"square", "landscape", "portrait"} {
		for _, size := range []string{"icon", "thumb", "sm", "md", "lg"} {
			if strings.HasSuffix(stem, "_"+aspect+"_"+size) {
				return true
			}
		}
	}
	return false
}

func mediaAccessIsPrivatePhoto(access ports.MediaFileAccess) bool {
	return strings.EqualFold(strings.TrimSpace(access.Role), "private_photo") &&
		strings.EqualFold(strings.TrimSpace(access.OwnerType), "user")
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
