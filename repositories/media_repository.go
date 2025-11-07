package repositories

import (
	"coolvibes/helpers"
	"coolvibes/models/media"
	"coolvibes/models/shared"
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
			return 1440
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

func (r *MediaRepository) generateImageVariants(originalPath string, ext string, role media.MediaRole) (*shared.ImageVariants, *int, *int, error) {
	img, err := imaging.Open(originalPath)
	if err != nil {
		return nil, nil, nil, err
	}

	baseDir := filepath.Dir(originalPath)
	baseName := strings.TrimSuffix(filepath.Base(originalPath), ext)

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	original := &shared.VariantInfo{
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
		resizeFunc = helpers.ResizeSquareKeepAspect
	case media.RoleCover:
		aspect = "landscape"
		resizeFunc = helpers.ResizeLandscapeKeepAspect
	case media.RoleStory:
		aspect = "portrait"
		resizeFunc = helpers.ResizePortraitKeepAspect
	default:
		aspect = "portrait"
		resizeFunc = helpers.ResizePortraitKeepAspect
	}

	// ICON: Avatar ise square crop, diğerleri için square fit
	iconPath := filepath.Join(baseDir, baseName+"_"+aspect+"_icon"+ext)
	if role == media.RoleAvatar {
		if err := helpers.ResizeSquareCrop(originalPath, iconPath, 128, 128); err != nil {
			return nil, &w, &h, err
		}
	} else {
		// Diğerleri için square aspect keep resize 128x128
		if err := helpers.ResizeSquareKeepAspect(originalPath, iconPath, 128, 128); err != nil {
			return nil, &w, &h, err
		}
	}

	// Thumbnail 240x240 role göre
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

	return &shared.ImageVariants{
		Original:  original,
		Icon:      makeVariant(iconPath, "webp"),
		Thumbnail: makeVariant(thumbPath, "webp"),
		Small:     makeVariant(smallPath, "webp"),
		Medium:    makeVariant(mediumPath, "webp"),
		Large:     makeVariant(largePath, "webp"),
	}, &w, &h, nil
}

// --- ANA FONKSİYON ---
func (r *MediaRepository) AddMedia(ownerID uuid.UUID, ownerType media.OwnerType, userId uuid.UUID, role media.MediaRole, file *multipart.FileHeader) (*media.Media, error) {
	ext := filepath.Ext(file.Filename)
	newFileName := fmt.Sprintf("%d_%s%s", time.Now().Unix(), uuid.New().String(), ext)
	storagePath := r.GenerateStoragePath(userId, ownerID, ownerType, role, newFileName)

	if err := r.SaveUploadedFile(file, storagePath); err != nil {
		return nil, err
	}

	variants, width, height, err := r.generateImageVariants(storagePath, ext, role)
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
