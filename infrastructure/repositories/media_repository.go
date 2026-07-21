package repositories

import (
	"bytes"
	"context"
	"core/application/ports"
	domainmedia "core/domain/media"
	"core/helpers"
	"core/models/media"
	"core/models/utils"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/boxes-ltd/imaging"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MediaRepository struct {
	db            *gorm.DB
	snowFlakeNode *helpers.Node
}

func (r *MediaRepository) FindMediaFileAccess(ctx context.Context, storagePrefix string) (ports.MediaFileAccess, error) {
	var access ports.MediaFileAccess
	result := r.db.WithContext(ctx).
		Table("file_metadata AS f").
		Select(`
			f.storage_path,
			m.is_public,
			m.owner_id,
			m.owner_type,
			m.role,
			p.id AS post_id,
			p.author_id AS post_author_id,
			COALESCE(p.post_kind, '') AS post_kind,
			COALESCE(p.contentable_type, '') AS contentable_type,
			p.contentable_id AS chat_id,
			COALESCE(p.published, FALSE) AS published,
			p.audience
		`).
		Joins("JOIN medias m ON m.file_id = f.id").
		Joins("LEFT JOIN posts p ON p.id = m.owner_id AND p.deleted_at IS NULL").
		Where("f.storage_path = ? OR f.storage_path LIKE ?", storagePrefix, storagePrefix+".%").
		Limit(1).
		Scan(&access)
	if result.Error != nil {
		return ports.MediaFileAccess{}, result.Error
	}
	if result.RowsAffected == 0 {
		return ports.MediaFileAccess{}, ports.ErrNotFound
	}
	return access, nil
}

func (r *MediaRepository) FindMediaAccessPrincipal(ctx context.Context, publicID int64) (ports.MediaAccessPrincipal, error) {
	var principal ports.MediaAccessPrincipal
	result := r.db.WithContext(ctx).
		Table("users").
		Select("id, user_role AS role").
		Where("public_id = ? AND deleted_at IS NULL", publicID).
		Where("user_role NOT IN ?", []string{"banned", "deleted"}).
		Limit(1).
		Scan(&principal)
	if result.Error != nil {
		return ports.MediaAccessPrincipal{}, result.Error
	}
	if result.RowsAffected == 0 {
		return ports.MediaAccessPrincipal{}, ports.ErrNotFound
	}
	return principal, nil
}

func (r *MediaRepository) IsActiveChatParticipant(ctx context.Context, chatID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("chat_participants").
		Where("chat_id = ? AND user_id = ? AND left_at IS NULL", chatID, userID).
		Count(&count).Error
	return count > 0, err
}

func NewMediaRepository(db *gorm.DB, snowFlakeNode *helpers.Node) *MediaRepository {
	return &MediaRepository{db: db, snowFlakeNode: snowFlakeNode}
}

func (r *MediaRepository) Node() *helpers.Node {
	return r.snowFlakeNode
}

func (r *MediaRepository) GenerateStoragePath(userId uuid.UUID, ownerID uuid.UUID, ownerType media.OwnerType, role media.MediaRole, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	id := uuid.New().String()
	date := time.Now().Format("2006-01-02")
	baseDir := "./static/uploads"

	switch ownerType {
	case media.OwnerUser:
		switch role {
		case media.RoleProfile:
			return fmt.Sprintf("%s/users/%s/profile/%s/%s%s", baseDir, userId.String(), date, id, ext)
		case media.RoleCover:
			return fmt.Sprintf("%s/users/%s/cover/%s/%s%s", baseDir, userId.String(), date, id, ext)
		case media.RoleAvatar:
			return fmt.Sprintf("%s/users/%s/avatar/%s/%s%s", baseDir, userId.String(), date, id, ext)
		case media.RoleStory:
			return fmt.Sprintf("%s/users/%s/stories/%s/%s%s", baseDir, userId.String(), date, id, ext)
		default:
			return fmt.Sprintf("%s/users/%s/media/%s/%s%s", baseDir, userId.String(), date, id, ext)
		}
	case media.OwnerPost:
		return fmt.Sprintf("%s/users/%s/posts/%s/%s/%s%s", baseDir, userId.String(), date, ownerID.String(), id, ext)
	case media.OwnerNews:
		return fmt.Sprintf("%s/users/%s/news/%s/%s/%s%s", baseDir, userId.String(), date, ownerID.String(), id, ext)
	case media.OwnerChat:
		return fmt.Sprintf("%s/users/%s/chat/%s/%s/%s%s", baseDir, userId.String(), date, ownerID.String(), id, ext)
	default:
		return fmt.Sprintf("%s/users/%s/other/%s/%s/%s%s", baseDir, userId.String(), date, ownerID.String(), id, ext)
	}
}

