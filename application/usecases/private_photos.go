package usecases

import (
	"context"
	"core/application/ports"
	domainmedia "core/domain/media"
	modelmedia "core/models/media"
	modelutils "core/models/utils"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MaxPrivatePhotosPerOwner  = 20
	MaxPrivatePhotosPerUpload = 10
)

var (
	ErrPrivatePhotoPrincipalRequired = errors.New("authenticated user is required")
	ErrPrivatePhotoOwnerRequired     = errors.New("owner_id is required")
	ErrPrivatePhotoRequestRequired   = errors.New("request_id is required")
	ErrPrivatePhotoViewerRequired    = errors.New("viewer_id is required")
	ErrPrivatePhotoIDRequired        = errors.New("photo_id is required")
	ErrPrivatePhotoUserNotFound      = errors.New("private photo user not found")
	ErrPrivatePhotoNotFound          = errors.New("private photo not found")
	ErrPrivatePhotoRequestNotFound   = errors.New("private photo access request not found")
	ErrPrivatePhotoForbidden         = errors.New("private photo action is forbidden")
	ErrPrivatePhotoSelfRequest       = errors.New("cannot request your own private photos")
	ErrPrivatePhotoFilesRequired     = errors.New("at least one private photo is required")
	ErrPrivatePhotoUploadLimit       = errors.New("private photo upload limit exceeded")
	ErrPrivatePhotoOwnerLimit        = errors.New("private photo album limit reached")
	ErrPrivatePhotoImageRequired     = domainmedia.ErrPrivatePhotoImageRequired
	ErrPrivatePhotoImageDimensions   = domainmedia.ErrPrivatePhotoImageDimensions
	ErrPhotoDestinationRequired      = errors.New("destination must be public or private")
)

type PrivatePhotoAccessSummary struct {
	Status    string  `json:"status"`
	RequestID *string `json:"request_id,omitempty"`
}

type PrivatePhotoFetchResult struct {
	OwnerID string                    `json:"owner_id"`
	Count   int64                     `json:"count"`
	Access  PrivatePhotoAccessSummary `json:"access"`
	Photos  []PrivatePhotoMediaView   `json:"photos"`
}

type ProfilePhotoFetchResult struct {
	OwnerID string                  `json:"owner_id"`
	Count   int64                   `json:"count"`
	Photos  []PrivatePhotoMediaView `json:"photos"`
}

type MoveProfilePhotoResult struct {
	Photo       PrivatePhotoMediaView `json:"photo"`
	Destination string                `json:"destination"`
}

// PrivatePhotoMediaView deliberately exposes only the public media ID and
// re-encoded image variants. Database UUIDs, storage paths, original names,
// and the raw/original upload URL never leave the private-photo API.
type PrivatePhotoMediaView struct {
	PublicID         string                      `json:"public_id"`
	ProcessingStatus modelmedia.ProcessingStatus `json:"processing_status"`
	File             PrivatePhotoFileView        `json:"file"`
	CreatedAt        time.Time                   `json:"created_at"`
}

type PrivatePhotoFileView struct {
	URL      string                        `json:"url,omitempty"`
	MimeType string                        `json:"mime_type,omitempty"`
	Width    *int                          `json:"width,omitempty"`
	Height   *int                          `json:"height,omitempty"`
	Variants *PrivatePhotoFileVariantsView `json:"variants,omitempty"`
}

type PrivatePhotoFileVariantsView struct {
	Image *PrivatePhotoImageVariantsView `json:"image,omitempty"`
}

type PrivatePhotoImageVariantsView struct {
	Icon      *modelutils.VariantInfo `json:"icon,omitempty"`
	Thumbnail *modelutils.VariantInfo `json:"thumbnail,omitempty"`
	Small     *modelutils.VariantInfo `json:"small,omitempty"`
	Medium    *modelutils.VariantInfo `json:"medium,omitempty"`
	Large     *modelutils.VariantInfo `json:"large,omitempty"`
}

type PrivatePhotoUserView struct {
	PublicID    string                 `json:"public_id"`
	Username    string                 `json:"username"`
	DisplayName string                 `json:"displayname"`
	Avatar      *PrivatePhotoMediaView `json:"avatar,omitempty"`
}

