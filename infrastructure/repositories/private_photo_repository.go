package repositories

import (
	"bufio"
	"context"
	"core/application/ports"
	domainmedia "core/domain/media"
	"core/helpers"
	"core/models"
	modelmedia "core/models/media"
	"core/models/notifications"
	modelutils "core/models/utils"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime"
	"strings"
	"time"

	_ "github.com/chai2010/webp"
	"github.com/google/uuid"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PrivatePhotoRepository struct {
	db            *gorm.DB
	snowFlakeNode *helpers.Node
	mediaRepo     *MediaRepository
}

const (
	maxPrivatePhotoDimension = 12000
	maxPrivatePhotoPixels    = 50_000_000
)

type validatedPrivatePhotoFile struct {
	ports.UploadedFile
	filename    string
	contentType string
}

func (f validatedPrivatePhotoFile) Filename() string    { return f.filename }
func (f validatedPrivatePhotoFile) ContentType() string { return f.contentType }

func NewPrivatePhotoRepository(db *gorm.DB, snowFlakeNode *helpers.Node, mediaRepo *MediaRepository) *PrivatePhotoRepository {
	return &PrivatePhotoRepository{db: db, snowFlakeNode: snowFlakeNode, mediaRepo: mediaRepo}
}

func (r *PrivatePhotoRepository) FindPrivatePhotoUserByPublicID(ctx context.Context, publicID int64) (ports.PrivatePhotoUser, error) {
	var user models.User
	result := r.db.WithContext(ctx).
		Preload("Avatar.File").
		Where("public_id = ? AND deleted_at IS NULL", publicID).
		Where("user_role NOT IN ?", []string{"banned", "deleted"}).
		Take(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return ports.PrivatePhotoUser{}, ports.ErrNotFound
	}
	if result.Error != nil {
		return ports.PrivatePhotoUser{}, result.Error
	}
	return privatePhotoUser(user), nil
}

func (r *PrivatePhotoRepository) FindPrivatePhotoUserByID(ctx context.Context, userID uuid.UUID) (ports.PrivatePhotoUser, error) {
	if userID == uuid.Nil {
		return ports.PrivatePhotoUser{}, ports.ErrNotFound
	}
	var user models.User
	result := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", userID).
		Where("user_role NOT IN ?", []string{"banned", "deleted"}).
		Take(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return ports.PrivatePhotoUser{}, ports.ErrNotFound
	}
	if result.Error != nil {
		return ports.PrivatePhotoUser{}, result.Error
	}
	return privatePhotoUser(user), nil
}

func (r *PrivatePhotoRepository) CountPrivatePhotos(ctx context.Context, ownerID uuid.UUID) (int64, error) {
	var count int64
	err := privatePhotoMediaBaseScope(r.db.WithContext(ctx), ownerID).
		Where("processing_status <> ?", modelmedia.ProcessingStatusFailed).
		Count(&count).Error
	return count, err
}

func (r *PrivatePhotoRepository) ListPrivatePhotos(ctx context.Context, ownerID uuid.UUID) ([]modelmedia.Media, error) {
	photos := make([]modelmedia.Media, 0)
	err := privatePhotoMediaBaseScope(r.db.WithContext(ctx), ownerID).
		Preload("File").
		Order("public_id DESC").
		Find(&photos).Error
	return photos, err
}

func (r *PrivatePhotoRepository) ListProfilePhotos(ctx context.Context, ownerID uuid.UUID) ([]modelmedia.Media, error) {
	photos := make([]modelmedia.Media, 0)
	err := r.db.WithContext(ctx).
		Model(&modelmedia.Media{}).
		Where("owner_id = ? AND user_id = ?", ownerID, ownerID).
		Where("owner_type = ? AND role = ? AND is_public = ?", modelmedia.OwnerUser, modelmedia.RoleProfile, true).
		Where("processing_status <> ?", modelmedia.ProcessingStatusFailed).
		Preload("File").
		Order("public_id DESC").
		Find(&photos).Error
	return photos, err
}

func (r *PrivatePhotoRepository) MoveProfilePhoto(
	ctx context.Context,
	ownerID uuid.UUID,
	photoPublicID int64,
	destination modelmedia.MediaRole,
	maxPrivateCount int64,
) (moved *modelmedia.Media, retErr error) {
	if destination != modelmedia.RoleProfile && destination != modelmedia.RolePrivatePhoto {
		return nil, errors.New("unsupported profile photo destination")
	}

	retErr = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if destination == modelmedia.RolePrivatePhoto {
			if err := lockPrivatePhotoAlbum(tx, ownerID); err != nil {
				return err
			}
			if maxPrivateCount > 0 {
				var count int64
				if err := privatePhotoMediaBaseScope(tx, ownerID).
					Where("processing_status <> ?", modelmedia.ProcessingStatusFailed).
					Count(&count).Error; err != nil {
					return err
				}
				if count >= maxPrivateCount {
					return ports.ErrPrivatePhotoLimitReached
				}
			}
		}

		var photo modelmedia.Media
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("File").
			Where("public_id = ?", photoPublicID).
			Where("owner_id = ? AND user_id = ? AND owner_type = ?", ownerID, ownerID, modelmedia.OwnerUser).
			Where("role IN ?", []modelmedia.MediaRole{modelmedia.RoleProfile, modelmedia.RolePrivatePhoto}).
			Take(&photo)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return ports.ErrNotFound
		}
		if result.Error != nil {
			return result.Error
		}
		if photo.Role == destination {
			moved = &photo
			return nil
		}

		isPublic := destination == modelmedia.RoleProfile
		if err := tx.Model(&modelmedia.Media{}).
			Where("id = ?", photo.ID).
			Updates(map[string]any{"role": destination, "is_public": isPublic}).Error; err != nil {
			return err
		}
		photo.Role = destination
		photo.IsPublic = isPublic
		moved = &photo
		return nil
	})
	return moved, retErr
}