func getRoleWidth(role media.MediaRole, size string) int {
	switch role {
	case media.RoleCover:
		switch size {
		case "small":
			return 1280
		case "medium":
			return 1920
		case "large":
			return 2560
		}
	case media.RoleAvatar:
		switch size {
		case "small":
			return 128
		case "medium":
			return 256
		case "large":
			return 512
		}
	case media.RoleStory:
		switch size {
		case "small":
			return 480
		case "medium":
			return 720
		case "large":
			return 1080
		}
	default:
		// Diğer medya türleri için dikey
		switch size {
		case "small":
			return 480
		case "medium":
			return 720
		case "large":
			return 1080
		}
	}
	return 720
}

func getRoleHeight(role media.MediaRole, size string) int {
	switch role {
	case media.RoleCover:
		switch size {
		case "small":
			return 720
		case "medium":
			return 1080
		case "large":
			return 1440
		}
	case media.RoleAvatar:
		// Kare olmalı
		return getRoleWidth(role, size)
	case media.RoleStory:
		switch size {
		case "small":
			return 720
		case "medium":
			return 1080
		case "large":
			return 1920
		}
	default:
		// Diğer medya türleri için dikey
		switch size {
		case "small":
			return 720
		case "medium":
			return 1080
		case "large":
			return 1440
		}
	}
	return 1080
}

// --- HELPERLAR ---