type PrivatePhotoAccessRequestView struct {
	RequestID   string                               `json:"request_id"`
	OwnerID     string                               `json:"owner_id"`
	ViewerID    string                               `json:"viewer_id"`
	Status      domainmedia.PrivatePhotoAccessStatus `json:"status"`
	RequestedAt time.Time                            `json:"requested_at"`
	RespondedAt *time.Time                           `json:"responded_at,omitempty"`
	Viewer      PrivatePhotoUserView                 `json:"viewer"`
}

type PrivatePhotoService struct {
	repository ports.PrivatePhotoRepository
	notifier   ports.PrivatePhotoNotifier
	realtime   ports.PrivatePhotoRealtimePublisher
}

func NewPrivatePhotoService(
	repository ports.PrivatePhotoRepository,
	notifier ports.PrivatePhotoNotifier,
	realtime ports.PrivatePhotoRealtimePublisher,
) *PrivatePhotoService {
	return &PrivatePhotoService{repository: repository, notifier: notifier, realtime: realtime}
}

func (s *PrivatePhotoService) Fetch(ctx context.Context, principal ports.PrivatePhotoPrincipal, ownerPublicID int64) (PrivatePhotoFetchResult, error) {
	if err := validatePrivatePhotoPrincipal(principal); err != nil {
		return PrivatePhotoFetchResult{}, err
	}
	if ownerPublicID <= 0 {
		return PrivatePhotoFetchResult{}, ErrPrivatePhotoOwnerRequired
	}

	owner, err := s.repository.FindPrivatePhotoUserByPublicID(ctx, ownerPublicID)
	if errors.Is(err, ports.ErrNotFound) {
		return PrivatePhotoFetchResult{}, ErrPrivatePhotoUserNotFound
	}
	if err != nil {
		return PrivatePhotoFetchResult{}, err
	}
	if owner.ID != principal.ID {
		blocked, blockErr := s.repository.ArePrivatePhotoUsersBlocked(ctx, owner.ID, principal.ID)
		if blockErr != nil {
			return PrivatePhotoFetchResult{}, blockErr
		}
		if blocked {
			return PrivatePhotoFetchResult{}, ErrPrivatePhotoForbidden
		}
	}
	count, err := s.repository.CountPrivatePhotos(ctx, owner.ID)
	if err != nil {
		return PrivatePhotoFetchResult{}, err
	}

	result := PrivatePhotoFetchResult{
		OwnerID: publicIDString(owner.PublicID),
		Count:   count,
		Access:  PrivatePhotoAccessSummary{Status: "none"},
		Photos:  make([]PrivatePhotoMediaView, 0),
	}

	canView := false
	if owner.ID == principal.ID {
		result.Access.Status = "owner"
		canView = true
	} else {
		request, requestErr := s.repository.GetPrivatePhotoAccess(ctx, owner.ID, principal.ID)
		switch {
		case errors.Is(requestErr, ports.ErrNotFound):
			// The default none state is intentionally returned without a URL.
		case requestErr != nil:
			return PrivatePhotoFetchResult{}, requestErr
		default:
			result.Access = privatePhotoAccessSummary(request)
			canView = request.Status.CanView()
		}
	}

	if !canView {
		return result, nil
	}
	photos, err := s.repository.ListPrivatePhotos(ctx, owner.ID)
	if err != nil {
		return PrivatePhotoFetchResult{}, err
	}
	if owner.ID != principal.ID {
		visible := make([]modelmedia.Media, 0, len(photos))
		for _, photo := range photos {
			if photo.ProcessingStatus != modelmedia.ProcessingStatusFailed {
				visible = append(visible, photo)
			}
		}
		photos = visible
	}
	result.Count = int64(len(photos))
	result.Photos = privatePhotoMediaViews(photos)
	return result, nil
}

