package repositories

import (
	"bufio"
	"context"
	"core/application/ports"
	"core/constants"
	domainmedia "core/domain/media"
	"core/helpers"
	"core/models"
	modelmedia "core/models/media"
	"core/models/notifications"
	modelutils "core/models/utils"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime"
	"strconv"
	"strings"
	"time"

	_ "github.com/chai2010/webp"
	"github.com/google/uuid"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// privatePhotoAccessDetails is the JSON payload carried in
// EngagementDetail.Details for the single row representing one
// (owner, viewer) direction. The status and distinct-users invariants this
// used to get from chk_private_photo_access_status and
// chk_private_photo_access_distinct_users are enforced by raw SQL CHECK
// constraints on this JSON path (see EnsurePrivatePhotoAccessConstraints)
// since GORM cannot express them through tags.
type privatePhotoAccessDetails struct {
	Status      domainmedia.PrivatePhotoAccessStatus `json:"status"`
	PublicID    int64                                `json:"public_id,string"`
	RequestedAt time.Time                            `json:"requested_at"`
	RespondedAt *time.Time                           `json:"responded_at,omitempty"`
}

func decodePrivatePhotoAccessDetails(raw datatypes.JSON) (privatePhotoAccessDetails, error) {
	var details privatePhotoAccessDetails
	if len(raw) == 0 {
		return details, nil
	}
	err := json.Unmarshal(raw, &details)
	return details, err
}

// privatePhotoAccessDedupeKey is fixed-role, not canonically ordered: an
// owner can independently grant a viewer access while the viewer separately
// owns no reciprocal grant, so (owner, viewer) and (viewer, owner) are two
// distinct rows, each unique via this key's existing index — the same
// guarantee the dedicated table used to get from
// uidx_private_photo_access_owner_viewer.
func privatePhotoAccessDedupeKey(ownerID, viewerID uuid.UUID) string {
	return "private_photo_access:" + ownerID.String() + ":" + viewerID.String()
}

func findPrivatePhotoAccessDetail(tx *gorm.DB, ownerID, viewerID uuid.UUID, lock bool) (*models.EngagementDetail, error) {
	query := tx.Where("dedupe_key = ?", privatePhotoAccessDedupeKey(ownerID, viewerID))
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var row models.EngagementDetail
	err := query.Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func createPrivatePhotoAccessDetail(tx *gorm.DB, ownerID, viewerID uuid.UUID, details privatePhotoAccessDetails, now time.Time) (models.EngagementDetail, error) {
	payload, err := json.Marshal(details)
	if err != nil {
		return models.EngagementDetail{}, err
	}
	aggregate, err := loadOrCreateEngagementAggregate(tx, ownerID, models.EngagementContentableTypeUser)
	if err != nil {
		return models.EngagementDetail{}, err
	}
	dedupeKey := privatePhotoAccessDedupeKey(ownerID, viewerID)
	row := models.EngagementDetail{
		ID: uuid.New(), EngagementID: aggregate.ID, DedupeKey: &dedupeKey,
		EngagerID: ownerID, EngageeID: viewerID, Kind: models.EngagementKindPrivatePhotoAccess,
		Details: datatypes.JSON(payload), CreatedAt: now, UpdatedAt: now,
	}
	if err := tx.Create(&row).Error; err != nil {
		return models.EngagementDetail{}, err
	}
	return row, nil
}

func updatePrivatePhotoAccessDetail(tx *gorm.DB, id uuid.UUID, details privatePhotoAccessDetails, now time.Time) (datatypes.JSON, error) {
	payload, err := json.Marshal(details)
	if err != nil {
		return nil, err
	}
	if err := tx.Model(&models.EngagementDetail{}).Where("id = ?", id).
		Updates(map[string]any{"details": datatypes.JSON(payload), "updated_at": now}).Error; err != nil {
		return nil, err
	}
	return datatypes.JSON(payload), nil
}

type PrivatePhotoRepository struct {
	db            *gorm.DB
	snowFlakeNode *helpers.Node
	mediaRepo     *MediaRepository
}

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

func (r *PrivatePhotoRepository) IsPrivatePhotoOwnerPremium(ctx context.Context, ownerID uuid.UUID) (bool, error) {
	var isPremium bool
	result := r.db.WithContext(ctx).
		Model(&models.User{}).
		Select("is_premium").
		Where("id = ? AND deleted_at IS NULL", ownerID).
		Where("user_role NOT IN ?", []string{"banned", "deleted"}).
		Scan(&isPremium)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		return false, ports.ErrNotFound
	}
	return isPremium, nil
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
		return nil, constants.ErrProfilePhotoDestinationUnsupported
	}

	retErr = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if destination == modelmedia.RolePrivatePhoto {
			if err := lockPrivatePhotoAlbum(tx, ownerID); err != nil {
				return err
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
		if destination == modelmedia.RolePrivatePhoto && maxPrivateCount > 0 && photo.ProcessingStatus != modelmedia.ProcessingStatusFailed {
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

func (r *PrivatePhotoRepository) AddPrivatePhoto(ctx context.Context, ownerID uuid.UUID, file *domainmedia.UploadedFile, maxCount int64) (created *modelmedia.Media, retErr error) {
	if r.mediaRepo == nil {
		return nil, constants.ErrPrivatePhotoMediaRepositoryNotConfigured
	}
	if file == nil || file.Open == nil {
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
	if !validImageDimensions(config, constants.PrivatePhotoMaxDimension, constants.PrivatePhotoMaxPixels) {
		return nil, domainmedia.ErrPrivatePhotoImageDimensions
	}

	canonicalMIME, canonicalExtension, ok := imageFormatMetadata(format)
	if !ok {
		return nil, domainmedia.ErrPrivatePhotoImageRequired
	}
	if declared := strings.TrimSpace(file.ContentType); declared != "" {
		declaredMIME, _, parseErr := mime.ParseMediaType(declared)
		if parseErr != nil {
			return nil, domainmedia.ErrPrivatePhotoImageRequired
		}
		if strings.EqualFold(declaredMIME, constants.MIMEImageJPG) {
			declaredMIME = constants.MIMEImageJPEG
		}
		if !strings.EqualFold(declaredMIME, canonicalMIME) {
			return nil, domainmedia.ErrPrivatePhotoImageRequired
		}
	}
	validatedFile := *file
	validatedFile.Filename = "private-photo" + canonicalExtension
	validatedFile.ContentType = canonicalMIME
	file = &validatedFile
	retErr = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPrivatePhotoAlbum(tx, ownerID); err != nil {
			return err
		}
		if maxCount > 0 {
			var count int64
			if err := privatePhotoMediaBaseScope(tx, ownerID).
				Where("processing_status <> ?", modelmedia.ProcessingStatusFailed).
				Count(&count).Error; err != nil {
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
			return constants.ErrPrivatePhotoProtectionRequired
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
	row, err := findPrivatePhotoAccessDetail(r.db.WithContext(ctx), ownerID, viewerID, false)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ports.ErrNotFound
	}
	return r.privatePhotoAccessRecord(ctx, *row)
}

func (r *PrivatePhotoRepository) GrantPrivatePhotoAccess(ctx context.Context, ownerID, viewerID uuid.UUID, now time.Time) (*ports.PrivatePhotoAccessRecord, bool, error) {
	if r.snowFlakeNode == nil {
		return nil, false, constants.ErrPrivatePhotoPublicIDNotConfigured
	}
	if ownerID == viewerID {
		return nil, false, ports.ErrPrivatePhotoAccessBlocked
	}
	var row models.EngagementDetail
	var details privatePhotoAccessDetails
	changed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPrivatePhotoAccessPair(tx, ownerID, viewerID); err != nil {
			return err
		}
		if err := ensurePrivatePhotoUsersNotBlocked(tx, ownerID, viewerID); err != nil {
			return err
		}
		existing, err := findPrivatePhotoAccessDetail(tx, ownerID, viewerID, true)
		if err != nil {
			return err
		}
		if existing != nil {
			details, err = decodePrivatePhotoAccessDetails(existing.Details)
			if err != nil {
				return err
			}
			if details.Status == domainmedia.PrivatePhotoAccessApproved {
				row = *existing
				return nil
			}
		} else {
			details.PublicID, details.RequestedAt = r.snowFlakeNode.Generate().Int64(), now
		}
		details.Status, details.RespondedAt = domainmedia.PrivatePhotoAccessApproved, &now
		changed = true
		if existing != nil {
			payload, err := updatePrivatePhotoAccessDetail(tx, existing.ID, details, now)
			if err != nil {
				return err
			}
			row = *existing
			row.Details, row.UpdatedAt = payload, now
		} else {
			row, err = createPrivatePhotoAccessDetail(tx, ownerID, viewerID, details, now)
			if err != nil {
				return err
			}
		}
		return enqueuePrivatePhotoNotification(tx, row, details, true)
	})
	if err != nil {
		return nil, false, err
	}
	record, err := r.privatePhotoAccessRecord(ctx, row)
	return record, changed, err
}

func (r *PrivatePhotoRepository) RequestPrivatePhotoAccess(ctx context.Context, ownerID, viewerID uuid.UUID, now time.Time) (*ports.PrivatePhotoAccessRecord, bool, error) {
	if r.snowFlakeNode == nil {
		return nil, false, constants.ErrPrivatePhotoPublicIDNotConfigured
	}

	var row models.EngagementDetail
	var details privatePhotoAccessDetails
	changed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPrivatePhotoAccessPair(tx, ownerID, viewerID); err != nil {
			return err
		}
		if err := ensurePrivatePhotoUsersNotBlocked(tx, ownerID, viewerID); err != nil {
			return err
		}

		existing, err := findPrivatePhotoAccessDetail(tx, ownerID, viewerID, true)
		if err != nil {
			return err
		}
		if existing == nil {
			details = privatePhotoAccessDetails{Status: domainmedia.PrivatePhotoAccessPending, PublicID: r.snowFlakeNode.Generate().Int64(), RequestedAt: now}
			changed = true
			row, err = createPrivatePhotoAccessDetail(tx, ownerID, viewerID, details, now)
			if err != nil {
				return err
			}
			return enqueuePrivatePhotoNotification(tx, row, details, false)
		}

		details, err = decodePrivatePhotoAccessDetails(existing.Details)
		if err != nil {
			return err
		}
		next, requestChanged := domainmedia.RequestPrivatePhotoAccess(&details.Status)
		changed = requestChanged
		if !changed {
			row = *existing
			return nil
		}
		details.Status, details.RequestedAt, details.RespondedAt = next, now, nil
		payload, err := updatePrivatePhotoAccessDetail(tx, existing.ID, details, now)
		if err != nil {
			return err
		}
		row = *existing
		row.Details, row.UpdatedAt = payload, now
		return enqueuePrivatePhotoNotification(tx, row, details, false)
	})
	if err != nil {
		return nil, false, err
	}
	record, err := r.privatePhotoAccessRecord(ctx, row)
	return record, changed, err
}

func (r *PrivatePhotoRepository) FindPrivatePhotoAccessByPublicID(ctx context.Context, requestPublicID int64) (*ports.PrivatePhotoAccessRecord, error) {
	var row models.EngagementDetail
	result := r.db.WithContext(ctx).
		Where("kind = ? AND details->>'public_id' = ?", models.EngagementKindPrivatePhotoAccess, strconv.FormatInt(requestPublicID, 10)).
		Take(&row)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ports.ErrNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return r.privatePhotoAccessRecord(ctx, row)
}

func (r *PrivatePhotoRepository) RespondPrivatePhotoAccess(ctx context.Context, requestPublicID int64, ownerID uuid.UUID, decision domainmedia.PrivatePhotoAccessStatus, now time.Time) (*ports.PrivatePhotoAccessRecord, bool, error) {
	var row models.EngagementDetail
	var details privatePhotoAccessDetails
	changed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Read the immutable pair first, then lock it before the request row.
		// Blocks, new requests and decisions all use this same lock order.
		if err := tx.Select("engager_id", "engagee_id").
			Where("kind = ? AND details->>'public_id' = ? AND engager_id = ?", models.EngagementKindPrivatePhotoAccess, strconv.FormatInt(requestPublicID, 10), ownerID).
			Take(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ports.ErrNotFound
			}
			return err
		}
		viewerID := row.EngageeID
		if err := lockPrivatePhotoAccessPair(tx, ownerID, viewerID); err != nil {
			return err
		}
		if decision == domainmedia.PrivatePhotoAccessApproved {
			if err := ensurePrivatePhotoUsersNotBlocked(tx, ownerID, viewerID); err != nil {
				return err
			}
		}
		existing, err := findPrivatePhotoAccessDetail(tx, ownerID, viewerID, true)
		if err != nil {
			return err
		}
		if existing == nil {
			return ports.ErrNotFound
		}
		details, err = decodePrivatePhotoAccessDetails(existing.Details)
		if err != nil {
			return err
		}
		if details.PublicID != requestPublicID {
			return ports.ErrNotFound
		}

		next, responseChanged, err := domainmedia.RespondToPrivatePhotoAccess(details.Status, decision)
		if err != nil {
			return err
		}
		changed = responseChanged
		if !changed {
			row = *existing
			return nil
		}
		details.Status, details.RespondedAt = next, &now
		payload, err := updatePrivatePhotoAccessDetail(tx, existing.ID, details, now)
		if err != nil {
			return err
		}
		row = *existing
		row.Details, row.UpdatedAt = payload, now
		return enqueuePrivatePhotoNotification(tx, row, details, true)
	})
	if err != nil {
		return nil, false, err
	}
	record, err := r.privatePhotoAccessRecord(ctx, row)
	return record, changed, err
}