func makeVariant(path, ext string) *utils.VariantInfo {
	img, err := imaging.Open(path)
	if err != nil {
		return nil
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	return &utils.VariantInfo{
		URL:    strings.TrimPrefix(path, "."),
		Width:  &w,
		Height: &h,
		Format: strings.TrimPrefix(ext, "."),
		Size:   getFileSizeSafe(path),
	}
}

func getFileSizeSafe(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func publicFileURL(path string) string {
	return strings.TrimPrefix(path, ".")
}

func shouldProcessAsync(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/") || strings.HasPrefix(mimeType, "video/")
}

func initialFileVariants(mimeType, storagePath, ext string, size int64) *utils.FileVariants {
	if !strings.HasPrefix(mimeType, "image/") {
		return nil
	}

	format := strings.TrimPrefix(strings.ToLower(ext), ".")
	return &utils.FileVariants{
		Image: &utils.ImageVariants{
			Original: &utils.VariantInfo{
				URL:    publicFileURL(storagePath),
				Format: format,
				Size:   size,
			},
		},
	}
}

func (r *MediaRepository) MakeSureDirectoryPathExists(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, os.ModePerm)
}

func (r *MediaRepository) SaveUploadedFile(file ports.UploadedFile, path string) error {
	if err := r.MakeSureDirectoryPathExists(path); err != nil {
		return err
	}

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer func() {
		if cerr := src.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	dst, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := dst.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	_, err = dst.ReadFrom(src)
	return err
}

func (r *MediaRepository) generateImageVariants(originalPath string, ext string, role media.MediaRole) (*utils.ImageVariants, *int, *int, error) {
	img, err := helpers.LoadImageWithOrientation(originalPath)
	if err != nil {
		return nil, nil, nil, err
	}

	baseDir := filepath.Dir(originalPath)
	baseName := strings.TrimSuffix(filepath.Base(originalPath), ext)

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	original := &utils.VariantInfo{
		URL:    strings.TrimPrefix(originalPath, "."),
		Width:  &w,
		Height: &h,
		Format: strings.TrimPrefix(ext, "."),
		Size:   getFileSizeSafe(originalPath),
	}

	ext = ".webp" // Hepsi webp formatında kaydedilecek

	// Fonksiyon seçimi ve aspect belirle
	var resizeFunc func(string, string, int, int) error
	var aspect string

	switch role {
	case media.RoleAvatar:
		aspect = "square"
		resizeFunc = helpers.ResizeSquareCrop
	case media.RoleCover:
		aspect = "landscape"
		resizeFunc = helpers.ResizeLandscapeKeepAspect
	case media.RoleStory:
		aspect = "portrait"
		resizeFunc = helpers.ResizePortraitKeepAspect
	default:
		aspect = "landscape"
		resizeFunc = helpers.ResizeLandscapeKeepAspect
	}

	// ICON: Avatar ise square crop, diğerleri için blur arka planla kare kutuya sığdır
	iconPath := filepath.Join(baseDir, baseName+"_"+aspect+"_icon"+ext)
	if role == media.RoleAvatar {
		if err := helpers.ResizeSquareCrop(originalPath, iconPath, 128, 128); err != nil {
			return nil, &w, &h, err
		}
	} else {
		if err := helpers.ResizeSquareKeepAspect(originalPath, iconPath, 128, 128); err != nil {
			return nil, &w, &h, err
		}
	}

	// Thumbnail: exact 240x240, blur arka planla aspect korunur.
	thumbPath := filepath.Join(baseDir, baseName+"_"+aspect+"_thumb"+ext)
	if err := resizeFunc(originalPath, thumbPath, 240, 240); err != nil {
		return nil, &w, &h, err
	}

	// Small
	smallPath := filepath.Join(baseDir, baseName+"_"+aspect+"_sm"+ext)
	if err := resizeFunc(originalPath, smallPath, getRoleWidth(role, "small"), getRoleHeight(role, "small")); err != nil {
		return nil, &w, &h, err
	}

	// Medium
	mediumPath := filepath.Join(baseDir, baseName+"_"+aspect+"_md"+ext)
	if err := resizeFunc(originalPath, mediumPath, getRoleWidth(role, "medium"), getRoleHeight(role, "medium")); err != nil {
		return nil, &w, &h, err
	}

	// Large
	largePath := filepath.Join(baseDir, baseName+"_"+aspect+"_lg"+ext)
	if err := resizeFunc(originalPath, largePath, getRoleWidth(role, "large"), getRoleHeight(role, "large")); err != nil {
		return nil, &w, &h, err
	}

	return &utils.ImageVariants{
		Original:  original,
		Icon:      makeVariant(iconPath, "webp"),
		Thumbnail: makeVariant(thumbPath, "webp"),
		Small:     makeVariant(smallPath, "webp"),
		Medium:    makeVariant(mediumPath, "webp"),
		Large:     makeVariant(largePath, "webp"),
	}, &w, &h, nil
}

// generateVideoVariants: video için poster + 3 kalite + preview üretir
// döndürür: *utils.VideoVariants, *width, *height, error
func (r *MediaRepository) generateVideoVariants(originalPath string, ext string, role media.MediaRole) (*utils.VideoVariants, *int, *int, error) {
	// 1) ffprobe ile orijinal çözünürlüğü al
	width, height, err := probeVideoDimensions(originalPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("probe video dimensions: %w", err)
	}

	fmt.Println("ROLE", role)
	// helper to create output filenames in same dir
	baseDir := filepath.Dir(originalPath)
	baseName := strings.TrimSuffix(filepath.Base(originalPath), ext)

	// Ensure directory exists
	if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
		return nil, nil, nil, err
	}

	// paths
	posterPath := filepath.Join(baseDir, baseName+"_poster.jpg")
	lowPath := filepath.Join(baseDir, baseName+"_low.mp4")       // 480p
	mediumPath := filepath.Join(baseDir, baseName+"_medium.mp4") // 720p
	highPath := filepath.Join(baseDir, baseName+"_high.mp4")     // 1080p
	previewPath := filepath.Join(baseDir, baseName+"_preview.mp4")

	// 2) Extract poster (single frame at 1s)
	// -y overwrite, -ss for seek, -i input, -frames:v 1 output
	if err := runCmd("ffmpeg", "-y", "-ss", "00:00:01", "-i", originalPath, "-frames:v", "1", "-q:v", "2", posterPath); err != nil {
		// poster extraction hatası kritik değil, ama uyar
		fmt.Println("WARN: failed to extract poster:", err)
	}

	// 3) Transcode low / medium / high
	// Kullanılacak kare genişlik/height değerlerini (target) belirle.
	// Oran korunacak şekilde scale parametresi veriyoruz (ffmpeg scale=-2:480 vb.)
	// -2 kullanıyoruz böylece width/heigth çiftleri 2'nin katı olacak (codec uyumu)
	if err := runCmd("ffmpeg", "-y", "-i", originalPath, "-c:v", "libx264", "-preset", "veryfast", "-crf", "28", "-c:a", "aac", "-b:a", "96k", "-vf", "scale=-2:480", lowPath); err != nil {
		fmt.Println("WARN: failed to encode low:", err)
	}
	if err := runCmd("ffmpeg", "-y", "-i", originalPath, "-c:v", "libx264", "-preset", "fast", "-crf", "24", "-c:a", "aac", "-b:a", "128k", "-vf", "scale=-2:720", mediumPath); err != nil {
		fmt.Println("WARN: failed to encode medium:", err)
	}
	if err := runCmd("ffmpeg", "-y", "-i", originalPath, "-c:v", "libx264", "-preset", "slow", "-crf", "22", "-c:a", "aac", "-b:a", "192k", "-vf", "scale=-2:1080", highPath); err != nil {
		fmt.Println("WARN: failed to encode high:", err)
	}

	// 4) Preview: kısa sessiz loop (ör. 3 saniye), scaled to 360p for small preview
	// create a 3s clip starting from 0s, remove audio (-an), set bitrate low to keep small size
	if err := runCmd("ffmpeg", "-y", "-ss", "00:00:00", "-t", "3", "-i", originalPath, "-an", "-c:v", "libx264", "-preset", "veryfast", "-crf", "28", "-vf", "scale=-2:360", previewPath); err != nil {
		fmt.Println("WARN: failed to create preview:", err)
	}

	// 5) build VariantInfo structs (URL = path without leading dot)
	makeVideoVariant := func(p string) *utils.VariantInfo {
		if _, er := os.Stat(p); er != nil {
			return nil
		}
		w, h := probeFileDimensionsOrNil(p) // helper to probe or nil
		size := getFileSizeSafe(p)
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(p)), ".")
		return &utils.VariantInfo{
			URL:    strings.TrimPrefix(p, "."),
			Width:  w,
			Height: h,
			Format: ext,
			Size:   size,
		}
	}

	videoVars := &utils.VideoVariants{
		Poster:  makeImageVariantInfo(posterPath),
		Low:     makeVideoVariant(lowPath),
		Medium:  makeVideoVariant(mediumPath),
		High:    makeVideoVariant(highPath),
		Preview: makeVideoVariant(previewPath),
	}

	// dönülecek width/height orijinal video çözünürlüğü
	wptr := new(int)
	hptr := new(int)
	*wptr = width
	*hptr = height

	return videoVars, wptr, hptr, nil
}