func (s *PrivatePhotoService) FetchProfilePhotos(ctx context.Context, principal ports.PrivatePhotoPrincipal, ownerPublicID int64) (ProfilePhotoFetchResult, error) {
	if err := validatePrivatePhotoPrincipal(principal); err != nil {
		return ProfilePhotoFetchResult{}, err
	}
	if ownerPublicID <= 0 {
		return ProfilePhotoFetchResult{}, ErrPrivatePhotoOwnerRequired
	}
	owner, err := s.repository.FindPrivatePhotoUserByPublicID(ctx, ownerPublicID)
	if errors.Is(err, ports.ErrNotFound) {
		return ProfilePhotoFetchResult{}, ErrPrivatePhotoUserNotFound
	}
	if err != nil {
		return ProfilePhotoFetchResult{}, err
	}
	if owner.ID != principal.ID {
		blocked, blockErr := s.repository.ArePrivatePhotoUsersBlocked(ctx, owner.ID, principal.ID)
		if blockErr != nil {
			return ProfilePhotoFetchResult{}, blockErr
		}
		if blocked {
			return ProfilePhotoFetchResult{}, ErrPrivatePhotoForbidden
		}
	}
	repository, ok := s.repository.(ports.ProfilePhotoRepository)
	if !ok {
		return ProfilePhotoFetchResult{}, errors.New("profile photo repository is not configured")
	}
	photos, err := repository.ListProfilePhotos(ctx, owner.ID)
	if err != nil {
		return ProfilePhotoFetchResult{}, err
	}
	return ProfilePhotoFetchResult{
		OwnerID: publicIDString(owner.PublicID),
		Count:   int64(len(photos)),
		Photos:  privatePhotoMediaViews(photos),
	}, nil
}

func (s *PrivatePhotoService) MoveProfilePhoto(ctx context.Context, principal ports.PrivatePhotoPrincipal, photoPublicID int64, destination string) (MoveProfilePhotoResult, error) {
	if err := validatePrivatePhotoPrincipal(principal); err != nil {
		return MoveProfilePhotoResult{}, err
	}
	if photoPublicID <= 0 {
		return MoveProfilePhotoResult{}, ErrPrivatePhotoIDRequired
	}
	normalizedDestination := strings.ToLower(strings.TrimSpace(destination))
	var role modelmedia.MediaRole
	switch normalizedDestination {
	case "public":
		role = modelmedia.RoleProfile
	case "private":
		role = modelmedia.RolePrivatePhoto
	default:
		return MoveProfilePhotoResult{}, ErrPhotoDestinationRequired
	}
	repository, ok := s.repository.(ports.ProfilePhotoRepository)
	if !ok {
		return MoveProfilePhotoResult{}, errors.New("profile photo repository is not configured")
	}
	photo, err := repository.MoveProfilePhoto(ctx, principal.ID, photoPublicID, role, MaxPrivatePhotosPerOwner)
	if errors.Is(err, ports.ErrNotFound) {
		return MoveProfilePhotoResult{}, ErrPrivatePhotoNotFound
	}
	if errors.Is(err, ports.ErrPrivatePhotoLimitReached) {
		return MoveProfilePhotoResult{}, ErrPrivatePhotoOwnerLimit
	}
	if err != nil {
		return MoveProfilePhotoResult{}, err
	}
	s.publishPrivatePhotoAlbumChanged(ctx, principal.ID, principal.PublicID)
	return MoveProfilePhotoResult{
		Photo:       privatePhotoMediaView(*photo),
		Destination: normalizedDestination,
	}, nil
}