func (r *PrivatePhotoRepository) AddPrivatePhoto(ctx context.Context, ownerID uuid.UUID, file ports.UploadedFile, maxCount int64) (created *modelmedia.Media, retErr error) {
	if r.mediaRepo == nil {
		return nil, errors.New("private photo media repository is not configured")
	}
	if file == nil {
		return nil, domainmedia.ErrPrivatePhotoImageRequired
	}
	source, err := file.Open()
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(source)
	header, readErr := reader.Peek(512)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		_ = source.Close()
		return nil, readErr
	}
	if !domainmedia.IsPrivatePhotoImageHeader(header) {
		_ = source.Close()
		return nil, domainmedia.ErrPrivatePhotoImageRequired
	}
	config, format, decodeErr := image.DecodeConfig(reader)
	closeErr := source.Close()
	if decodeErr != nil {
		return nil, domainmedia.ErrPrivatePhotoImageRequired
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if config.Width <= 0 || config.Height <= 0 ||
		config.Width > maxPrivatePhotoDimension || config.Height > maxPrivatePhotoDimension ||
		int64(config.Width)*int64(config.Height) > maxPrivatePhotoPixels {
		return nil, domainmedia.ErrPrivatePhotoImageDimensions
	}

	canonicalMIME, canonicalExtension, ok := privatePhotoFormat(format)
	if !ok {
		return nil, domainmedia.ErrPrivatePhotoImageRequired
	}
	if declared := strings.TrimSpace(file.ContentType()); declared != "" {
		declaredMIME, _, parseErr := mime.ParseMediaType(declared)
		if parseErr != nil {
			return nil, domainmedia.ErrPrivatePhotoImageRequired
		}
		if strings.EqualFold(declaredMIME, "image/jpg") {
			declaredMIME = "image/jpeg"
		}
		if !strings.EqualFold(declaredMIME, canonicalMIME) {
			return nil, domainmedia.ErrPrivatePhotoImageRequired
		}
	}
	file = validatedPrivatePhotoFile{
		UploadedFile: file,
		filename:     "private-photo" + canonicalExtension,
		contentType:  canonicalMIME,
	}
	retErr = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPrivatePhotoAlbum(tx, ownerID); err != nil {
			return err
		}
		if maxCount > 0 {
			var count int64
			if err := privatePhotoMediaBaseScope(tx, ownerID).Count(&count).Error; err != nil {
				return err
			}
			if count >= maxCount {
				return ports.ErrPrivatePhotoLimitReached
			}
		}

		scopedMediaRepository := *r.mediaRepo
		scopedMediaRepository.db = tx.WithContext(ctx)
		var err error
		created, err = scopedMediaRepository.AddMedia(ownerID, modelmedia.OwnerUser, ownerID, modelmedia.RolePrivatePhoto, file)
		if err != nil {
			return err
		}
		// Keep the invariant check inside the outer transaction. If a media
		// adapter ever persists the wrong visibility, the database row rolls
		// back together with the album operation instead of leaving a public
		// orphan behind.
		if created == nil || created.IsPublic {
			return errors.New("private photo was not stored as protected media")
		}
		return nil
	})
	if retErr != nil {
		if created != nil {
			retErr = errors.Join(retErr, cleanupStoredUploads(privatePhotoStoredPaths(created.File)))
		}
		return nil, retErr
	}
	return created, nil
}