func (r *PrivatePhotoRepository) RevokePrivatePhotoAccess(ctx context.Context, ownerID, viewerID uuid.UUID, now time.Time) (*ports.PrivatePhotoAccessRecord, bool, error) {
	var row models.EngagementDetail
	var details privatePhotoAccessDetails
	changed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPrivatePhotoAccessPair(tx, ownerID, viewerID); err != nil {
			return err
		}
		existing, err := findPrivatePhotoAccessDetail(tx, ownerID, viewerID, true)
		if err != nil {
			return err
		}
		if existing == nil {
			return ports.ErrNotFound
		}
		details, err = decodePrivatePhotoAccessDetails(existing.Details)
		if err != nil {
			return err
		}

		next, revoked, err := domainmedia.RevokePrivatePhotoAccess(details.Status)
		if err != nil {
			return err
		}
		changed = revoked
		if !changed {
			row = *existing
			return nil
		}
		details.Status, details.RespondedAt = next, &now
		payload, err := updatePrivatePhotoAccessDetail(tx, existing.ID, details, now)
		if err != nil {
			return err
		}
		row = *existing
		row.Details, row.UpdatedAt = payload, now
		return enqueuePrivatePhotoNotification(tx, row, details, true)
	})
	if err != nil {
		return nil, false, err
	}
	record, err := r.privatePhotoAccessRecord(ctx, row)
	return record, changed, err
}