func (s *PrivatePhotoService) Upload(ctx context.Context, principal ports.PrivatePhotoPrincipal, files []ports.UploadedFile) ([]PrivatePhotoMediaView, error) {
	if err := validatePrivatePhotoPrincipal(principal); err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, ErrPrivatePhotoFilesRequired
	}
	if len(files) > MaxPrivatePhotosPerUpload {
		return nil, ErrPrivatePhotoUploadLimit
	}
	for _, file := range files {
		if file == nil {
			return nil, ErrPrivatePhotoImageRequired
		}
		if file.Size() <= 0 {
			return nil, ErrPrivatePhotoImageRequired
		}
		contentType := strings.ToLower(strings.TrimSpace(file.ContentType()))
		if contentType != "" && !strings.HasPrefix(contentType, "image/") {
			return nil, ErrPrivatePhotoImageRequired
		}
	}
	count, err := s.repository.CountPrivatePhotos(ctx, principal.ID)
	if err != nil {
		return nil, err
	}
	if count+int64(len(files)) > MaxPrivatePhotosPerOwner {
		return nil, ErrPrivatePhotoOwnerLimit
	}

	created := make([]modelmedia.Media, 0, len(files))
	for _, file := range files {
		photo, createErr := s.repository.AddPrivatePhoto(ctx, principal.ID, file, MaxPrivatePhotosPerOwner)
		if createErr != nil {
			for _, rollbackPhoto := range created {
				_ = s.repository.DeletePrivatePhoto(ctx, principal.ID, rollbackPhoto.PublicID)
			}
			if errors.Is(createErr, ports.ErrPrivatePhotoLimitReached) {
				return nil, ErrPrivatePhotoOwnerLimit
			}
			return nil, createErr
		}
		created = append(created, *photo)
	}
	views := privatePhotoMediaViews(created)
	s.publishPrivatePhotoAlbumChanged(ctx, principal.ID, principal.PublicID)
	return views, nil
}

func (s *PrivatePhotoService) Delete(ctx context.Context, principal ports.PrivatePhotoPrincipal, photoPublicID int64) error {
	if err := validatePrivatePhotoPrincipal(principal); err != nil {
		return err
	}
	if photoPublicID <= 0 {
		return ErrPrivatePhotoIDRequired
	}
	err := s.repository.DeletePrivatePhoto(ctx, principal.ID, photoPublicID)
	if errors.Is(err, ports.ErrNotFound) {
		return ErrPrivatePhotoNotFound
	}
	if err != nil {
		return err
	}
	s.publishPrivatePhotoAlbumChanged(ctx, principal.ID, principal.PublicID)
	return nil
}

func (s *PrivatePhotoService) RequestAccess(ctx context.Context, principal ports.PrivatePhotoPrincipal, ownerPublicID int64) (PrivatePhotoAccessSummary, error) {
	if err := validatePrivatePhotoPrincipal(principal); err != nil {
		return PrivatePhotoAccessSummary{}, err
	}
	if ownerPublicID <= 0 {
		return PrivatePhotoAccessSummary{}, ErrPrivatePhotoOwnerRequired
	}
	if ownerPublicID == principal.PublicID {
		return PrivatePhotoAccessSummary{}, ErrPrivatePhotoSelfRequest
	}

	owner, err := s.repository.FindPrivatePhotoUserByPublicID(ctx, ownerPublicID)
	if errors.Is(err, ports.ErrNotFound) {
		return PrivatePhotoAccessSummary{}, ErrPrivatePhotoUserNotFound
	}
	if err != nil {
		return PrivatePhotoAccessSummary{}, err
	}
	if owner.ID == principal.ID {
		return PrivatePhotoAccessSummary{}, ErrPrivatePhotoSelfRequest
	}
	viewer, err := s.repository.FindPrivatePhotoUserByPublicID(ctx, principal.PublicID)
	if err != nil {
		return PrivatePhotoAccessSummary{}, err
	}
	blocked, err := s.repository.ArePrivatePhotoUsersBlocked(ctx, owner.ID, principal.ID)
	if err != nil {
		return PrivatePhotoAccessSummary{}, err
	}
	if blocked {
		return PrivatePhotoAccessSummary{}, ErrPrivatePhotoForbidden
	}

	request, changed, err := s.repository.RequestPrivatePhotoAccess(ctx, owner.ID, principal.ID, time.Now().UTC())
	if err != nil {
		return PrivatePhotoAccessSummary{}, err
	}
	if changed && s.notifier != nil {
		_ = s.notifier.NotifyPrivatePhotoAccessRequested(ctx, owner, viewer, *request)
	}
	if changed {
		s.publishPrivatePhotoEvent(ctx, []int64{owner.PublicID, principal.PublicID}, ports.PrivatePhotoRealtimeEnvelope{
			Version:    ports.PrivatePhotoRealtimeVersion,
			EventID:    uuid.NewString(),
			Type:       ports.PrivatePhotoEventAccessRequested,
			OccurredAt: time.Now().UTC(),
			Data: ports.PrivatePhotoRealtimeEventData{
				RequestID: publicIDString(request.PublicID),
				OwnerID:   publicIDString(owner.PublicID),
				ViewerID:  publicIDString(principal.PublicID),
				Status:    string(request.Status),
			},
		})
	}
	return privatePhotoAccessSummary(request), nil
}