func privatePhotoFormat(format string) (mimeType, extension string, ok bool) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "jpeg":
		return "image/jpeg", ".jpg", true
	case "png":
		return "image/png", ".png", true
	case "gif":
		return "image/gif", ".gif", true
	case "webp":
		return "image/webp", ".webp", true
	case "bmp":
		return "image/bmp", ".bmp", true
	case "tiff":
		return "image/tiff", ".tiff", true
	default:
		return "", "", false
	}
}

func (r *PrivatePhotoRepository) DeletePrivatePhoto(ctx context.Context, ownerID uuid.UUID, photoPublicID int64) error {
	var photo modelmedia.Media
	result := r.db.WithContext(ctx).
		Preload("File").
		Where("public_id = ?", photoPublicID).
		Where("owner_id = ? AND user_id = ?", ownerID, ownerID).
		Where("owner_type = ? AND role = ?", modelmedia.OwnerUser, modelmedia.RolePrivatePhoto).
		Take(&photo)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return ports.ErrNotFound
	}
	if result.Error != nil {
		return result.Error
	}

	paths := privatePhotoStoredPaths(photo.File)
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&modelmedia.Media{}, "id = ?", photo.ID).Error; err != nil {
			return err
		}
		return tx.Delete(&modelutils.FileMetadata{}, "id = ?", photo.FileID).Error
	}); err != nil {
		return err
	}

	if err := cleanupStoredUploads(paths); err != nil {
		// The logical delete is already committed and access has been removed.
		// Report success to keep delete idempotent; orphan cleanup can be
		// retried operationally without resurrecting the media record.
		log.Printf("private photo %d stored file cleanup failed: %v", photoPublicID, err)
	}
	return nil
}

func (r *PrivatePhotoRepository) ArePrivatePhotoUsersBlocked(ctx context.Context, firstUserID, secondUserID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.EngagementDetail{}).
		Where("kind = ?", models.EngagementKindBlocking).
		Where("(engager_id = ? AND engagee_id = ?) OR (engager_id = ? AND engagee_id = ?)", firstUserID, secondUserID, secondUserID, firstUserID).
		Count(&count).Error
	return count > 0, err
}

func (r *PrivatePhotoRepository) GetPrivatePhotoAccess(ctx context.Context, ownerID, viewerID uuid.UUID) (*ports.PrivatePhotoAccessRecord, error) {
	var request models.PrivatePhotoAccessRequest
	result := r.db.WithContext(ctx).
		Where("owner_id = ? AND viewer_id = ?", ownerID, viewerID).
		Take(&request)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ports.ErrNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return r.privatePhotoAccessRecord(ctx, request)
}

func (r *PrivatePhotoRepository) RequestPrivatePhotoAccess(ctx context.Context, ownerID, viewerID uuid.UUID, now time.Time) (*ports.PrivatePhotoAccessRecord, bool, error) {
	if r.snowFlakeNode == nil {
		return nil, false, errors.New("private photo public ID generator is not configured")
	}

	var request models.PrivatePhotoAccessRequest
	changed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPrivatePhotoAccessPair(tx, ownerID, viewerID); err != nil {
			return err
		}

		query := privatePhotoLockingQuery(tx).
			Where("owner_id = ? AND viewer_id = ?", ownerID, viewerID).
			Take(&request)
		if errors.Is(query.Error, gorm.ErrRecordNotFound) {
			request = models.PrivatePhotoAccessRequest{
				ID:          uuid.New(),
				PublicID:    r.snowFlakeNode.Generate().Int64(),
				OwnerID:     ownerID,
				ViewerID:    viewerID,
				Status:      domainmedia.PrivatePhotoAccessPending,
				RequestedAt: now,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			changed = true
			return tx.Create(&request).Error
		}
		if query.Error != nil {
			return query.Error
		}

		next, requestChanged := domainmedia.RequestPrivatePhotoAccess(&request.Status)
		changed = requestChanged
		if !changed {
			return nil
		}
		request.Status = next
		request.RequestedAt = now
		request.RespondedAt = nil
		request.UpdatedAt = now
		return tx.Model(&models.PrivatePhotoAccessRequest{}).
			Where("id = ?", request.ID).
			Updates(map[string]any{
				"status":       request.Status,
				"requested_at": request.RequestedAt,
				"responded_at": nil,
				"updated_at":   request.UpdatedAt,
			}).Error
	})
	if err != nil {
		return nil, false, err
	}
	record, err := r.privatePhotoAccessRecord(ctx, request)
	return record, changed, err
}

