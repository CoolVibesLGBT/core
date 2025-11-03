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

func (r *MediaRepository) GenerateStoragePath(ownerID uuid.UUID, ownerType media.OwnerType, role media.MediaRole, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	id := uuid.New().String()
	date := time.Now().Format("2006-01-02") // YYYY-MM-DD
	baseDir := "./static"

	switch ownerType {
	case media.OwnerUser:
		switch role {
		case media.RoleProfile:
			return fmt.Sprintf("%s/users/%s/avatars/%s%s", baseDir, ownerID.String(), id, ext)
		case media.RoleCover:
			return fmt.Sprintf("%s/users/%s/covers/%s%s", baseDir, ownerID.String(), id, ext)
		case media.RoleStory:
			return fmt.Sprintf("%s/users/%s/stories/%s%s", baseDir, ownerID.String(), id, ext)
		default:
			return fmt.Sprintf("%s/users/%s/media/%s%s", baseDir, ownerID.String(), id, ext)
		}
	case media.OwnerPost:
		return fmt.Sprintf("%s/posts/%s/%s/%s%s", baseDir, ownerID.String(), date, id, ext)
	case media.OwnerBlog:
		return fmt.Sprintf("%s/blogs/%s/%s/%s%s", baseDir, ownerID.String(), date, id, ext)
	case media.OwnerChat:
		if role == media.RoleChatVideo {
			return fmt.Sprintf("%s/chat/%s/videos/%s%s", baseDir, ownerID.String(), id, ext)
		}
		return fmt.Sprintf("%s/chat/%s/images/%s%s", baseDir, ownerID.String(), id, ext)
	case media.OwnerPage:
		return fmt.Sprintf("%s/pages/%s/%s/%s%s", baseDir, ownerID.String(), date, id, ext)
	default:
		return fmt.Sprintf("%s/other/%s/%s%s", baseDir, ownerID.String(), id, ext)
	}
}

func (r *MediaRepository) AddUserMedia(userId uuid.UUID, role media.MediaRole, filename, url string, mimeType string, size int64, width, height *int) (*media.Media, error) {
	media := &media.Media{
		ID:        uuid.New(),
		FileID:    uuid.New(), // FileMetadata kaydı için
		PublicID:  r.snowFlakeNode.Generate().Int64(),
		OwnerID:   userId,
		UserID:    userId,
		OwnerType: media.OwnerUser,
		Role:      role,
		IsPublic:  true,
		File: shared.FileMetadata{
			ID:          uuid.New(),
			URL:         url,
			StoragePath: r.GenerateStoragePath(userId, media.OwnerUser, role, filename),
			MimeType:    mimeType,
			Size:        size,
			Name:        filename,
			Width:       width,
			Height:      height,
			CreatedAt:   time.Now(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// DB'ye kaydet
	if err := r.db.Create(&media.File).Error; err != nil {
		return nil, err
	}

	if err := r.db.Create(media).Error; err != nil {
		return nil, err
	}

	return media, nil
}

// Generic media ekleme
func (r *MediaRepository) AddMedia(ownerID uuid.UUID, ownerType media.OwnerType, userId uuid.UUID, role media.MediaRole, file *multipart.FileHeader) (*media.Media, error) {
	ext := filepath.Ext(file.Filename)
	newFileName := fmt.Sprintf("%d_%s%s", time.Now().Unix(), uuid.New().String(), ext)
	storagePath := r.GenerateStoragePath(ownerID, ownerType, role, newFileName)

	urlPath := fmt.Sprintf("%s%s", helpers.MD5Hash(storagePath), ext)
	fmt.Println("STORAGE PATH", storagePath)
	fmt.Println("URL PATH", urlPath)

	if err := r.SaveUploadedFile(file, storagePath); err != nil {
		return nil, err
	}

	// Burada basit width/height ve duration default null bırakıldı
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
			URL:         urlPath,
			StoragePath: storagePath,
			MimeType:    file.Header.Get("Content-Type"),
			Size:        file.Size,
			Name:        file.Filename,
			CreatedAt:   time.Now(),
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

// Helper

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

/*


avatarFile := FileMetadata{
	ID:         uuid.New(),
	URL:        "https://cdn.example.com/avatar.png",
	StoragePath: "users/avatars/avatar.png",
	MimeType:   "image/png",
	Size:       120000,
	Width:      ptrInt(512),
	Height:     ptrInt(512),
	CreatedAt:  time.Now(),
}

avatarMedia, _ := mediaRepo.AddMedia(user.ID, OwnerUser, RoleProfile, avatarFile, true)
user.ProfileImageURL = &avatarMedia.File.URL
userRepo.DB().Save(&user)

coverFile := FileMetadata{
	ID:         uuid.New(),
	URL:        "https://cdn.example.com/cover.png",
	StoragePath: "users/covers/cover.png",
	MimeType:   "image/png",
	Size:       240000,
	Width:      ptrInt(1200),
	Height:     ptrInt(400),
	CreatedAt:  time.Now(),
}

coverMedia, _ := mediaRepo.AddMedia(user.ID, OwnerUser, RoleCover, coverFile, true)

chatFile := FileMetadata{
	ID:         uuid.New(),
	URL:        "https://cdn.example.com/chat123.png",
	StoragePath: "chat/room123/2025-10-24/chat123.png",
	MimeType:   "image/png",
	Size:       50000,
	Width:      ptrInt(800),
	Height:     ptrInt(600),
	CreatedAt:  time.Now(),
}

chatMedia, _ := mediaRepo.AddMedia(chatID, OwnerChat, RoleChatImage, chatFile, false)

*/