func (s *PrivatePhotoService) ListAccessRequests(ctx context.Context, principal ports.PrivatePhotoPrincipal) ([]PrivatePhotoAccessRequestView, error) {
	if err := validatePrivatePhotoPrincipal(principal); err != nil {
		return nil, err
	}
	records, err := s.repository.ListPrivatePhotoAccessRequests(ctx, principal.ID)
	if err != nil {
		return nil, err
	}
	result := make([]PrivatePhotoAccessRequestView, 0, len(records))
	for _, record := range records {
		result = append(result, privatePhotoAccessRequestView(record))
	}
	return result, nil
}

func (s *PrivatePhotoService) RespondAccess(ctx context.Context, principal ports.PrivatePhotoPrincipal, requestPublicID int64, decisionValue string) (PrivatePhotoAccessSummary, error) {
	if err := validatePrivatePhotoPrincipal(principal); err != nil {
		return PrivatePhotoAccessSummary{}, err
	}
	if requestPublicID <= 0 {
		return PrivatePhotoAccessSummary{}, ErrPrivatePhotoRequestRequired
	}
	decision, err := domainmedia.ParsePrivatePhotoAccessDecision(decisionValue)
	if err != nil {
		return PrivatePhotoAccessSummary{}, err
	}

	existing, err := s.repository.FindPrivatePhotoAccessByPublicID(ctx, requestPublicID)
	if errors.Is(err, ports.ErrNotFound) {
		return PrivatePhotoAccessSummary{}, ErrPrivatePhotoRequestNotFound
	}
	if err != nil {
		return PrivatePhotoAccessSummary{}, err
	}
	if existing.OwnerID != principal.ID {
		return PrivatePhotoAccessSummary{}, ErrPrivatePhotoForbidden
	}
	if decision == domainmedia.PrivatePhotoAccessApproved {
		blocked, blockErr := s.repository.ArePrivatePhotoUsersBlocked(ctx, existing.OwnerID, existing.ViewerID)
		if blockErr != nil {
			return PrivatePhotoAccessSummary{}, blockErr
		}
		if blocked {
			return PrivatePhotoAccessSummary{}, ErrPrivatePhotoForbidden
		}
	}

	request, changed, err := s.repository.RespondPrivatePhotoAccess(ctx, requestPublicID, principal.ID, decision, time.Now().UTC())
	if errors.Is(err, ports.ErrNotFound) {
		return PrivatePhotoAccessSummary{}, ErrPrivatePhotoRequestNotFound
	}
	if err != nil {
		return PrivatePhotoAccessSummary{}, err
	}
	if changed && s.notifier != nil {
		owner, ownerErr := s.repository.FindPrivatePhotoUserByPublicID(ctx, principal.PublicID)
		if ownerErr == nil {
			_ = s.notifier.NotifyPrivatePhotoAccessResponded(ctx, owner, request.Viewer, *request)
		}
	}
	if changed {
		s.publishPrivatePhotoEvent(ctx, []int64{principal.PublicID, request.ViewerPublicID}, ports.PrivatePhotoRealtimeEnvelope{
			Version:    ports.PrivatePhotoRealtimeVersion,
			EventID:    uuid.NewString(),
			Type:       ports.PrivatePhotoEventAccessUpdated,
			OccurredAt: time.Now().UTC(),
			Data: ports.PrivatePhotoRealtimeEventData{
				RequestID: publicIDString(request.PublicID),
				OwnerID:   publicIDString(principal.PublicID),
				ViewerID:  publicIDString(request.ViewerPublicID),
				Status:    string(request.Status),
			},
		})
	}
	return privatePhotoAccessSummary(request), nil
}