// ---------- yardımcı fonksiyonlar ----------

// runCmd çalıştırıp stderr/stdout yakalar, hata döner
func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v failed: %v - %s", name, args, err, stderr.String())
	}
	return nil
}

// probeVideoDimensions ffprobe ile width x height döndürür
func probeVideoDimensions(path string) (int, int, error) {
	// ffprobe -v error -select_streams v:0 -show_entries stream=width,height -of csv=p=0:s=x input
	out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=p=0:s=x", path).Output()
	if err != nil {
		return 0, 0, err
	}
	s := strings.TrimSpace(string(out))
	parts := strings.Split(s, "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected ffprobe output: %s", s)
	}
	w, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	h, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return w, h, nil
}

// probeFileDimensionsOrNil: eğer ffprobe varsa dim döndürür, yoksa nil pointer
func probeFileDimensionsOrNil(path string) (*int, *int) {
	w, h, err := probeVideoDimensions(path)
	if err != nil {
		// dene image olarak açmayı (poster jpg için)
		img, imgErr := imaging.Open(path)
		if imgErr == nil {
			b := img.Bounds()
			wi := b.Dx()
			hi := b.Dy()
			return &wi, &hi
		}
		return nil, nil
	}
	return &w, &h
}

// makeImageVariantInfo (poster için) - poster genellikle jpg
func makeImageVariantInfo(path string) *utils.VariantInfo {
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	img, err := imaging.Open(path)
	if err != nil {
		return &utils.VariantInfo{
			URL:    strings.TrimPrefix(path, "."),
			Format: strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), "."),
			Size:   getFileSizeSafe(path),
		}
	}
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	return &utils.VariantInfo{
		URL:    strings.TrimPrefix(path, "."),
		Width:  &w,
		Height: &h,
		Format: strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), "."),
		Size:   getFileSizeSafe(path),
	}
}