func (r *PrivatePhotoRepository) FindPrivatePhotoAccessByPublicID(ctx context.Context, requestPublicID int64) (*ports.PrivatePhotoAccessRecord, error) {
	var request models.PrivatePhotoAccessRequest
	result := r.db.WithContext(ctx).Where("public_id = ?", requestPublicID).Take(&request)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ports.ErrNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return r.privatePhotoAccessRecord(ctx, request)
}

func (r *PrivatePhotoRepository) RespondPrivatePhotoAccess(ctx context.Context, requestPublicID int64, ownerID uuid.UUID, decision domainmedia.PrivatePhotoAccessStatus, now time.Time) (*ports.PrivatePhotoAccessRecord, bool, error) {
	var request models.PrivatePhotoAccessRequest
	changed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := privatePhotoLockingQuery(tx).
			Where("public_id = ? AND owner_id = ?", requestPublicID, ownerID).
			Take(&request)
		if errors.Is(query.Error, gorm.ErrRecordNotFound) {
			return ports.ErrNotFound
		}
		if query.Error != nil {
			return query.Error
		}

		next, responseChanged, err := domainmedia.RespondToPrivatePhotoAccess(request.Status, decision)
		if err != nil {
			return err
		}
		changed = responseChanged
		if !changed {
			return nil
		}
		request.Status = next
		request.RespondedAt = &now
		request.UpdatedAt = now
		return tx.Model(&models.PrivatePhotoAccessRequest{}).
			Where("id = ?", request.ID).
			Updates(map[string]any{
				"status":       request.Status,
				"responded_at": request.RespondedAt,
				"updated_at":   request.UpdatedAt,
			}).Error
	})
	if err != nil {
		return nil, false, err
	}
	record, err := r.privatePhotoAccessRecord(ctx, request)
	return record, changed, err
}

func (r *PrivatePhotoRepository) RevokePrivatePhotoAccess(ctx context.Context, ownerID, viewerID uuid.UUID, now time.Time) (*ports.PrivatePhotoAccessRecord, bool, error) {
	var request models.PrivatePhotoAccessRequest
	changed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := privatePhotoLockingQuery(tx).
			Where("owner_id = ? AND viewer_id = ?", ownerID, viewerID).
			Take(&request)
		if errors.Is(query.Error, gorm.ErrRecordNotFound) {
			return ports.ErrNotFound
		}
		if query.Error != nil {
			return query.Error
		}

		next, revoked, err := domainmedia.RevokePrivatePhotoAccess(request.Status)
		if err != nil {
			return err
		}
		changed = revoked
		if !changed {
			return nil
		}
		request.Status = next
		request.RespondedAt = &now
		request.UpdatedAt = now
		return tx.Model(&models.PrivatePhotoAccessRequest{}).
			Where("id = ?", request.ID).
			Updates(map[string]any{
				"status":       request.Status,
				"responded_at": request.RespondedAt,
				"updated_at":   request.UpdatedAt,
			}).Error
	})
	if err != nil {
		return nil, false, err
	}
	record, err := r.privatePhotoAccessRecord(ctx, request)
	return record, changed, err
}

func (r *PrivatePhotoRepository) ListPrivatePhotoAccessRequests(ctx context.Context, ownerID uuid.UUID) ([]ports.PrivatePhotoAccessRecord, error) {
	var requests []models.PrivatePhotoAccessRequest
	if err := r.db.WithContext(ctx).
		Where("owner_id = ?", ownerID).
		Order("requested_at DESC, public_id DESC").
		Find(&requests).Error; err != nil {
		return nil, err
	}
	if len(requests) == 0 {
		return []ports.PrivatePhotoAccessRecord{}, nil
	}

	viewerIDs := make([]uuid.UUID, 0, len(requests))
	for _, request := range requests {
		viewerIDs = append(viewerIDs, request.ViewerID)
	}
	var viewers []models.User
	if err := r.db.WithContext(ctx).
		Select("id", "public_id", "user_name", "display_name", "avatar_id").
		Preload("Avatar.File").
		Where("id IN ?", viewerIDs).
		Find(&viewers).Error; err != nil {
		return nil, err
	}
	viewerMap := make(map[uuid.UUID]ports.PrivatePhotoUser, len(viewers))
	for _, viewer := range viewers {
		viewerMap[viewer.ID] = privatePhotoUser(viewer)
	}

	var owner models.User
	if err := r.db.WithContext(ctx).Select("id", "public_id").Where("id = ?", ownerID).Take(&owner).Error; err != nil {
		return nil, err
	}

	result := make([]ports.PrivatePhotoAccessRecord, 0, len(requests))
	for _, request := range requests {
		viewer, exists := viewerMap[request.ViewerID]
		if !exists {
			// The access row can briefly outlive a viewer in installations that
			// predate the current foreign-key migration. Do not turn one stale
			// historical row into a 500 for the album owner.
			continue
		}
		result = append(result, privatePhotoAccessRecord(request, owner.PublicID, viewer))
	}
	return result, nil
}