func (r *PrivatePhotoRepository) ListPrivatePhotoAccessRequests(ctx context.Context, ownerID uuid.UUID) ([]ports.PrivatePhotoAccessRecord, error) {
	var rows []models.EngagementDetail
	if err := r.db.WithContext(ctx).
		Where("kind = ? AND engager_id = ?", models.EngagementKindPrivatePhotoAccess, ownerID).
		Order("(details->>'requested_at')::timestamptz DESC, (details->>'public_id')::bigint DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []ports.PrivatePhotoAccessRecord{}, nil
	}

	viewerIDs := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		viewerIDs = append(viewerIDs, row.EngageeID)
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

	result := make([]ports.PrivatePhotoAccessRecord, 0, len(rows))
	for _, row := range rows {
		viewer, exists := viewerMap[row.EngageeID]
		if !exists {
			// The access row can briefly outlive a viewer in installations that
			// predate the current foreign-key migration. Do not turn one stale
			// historical row into a 500 for the album owner.
			continue
		}
		details, err := decodePrivatePhotoAccessDetails(row.Details)
		if err != nil {
			return nil, err
		}
		result = append(result, privatePhotoAccessRecord(row, details, owner.PublicID, viewer))
	}
	return result, nil
}

func (r *PrivatePhotoRepository) HasApprovedPrivatePhotoAccess(ctx context.Context, ownerID, viewerID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.EngagementDetail{}).
		Joins("JOIN users AS owner ON owner.id = engagement_details.engager_id").
		Joins("JOIN users AS viewer ON viewer.id = engagement_details.engagee_id").
		Where("owner.deleted_at IS NULL AND viewer.deleted_at IS NULL").
		Where("owner.user_role NOT IN ? AND viewer.user_role NOT IN ?", []string{"banned", "deleted"}, []string{"banned", "deleted"}).
		Where("engagement_details.kind = ? AND engager_id = ? AND engagee_id = ? AND details->>'status' = ?",
			models.EngagementKindPrivatePhotoAccess, ownerID, viewerID, string(domainmedia.PrivatePhotoAccessApproved)).
		Where("? NOT IN (?)", ownerID, repositorySubquery(r.db).Model(&models.EngagementDetail{}).
			Select("engagee_id").Where("engagee_id IS NOT NULL").Where("kind = ? AND engager_id = ?", models.EngagementKindBlocking, viewerID)).
		Where("? NOT IN (?)", viewerID, repositorySubquery(r.db).Model(&models.EngagementDetail{}).
			Select("engagee_id").Where("engagee_id IS NOT NULL").Where("kind = ? AND engager_id = ?", models.EngagementKindBlocking, ownerID)).
		Count(&count).Error
	return count > 0, err
}

