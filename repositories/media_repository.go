package repositories

import (
	"bifrost/helpers"
	"bifrost/models/media"
	"bifrost/models/shared"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MediaRepository struct {
	db            *gorm.DB
	snowFlakeNode *helpers.Node
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
	case media.OwnerChat:
		return fmt.Sprintf("%s/users/%s/chat/%s/%s/%s%s", baseDir, userId.String(), date, ownerID.String(), id, ext)
	default:
		return fmt.Sprintf("%s/users/%s/other/%s/%s/%s%s", baseDir, userId.String(), date, ownerID.String(), id, ext)
	}
}

// --- ANA FONKSİYON ---
func (r *MediaRepository) AddMedia(ownerID uuid.UUID, ownerType media.OwnerType, userId uuid.UUID, role media.MediaRole, file *multipart.FileHeader) (*media.Media, error) {
	ext := filepath.Ext(file.Filename)
	newFileName := fmt.Sprintf("%d_%s%s", time.Now().Unix(), uuid.New().String(), ext)
	storagePath := r.GenerateStoragePath(userId, ownerID, ownerType, role, newFileName)

	if err := r.SaveUploadedFile(file, storagePath); err != nil {
		return nil, err
	}

	variants, width, height, err := r.generateImageVariants(storagePath, ext)
	if err != nil {
		fmt.Println("WARN: Variant generation failed:", err)
	}

	media := media.Media{
		ID:        uuid.New(),
		PublicID:  r.snowFlakeNode.Generate().Int64(),
		FileID:    uuid.New(),
		OwnerID:   ownerID,
		UserID:    userId,
		OwnerType: ownerType,
		Role:      role,
		IsPublic:  true,
		File: shared.FileMetadata{
			ID:          uuid.New(),
			StoragePath: storagePath,
			MimeType:    file.Header.Get("Content-Type"),
			Size:        file.Size,
			Name:        file.Filename,
			Width:       width,
			Height:      height,
			Variants: &shared.FileVariants{
				Image: variants,
			},
			CreatedAt: time.Now(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := r.db.Create(&media.File).Error; err != nil {
		return nil, err
	}
	if err := r.db.Create(&media).Error; err != nil {
		return nil, err
	}

	return &media, nil
}

// --- VARYANT ÜRETİCİ ---
func (r *MediaRepository) generateImageVariants(originalPath string, ext string) (*shared.ImageVariants, *int, *int, error) {
	img, err := imaging.Open(originalPath)
	if err != nil {
		return nil, nil, nil, err
	}

	baseDir := filepath.Dir(originalPath)
	baseName := strings.TrimSuffix(filepath.Base(originalPath), ext)

	// Original ölçüleri
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// --- Original ---
	original := &shared.VariantInfo{
		URL:    strings.TrimPrefix(originalPath, "."),
		Width:  &w,
		Height: &h,
		Format: strings.TrimPrefix(ext, "."),
		Size:   getFileSizeSafe(originalPath),
	}

	// --- Thumbnail (128x128 portrait crop) ---
	thumbPath := filepath.Join(baseDir, baseName+"_thumb"+ext)
	if err := helpers.ResizePortraitCrop(originalPath, thumbPath, 128, 128); err != nil {
		return nil, &w, &h, err
	}

	// --- Small (480x720) ---
	smallPath := filepath.Join(baseDir, baseName+"_sm"+ext)
	helpers.ResizePortraitCrop(originalPath, smallPath, 480, 720)

	// --- Medium (720x1080) ---
	mediumPath := filepath.Join(baseDir, baseName+"_md"+ext)
	helpers.ResizePortraitCrop(originalPath, mediumPath, 720, 1080)

	// --- Large (1080x1440) ---
	largePath := filepath.Join(baseDir, baseName+"_lg"+ext)
	helpers.ResizePortraitCrop(originalPath, largePath, 1080, 1440)

	return &shared.ImageVariants{
		Original:  original,
		Thumbnail: makeVariant(thumbPath, ext),
		Small:     makeVariant(smallPath, ext),
		Medium:    makeVariant(mediumPath, ext),
		Large:     makeVariant(largePath, ext),
	}, &w, &h, nil
}

// --- HELPERLAR ---

func makeVariant(path, ext string) *shared.VariantInfo {
	img, err := imaging.Open(path)
	if err != nil {
		return nil
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	return &shared.VariantInfo{
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

func (r *MediaRepository) MakeSureDirectoryPathExists(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, os.ModePerm)
}

func (r *MediaRepository) SaveUploadedFile(file *multipart.FileHeader, path string) error {
	if err := r.MakeSureDirectoryPathExists(path); err != nil {
		return err
	}

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(path)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = dst.ReadFrom(src)
	return err
}