func (r *PrivatePhotoRepository) HasApprovedPrivatePhotoAccess(ctx context.Context, ownerID, viewerID uuid.UUID) (bool, error) {
	blocked, err := r.ArePrivatePhotoUsersBlocked(ctx, ownerID, viewerID)
	if err != nil || blocked {
		return false, err
	}
	var count int64
	err = r.db.WithContext(ctx).
		Model(&models.PrivatePhotoAccessRequest{}).
		Where("owner_id = ? AND viewer_id = ? AND status = ?", ownerID, viewerID, domainmedia.PrivatePhotoAccessApproved).
		Count(&count).Error
	return count > 0, err
}

func (r *PrivatePhotoRepository) RevokePrivatePhotoAccessBetween(ctx context.Context, firstUserID, secondUserID uuid.UUID, now time.Time) error {
	if firstUserID == uuid.Nil || secondUserID == uuid.Nil || firstUserID == secondUserID {
		return errors.New("two different private photo users are required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var requests []models.PrivatePhotoAccessRequest
		query := privatePhotoLockingQuery(tx).
			Where(
				"(owner_id = ? AND viewer_id = ?) OR (owner_id = ? AND viewer_id = ?)",
				firstUserID,
				secondUserID,
				secondUserID,
				firstUserID,
			).
			Where("status <> ?", domainmedia.PrivatePhotoAccessDenied).
			Find(&requests)
		if query.Error != nil || len(requests) == 0 {
			return query.Error
		}

		requestIDs := make([]uuid.UUID, 0, len(requests))
		for index := range requests {
			requestIDs = append(requestIDs, requests[index].ID)
			requests[index].Status = domainmedia.PrivatePhotoAccessDenied
			requests[index].RespondedAt = &now
		}
		if err := tx.Model(&models.PrivatePhotoAccessRequest{}).
			Where("id IN ?", requestIDs).
			Updates(map[string]any{
				"status":       domainmedia.PrivatePhotoAccessDenied,
				"responded_at": now,
				"updated_at":   now,
			}).Error; err != nil {
			return err
		}
		return resolveBlockedPrivatePhotoNotifications(tx, requests, now)
	})
}

func resolveBlockedPrivatePhotoNotifications(tx *gorm.DB, requests []models.PrivatePhotoAccessRequest, now time.Time) error {
	requestByPublicID := make(map[string]models.PrivatePhotoAccessRequest, len(requests))
	ownerIDs := make([]uuid.UUID, 0, len(requests))
	for _, request := range requests {
		requestByPublicID[fmt.Sprint(request.PublicID)] = request
		ownerIDs = append(ownerIDs, request.OwnerID)
	}

	var items []notifications.Notification
	if err := tx.
		Where("user_id IN ? AND type = ?", ownerIDs, notifications.NotificationTypePrivatePhotoAccessRequest).
		Order("created_at DESC").
		Find(&items).Error; err != nil {
		return err
	}
	resolved := make(map[string]bool, len(requests))
	for _, item := range items {
		data, ok := item.Payload.Data.(map[string]any)
		if !ok {
			continue
		}
		requestID := fmt.Sprint(data["request_id"])
		request, ok := requestByPublicID[requestID]
		if !ok || request.OwnerID != item.UserID || resolved[requestID] {
			continue
		}
		status := fmt.Sprint(data["access_status"])
		if status == "<nil>" || status == "" {
			status = fmt.Sprint(data["status"])
		}
		if status != string(domainmedia.PrivatePhotoAccessPending) && status != string(domainmedia.PrivatePhotoAccessApproved) {
			continue
		}
		data["status"] = domainmedia.PrivatePhotoAccessDenied
		data["access_status"] = domainmedia.PrivatePhotoAccessDenied
		payload := item.Payload
		payload.Data = data
		if err := tx.Model(&notifications.Notification{}).
			Where("id = ?", item.ID).
			Updates(map[string]any{
				"payload": payload,
				"is_read": true,
				"read_at": now,
			}).Error; err != nil {
			return err
		}
		resolved[requestID] = true
	}
	return nil
}