func (s *PrivatePhotoService) RevokeAccess(ctx context.Context, principal ports.PrivatePhotoPrincipal, viewerPublicID int64) (PrivatePhotoAccessSummary, error) {
	if err := validatePrivatePhotoPrincipal(principal); err != nil {
		return PrivatePhotoAccessSummary{}, err
	}
	if viewerPublicID <= 0 {
		return PrivatePhotoAccessSummary{}, ErrPrivatePhotoViewerRequired
	}
	if viewerPublicID == principal.PublicID {
		return PrivatePhotoAccessSummary{}, ErrPrivatePhotoForbidden
	}
	viewer, err := s.repository.FindPrivatePhotoUserByPublicID(ctx, viewerPublicID)
	if errors.Is(err, ports.ErrNotFound) {
		return PrivatePhotoAccessSummary{}, ErrPrivatePhotoUserNotFound
	}
	if err != nil {
		return PrivatePhotoAccessSummary{}, err
	}
	if viewer.ID == principal.ID {
		return PrivatePhotoAccessSummary{}, ErrPrivatePhotoForbidden
	}

	request, changed, err := s.repository.RevokePrivatePhotoAccess(ctx, principal.ID, viewer.ID, time.Now().UTC())
	if errors.Is(err, ports.ErrNotFound) {
		return PrivatePhotoAccessSummary{}, ErrPrivatePhotoRequestNotFound
	}
	if err != nil {
		return PrivatePhotoAccessSummary{}, err
	}
	if changed {
		s.publishPrivatePhotoEvent(ctx, []int64{principal.PublicID, viewer.PublicID}, ports.PrivatePhotoRealtimeEnvelope{
			Version:    ports.PrivatePhotoRealtimeVersion,
			EventID:    uuid.NewString(),
			Type:       ports.PrivatePhotoEventAccessRevoked,
			OccurredAt: time.Now().UTC(),
			Data: ports.PrivatePhotoRealtimeEventData{
				RequestID: publicIDString(request.PublicID),
				OwnerID:   publicIDString(principal.PublicID),
				ViewerID:  publicIDString(viewer.PublicID),
				Status:    string(request.Status),
			},
		})
	}
	return privatePhotoAccessSummary(request), nil
}

func (s *PrivatePhotoService) publishPrivatePhotoAlbumChanged(ctx context.Context, ownerID uuid.UUID, ownerPublicID int64) {
	if s.realtime == nil || ownerID == uuid.Nil || ownerPublicID <= 0 {
		return
	}

	s.publishPrivatePhotoEvent(ctx, s.privatePhotoAudience(ctx, ownerID, ownerPublicID), ports.PrivatePhotoRealtimeEnvelope{
		Version:    ports.PrivatePhotoRealtimeVersion,
		EventID:    uuid.NewString(),
		Type:       ports.PrivatePhotoEventAlbumChanged,
		OccurredAt: time.Now().UTC(),
		Data: ports.PrivatePhotoRealtimeEventData{
			OwnerID: publicIDString(ownerPublicID),
		},
	})
}

// MediaProcessingUpdated is called by the media worker after a private image
// reaches a terminal state. It emits identifiers and status only; processed
// variant URLs remain available exclusively through the authorized HTTP API.
func (s *PrivatePhotoService) MediaProcessingUpdated(ctx context.Context, item *modelmedia.Media, status modelmedia.ProcessingStatus) {
	if s.realtime == nil || item == nil || item.Role != modelmedia.RolePrivatePhoto {
		return
	}
	if status != modelmedia.ProcessingStatusReady && status != modelmedia.ProcessingStatusFailed {
		return
	}
	owner, err := s.repository.FindPrivatePhotoUserByID(ctx, item.OwnerID)
	if err != nil || owner.PublicID <= 0 {
		return
	}
	s.publishPrivatePhotoEvent(ctx, s.privatePhotoAudience(ctx, owner.ID, owner.PublicID), ports.PrivatePhotoRealtimeEnvelope{
		Version:    ports.PrivatePhotoRealtimeVersion,
		EventID:    uuid.NewString(),
		Type:       ports.PrivatePhotoEventMediaProcessingUpdated,
		OccurredAt: time.Now().UTC(),
		Data: ports.PrivatePhotoRealtimeEventData{
			OwnerID: publicIDString(owner.PublicID),
			PhotoID: publicIDString(item.PublicID),
			Status:  string(status),
		},
	})
}