func (r *MediaRepository) MakeVideoVariant(path string) *utils.VariantInfo {
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	w, h, err := probeVideoDimensions(path)
	var wptr, hptr *int
	if err == nil {
		wptr = new(int)
		hptr = new(int)
		*wptr = w
		*hptr = h
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	return &utils.VariantInfo{
		URL:    strings.TrimPrefix(path, "."),
		Width:  wptr,
		Height: hptr,
		Format: ext,
		Size:   getFileSizeSafe(path),
	}
}

func (r *MediaRepository) ClaimNextPendingMedia(now time.Time) (*media.Media, error) {
	var claimedID uuid.UUID

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var claimed media.Media
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("processing_status = ?", media.ProcessingStatusPending).
			Order("created_at ASC").
			First(&claimed).Error; err != nil {
			return err
		}

		claimedID = claimed.ID
		return tx.Model(&media.Media{}).
			Where("id = ?", claimed.ID).
			Updates(map[string]interface{}{
				"processing_status":     media.ProcessingStatusProcessing,
				"processing_started_at": now,
				"processing_error":      nil,
				"processing_attempts":   gorm.Expr("processing_attempts + ?", 1),
			}).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	var claimed media.Media
	if err := r.db.Preload("File").First(&claimed, "id = ?", claimedID).Error; err != nil {
		return nil, err
	}

	return &claimed, nil
}

func (r *MediaRepository) RequeueStaleProcessing(timeout time.Duration) (int64, error) {
	cutoff := time.Now().Add(-timeout)
	result := r.db.Model(&media.Media{}).
		Where("processing_status = ? AND processing_started_at IS NOT NULL AND processing_started_at < ?", media.ProcessingStatusProcessing, cutoff).
		Updates(map[string]interface{}{
			"processing_status":     media.ProcessingStatusPending,
			"processing_started_at": nil,
			"processing_error":      "processing timed out and was re-queued",
		})

	return result.RowsAffected, result.Error
}

func (r *MediaRepository) markMediaFailed(mediaID uuid.UUID, processErr error) error {
	msg := processErr.Error()
	if len(msg) > 2048 {
		msg = msg[:2048]
	}

	return r.db.Model(&media.Media{}).
		Where("id = ?", mediaID).
		Updates(map[string]interface{}{
			"processing_status":     media.ProcessingStatusFailed,
			"processing_error":      msg,
			"processing_started_at": nil,
		}).Error
}

func (r *MediaRepository) markMediaReady(item *media.Media, width, height *int, variants *utils.FileVariants) error {
	now := time.Now()

	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&utils.FileMetadata{}).
			Where("id = ?", item.FileID).
			Updates(map[string]interface{}{
				"url":      publicFileURL(item.File.StoragePath),
				"width":    width,
				"height":   height,
				"variants": variants,
			}).Error; err != nil {
			return err
		}

		return tx.Model(&media.Media{}).
			Where("id = ?", item.ID).
			Updates(map[string]interface{}{
				"processing_status":     media.ProcessingStatusReady,
				"processing_error":      nil,
				"processing_started_at": nil,
				"processed_at":          now,
			}).Error
	})
}