func privatePhotoMediaBaseScope(db *gorm.DB, ownerID uuid.UUID) *gorm.DB {
	return db.Model(&modelmedia.Media{}).
		Where("owner_id = ? AND user_id = ?", ownerID, ownerID).
		Where("owner_type = ? AND role = ?", modelmedia.OwnerUser, modelmedia.RolePrivatePhoto).
		Where("is_public = FALSE")
}

func privatePhotoLockingQuery(db *gorm.DB) *gorm.DB {
	if db != nil && db.Dialector != nil && db.Name() == "postgres" {
		return db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return db
}

func lockPrivatePhotoAccessPair(db *gorm.DB, ownerID, viewerID uuid.UUID) error {
	if db == nil || db.Dialector == nil || db.Name() != "postgres" {
		return nil
	}
	key := ownerID.String() + ":" + viewerID.String()
	return db.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", key).Error
}

func lockPrivatePhotoAlbum(db *gorm.DB, ownerID uuid.UUID) error {
	if db == nil || db.Dialector == nil || db.Name() != "postgres" {
		return nil
	}
	return db.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "private-photos:"+ownerID.String()).Error
}

func (r *PrivatePhotoRepository) privatePhotoAccessRecord(ctx context.Context, request models.PrivatePhotoAccessRequest) (*ports.PrivatePhotoAccessRecord, error) {
	var users []models.User
	if err := r.db.WithContext(ctx).
		Preload("Avatar.File").
		Where("id IN ?", []uuid.UUID{request.OwnerID, request.ViewerID}).
		Find(&users).Error; err != nil {
		return nil, err
	}
	var ownerPublicID int64
	var viewer ports.PrivatePhotoUser
	for _, user := range users {
		switch user.ID {
		case request.OwnerID:
			ownerPublicID = user.PublicID
		case request.ViewerID:
			viewer = privatePhotoUser(user)
		}
	}
	if ownerPublicID == 0 || viewer.ID == uuid.Nil {
		return nil, ports.ErrNotFound
	}
	record := privatePhotoAccessRecord(request, ownerPublicID, viewer)
	return &record, nil
}

func privatePhotoAccessRecord(request models.PrivatePhotoAccessRequest, ownerPublicID int64, viewer ports.PrivatePhotoUser) ports.PrivatePhotoAccessRecord {
	return ports.PrivatePhotoAccessRecord{
		ID:             request.ID,
		PublicID:       request.PublicID,
		OwnerID:        request.OwnerID,
		OwnerPublicID:  ownerPublicID,
		ViewerID:       request.ViewerID,
		ViewerPublicID: viewer.PublicID,
		Status:         request.Status,
		RequestedAt:    request.RequestedAt,
		RespondedAt:    request.RespondedAt,
		Viewer:         viewer,
	}
}

func privatePhotoUser(user models.User) ports.PrivatePhotoUser {
	return ports.PrivatePhotoUser{
		ID:          user.ID,
		PublicID:    user.PublicID,
		UserName:    user.UserName,
		DisplayName: user.DisplayName,
		Avatar:      user.Avatar,
	}
}

func privatePhotoStoredPaths(file modelutils.FileMetadata) []string {
	paths := []string{file.StoragePath}
	if file.Variants == nil {
		return paths
	}
	appendVariant := func(url string) {
		url = strings.TrimSpace(url)
		if url == "" {
			return
		}
		if strings.HasPrefix(url, "/") {
			url = "." + url
		}
		paths = append(paths, url)
	}
	if image := file.Variants.Image; image != nil {
		for _, variant := range []*modelutils.VariantInfo{image.Icon, image.Thumbnail, image.Small, image.Medium, image.Large, image.Original} {
			if variant != nil {
				appendVariant(variant.URL)
			}
		}
	}
	if video := file.Variants.Video; video != nil {
		for _, variant := range []*modelutils.VariantInfo{video.Poster, video.Low, video.Medium, video.High, video.Preview} {
			if variant != nil {
				appendVariant(variant.URL)
			}
		}
	}
	return paths
}