func (s *PrivatePhotoService) privatePhotoAudience(ctx context.Context, ownerID uuid.UUID, ownerPublicID int64) []int64 {
	recipients := []int64{ownerPublicID}
	records, err := s.repository.ListPrivatePhotoAccessRequests(ctx, ownerID)
	if err == nil {
		for _, record := range records {
			if record.Status == domainmedia.PrivatePhotoAccessApproved {
				recipients = append(recipients, record.ViewerPublicID)
			}
		}
	}
	return recipients
}

func (s *PrivatePhotoService) publishPrivatePhotoEvent(ctx context.Context, recipients []int64, event ports.PrivatePhotoRealtimeEnvelope) {
	if s.realtime == nil {
		return
	}
	unique := make([]int64, 0, len(recipients))
	seen := make(map[int64]struct{}, len(recipients))
	for _, recipient := range recipients {
		if recipient <= 0 {
			continue
		}
		if _, exists := seen[recipient]; exists {
			continue
		}
		seen[recipient] = struct{}{}
		unique = append(unique, recipient)
	}
	if len(unique) == 0 {
		return
	}
	_ = s.realtime.PublishPrivatePhotoEvent(ctx, unique, event)
}

func validatePrivatePhotoPrincipal(principal ports.PrivatePhotoPrincipal) error {
	if principal.ID == uuid.Nil || principal.PublicID <= 0 {
		return ErrPrivatePhotoPrincipalRequired
	}
	return nil
}

func privatePhotoAccessSummary(record *ports.PrivatePhotoAccessRecord) PrivatePhotoAccessSummary {
	requestID := publicIDString(record.PublicID)
	return PrivatePhotoAccessSummary{
		Status:    string(record.Status),
		RequestID: &requestID,
	}
}

func privatePhotoAccessRequestView(record ports.PrivatePhotoAccessRecord) PrivatePhotoAccessRequestView {
	var avatar *PrivatePhotoMediaView
	if record.Viewer.Avatar != nil {
		view := privatePhotoMediaView(*record.Viewer.Avatar)
		avatar = &view
	}
	return PrivatePhotoAccessRequestView{
		RequestID:   publicIDString(record.PublicID),
		OwnerID:     publicIDString(record.OwnerPublicID),
		ViewerID:    publicIDString(record.ViewerPublicID),
		Status:      record.Status,
		RequestedAt: record.RequestedAt,
		RespondedAt: record.RespondedAt,
		Viewer: PrivatePhotoUserView{
			PublicID:    publicIDString(record.Viewer.PublicID),
			Username:    record.Viewer.UserName,
			DisplayName: record.Viewer.DisplayName,
			Avatar:      avatar,
		},
	}
}

func publicIDString(value int64) string {
	return strconv.FormatInt(value, 10)
}

func privatePhotoMediaViews(photos []modelmedia.Media) []PrivatePhotoMediaView {
	views := make([]PrivatePhotoMediaView, 0, len(photos))
	for _, photo := range photos {
		views = append(views, privatePhotoMediaView(photo))
	}
	return views
}

func privatePhotoMediaView(photo modelmedia.Media) PrivatePhotoMediaView {
	view := PrivatePhotoMediaView{
		PublicID:         publicIDString(photo.PublicID),
		ProcessingStatus: photo.ProcessingStatus,
		CreatedAt:        photo.CreatedAt,
		File: PrivatePhotoFileView{
			MimeType: photo.File.MimeType,
			Width:    photo.File.Width,
			Height:   photo.File.Height,
		},
	}
	if photo.File.Variants == nil || photo.File.Variants.Image == nil {
		return view
	}

	image := photo.File.Variants.Image
	view.File.Variants = &PrivatePhotoFileVariantsView{Image: &PrivatePhotoImageVariantsView{
		Icon:      image.Icon,
		Thumbnail: image.Thumbnail,
		Small:     image.Small,
		Medium:    image.Medium,
		Large:     image.Large,
	}}
	for _, variant := range []*modelutils.VariantInfo{image.Large, image.Medium, image.Small, image.Thumbnail, image.Icon} {
		if variant != nil && strings.TrimSpace(variant.URL) != "" {
			view.File.URL = variant.URL
			break
		}
	}
	return view
}