func (r *PrivatePhotoRepository) RevokePrivatePhotoAccessBetween(ctx context.Context, firstUserID, secondUserID uuid.UUID, now time.Time) error {
	if firstUserID == uuid.Nil || secondUserID == uuid.Nil || firstUserID == secondUserID {
		return constants.ErrPrivatePhotoDistinctUsersRequired
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPrivatePhotoAccessPair(tx, firstUserID, secondUserID); err != nil {
			return err
		}
		var rows []models.EngagementDetail
		query := privatePhotoLockingQuery(tx).
			Where(
				"kind = ? AND ((engager_id = ? AND engagee_id = ?) OR (engager_id = ? AND engagee_id = ?))",
				models.EngagementKindPrivatePhotoAccess, firstUserID, secondUserID, secondUserID, firstUserID,
			).
			Find(&rows)
		if query.Error != nil {
			return query.Error
		}

		revoked := make([]privatePhotoAccessRevocation, 0, len(rows))
		for i := range rows {
			details, err := decodePrivatePhotoAccessDetails(rows[i].Details)
			if err != nil {
				return err
			}
			if details.Status == domainmedia.PrivatePhotoAccessDenied {
				continue
			}
			details.Status, details.RespondedAt = domainmedia.PrivatePhotoAccessDenied, &now
			if _, err := updatePrivatePhotoAccessDetail(tx, rows[i].ID, details, now); err != nil {
				return err
			}
			revoked = append(revoked, privatePhotoAccessRevocation{ownerID: rows[i].EngagerID, publicID: details.PublicID})
		}
		if len(revoked) == 0 {
			return nil
		}
		return resolveBlockedPrivatePhotoNotifications(tx, revoked, now)
	})
}