func (r *MediaRepository) ProcessClaimedMedia(item *media.Media) error {
	if item == nil {
		return nil
	}

	ext := strings.ToLower(filepath.Ext(item.File.StoragePath))

	switch {
	case strings.HasPrefix(item.File.MimeType, "image/"):
		imageVariants, width, height, err := r.generateImageVariants(item.File.StoragePath, ext, item.Role)
		if err != nil {
			if markErr := r.markMediaFailed(item.ID, err); markErr != nil {
				return fmt.Errorf("process media: %w; mark failed: %v", err, markErr)
			}
			return err
		}
		return r.markMediaReady(item, width, height, &utils.FileVariants{Image: imageVariants})
	case strings.HasPrefix(item.File.MimeType, "video/"):
		videoVariants, width, height, err := r.generateVideoVariants(item.File.StoragePath, ext, item.Role)
		if err != nil {
			if markErr := r.markMediaFailed(item.ID, err); markErr != nil {
				return fmt.Errorf("process media: %w; mark failed: %v", err, markErr)
			}
			return err
		}
		return r.markMediaReady(item, width, height, &utils.FileVariants{Video: videoVariants})
	default:
		return r.markMediaReady(item, item.File.Width, item.File.Height, item.File.Variants)
	}
}

func (r *MediaRepository) AddMedia(ownerID uuid.UUID, ownerType media.OwnerType, userId uuid.UUID, role media.MediaRole, file ports.UploadedFile) (*media.Media, error) {
	if file == nil {
		return nil, domainmedia.ErrEmptyFile
	}
	upload, err := domainmedia.NewUploadedFile(file.Filename(), file.Size(), file.ContentType())
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(upload.Filename)
	newFileName := fmt.Sprintf("%d_%s%s", time.Now().Unix(), uuid.New().String(), ext)
	storagePath := r.GenerateStoragePath(userId, ownerID, ownerType, role, newFileName)

	if err := r.SaveUploadedFile(file, storagePath); err != nil {
		return nil, err
	}

	mimeType := upload.ContentType

	status := media.ProcessingStatusReady
	if shouldProcessAsync(mimeType) {
		status = media.ProcessingStatusPending
	}

	variants := initialFileVariants(mimeType, storagePath, ext, upload.Size)

	media := media.Media{
		ID:               uuid.New(),
		PublicID:         r.snowFlakeNode.Generate().Int64(),
		FileID:           uuid.New(),
		OwnerID:          ownerID,
		UserID:           userId,
		OwnerType:        ownerType,
		Role:             role,
		IsPublic:         isPublicMediaRole(role),
		ProcessingStatus: status,
		File: utils.FileMetadata{
			ID:          uuid.New(),
			URL:         publicFileURL(storagePath),
			StoragePath: storagePath,
			MimeType:    mimeType,
			Size:        upload.Size,
			Name:        upload.Filename,
			Variants:    variants,
			CreatedAt:   time.Now(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&media.File).Error; err != nil {
			return err
		}
		return tx.Create(&media).Error
	}); err != nil {
		return nil, err
	}

	return &media, nil
}

func isPublicMediaRole(role media.MediaRole) bool {
	return role != media.RoleChatImage && role != media.RoleChatMedia && role != media.RoleChatVideo
}