// privatePhotoAccessRevocation is the minimal projection resolveBlockedPrivatePhotoNotifications
// needs for each row RevokePrivatePhotoAccessBetween just denied.
type privatePhotoAccessRevocation struct {
	ownerID  uuid.UUID
	publicID int64
}

func resolveBlockedPrivatePhotoNotifications(tx *gorm.DB, revoked []privatePhotoAccessRevocation, now time.Time) error {
	requestByPublicID := make(map[string]privatePhotoAccessRevocation, len(revoked))
	ownerIDs := make([]uuid.UUID, 0, len(revoked))
	for _, request := range revoked {
		requestByPublicID[fmt.Sprint(request.publicID)] = request
		ownerIDs = append(ownerIDs, request.ownerID)
	}

	var items []notifications.Notification
	if err := tx.
		Where("user_id IN ? AND type = ?", ownerIDs, notifications.NotificationTypePrivatePhotoAccessRequest).
		Order("created_at DESC").
		Find(&items).Error; err != nil {
		return err
	}
	resolved := make(map[string]bool, len(revoked))
	for _, item := range items {
		data, ok := item.Payload.Data.(map[string]any)
		if !ok {
			continue
		}
		requestID := fmt.Sprint(data["request_id"])
		request, ok := requestByPublicID[requestID]
		if !ok || request.ownerID != item.UserID || resolved[requestID] {
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
	first, second := ownerID.String(), viewerID.String()
	if first > second {
		first, second = second, first
	}
	key := "private-photo-access:" + first + ":" + second
	return lockTransactionResource(db, key).Error
}

func ensurePrivatePhotoUsersNotBlocked(db *gorm.DB, ownerID, viewerID uuid.UUID) error {
	blocked, err := (&PrivatePhotoRepository{db: db}).ArePrivatePhotoUsersBlocked(db.Statement.Context, ownerID, viewerID)
	if err != nil {
		return err
	}
	if blocked {
		return ports.ErrPrivatePhotoAccessBlocked
	}
	return nil
}

func lockPrivatePhotoAlbum(db *gorm.DB, ownerID uuid.UUID) error {
	if db == nil || db.Dialector == nil || db.Name() != "postgres" {
		return nil
	}
	return lockTransactionResource(db, "private-photos:"+ownerID.String()).Error
}

func (r *PrivatePhotoRepository) privatePhotoAccessRecord(ctx context.Context, row models.EngagementDetail) (*ports.PrivatePhotoAccessRecord, error) {
	details, err := decodePrivatePhotoAccessDetails(row.Details)
	if err != nil {
		return nil, err
	}
	var users []models.User
	if err := r.db.WithContext(ctx).
		Preload("Avatar.File").
		Where("id IN ?", []uuid.UUID{row.EngagerID, row.EngageeID}).
		Find(&users).Error; err != nil {
		return nil, err
	}
	var ownerPublicID int64
	var viewer ports.PrivatePhotoUser
	for _, user := range users {
		switch user.ID {
		case row.EngagerID:
			ownerPublicID = user.PublicID
		case row.EngageeID:
			viewer = privatePhotoUser(user)
		}
	}
	if ownerPublicID == 0 || viewer.ID == uuid.Nil {
		return nil, ports.ErrNotFound
	}
	record := privatePhotoAccessRecord(row, details, ownerPublicID, viewer)
	return &record, nil
}

func privatePhotoAccessRecord(row models.EngagementDetail, details privatePhotoAccessDetails, ownerPublicID int64, viewer ports.PrivatePhotoUser) ports.PrivatePhotoAccessRecord {
	return ports.PrivatePhotoAccessRecord{
		ID:             row.ID,
		PublicID:       details.PublicID,
		OwnerID:        row.EngagerID,
		OwnerPublicID:  ownerPublicID,
		ViewerID:       row.EngageeID,
		ViewerPublicID: viewer.PublicID,
		Status:         details.Status,
		RequestedAt:    details.RequestedAt,
		RespondedAt:    details.RespondedAt,
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
