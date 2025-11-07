package repositories

import (
	"bytes"
	"coolvibes/helpers"
	"coolvibes/models/media"
	"coolvibes/models/shared"
	"fmt"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// generateVideoVariants: video için poster + 3 kalite + preview üretir
// döndürür: *shared.VideoVariants, *width, *height, error
func (r *MediaRepository) generateVideoVariants(originalPath string, ext string, role media.MediaRole) (*shared.VideoVariants, *int, *int, error) {
	// 1) ffprobe ile orijinal çözünürlüğü al
	width, height, err := probeVideoDimensions(originalPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("probe video dimensions: %w", err)
	}

	// helper to create output filenames in same dir
	baseDir := filepath.Dir(originalPath)
	baseName := strings.TrimSuffix(filepath.Base(originalPath), ext)
	if ext == "" {
		ext = filepath.Ext(originalPath)
	}

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
	makeVideoVariant := func(p string) *shared.VariantInfo {
		if _, er := os.Stat(p); er != nil {
			return nil
		}
		w, h := probeFileDimensionsOrNil(p) // helper to probe or nil
		size := getFileSizeSafe(p)
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(p)), ".")
		return &shared.VariantInfo{
			URL:    strings.TrimPrefix(p, "."),
			Width:  w,
			Height: h,
			Format: ext,
			Size:   size,
		}
	}

	videoVars := &shared.VideoVariants{
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
func makeImageVariantInfo(path string) *shared.VariantInfo {
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	img, err := imaging.Open(path)
	if err != nil {
		return &shared.VariantInfo{
			URL:    strings.TrimPrefix(path, "."),
			Format: strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), "."),
			Size:   getFileSizeSafe(path),
		}
	}
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	return &shared.VariantInfo{
		URL:    strings.TrimPrefix(path, "."),
		Width:  &w,
		Height: &h,
		Format: strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), "."),
		Size:   getFileSizeSafe(path),
	}
}

// makeVideoVariant - video için VariantInfo
func makeVideoVariant(path string) *shared.VariantInfo {
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
	return &shared.VariantInfo{
		URL:    strings.TrimPrefix(path, "."),
		Width:  wptr,
		Height: hptr,
		Format: ext,
		Size:   getFileSizeSafe(path),
	}
}

func (r *MediaRepository) AddMedia(ownerID uuid.UUID, ownerType media.OwnerType, userId uuid.UUID, role media.MediaRole, file *multipart.FileHeader) (*media.Media, error) {
	ext := filepath.Ext(file.Filename)
	newFileName := fmt.Sprintf("%d_%s%s", time.Now().Unix(), uuid.New().String(), ext)
	storagePath := r.GenerateStoragePath(userId, ownerID, ownerType, role, newFileName)

	// Dosyayı kaydet
	if err := r.SaveUploadedFile(file, storagePath); err != nil {
		return nil, err
	}

	mimeType := file.Header.Get("Content-Type")

	var (
		variants *shared.FileVariants
		width    *int
		height   *int
		err      error
	)

	// MIME tipine göre varyant üretimi
	if strings.HasPrefix(mimeType, "image/") {
		var imageVariants *shared.ImageVariants
		var w, h *int
		imageVariants, w, h, err = r.generateImageVariants(storagePath, ext, role)
		if err != nil {
			fmt.Println("WARN: image variant generation failed:", err)
		} else {
			variants = &shared.FileVariants{Image: imageVariants}
			width, height = w, h
		}
	} else if strings.HasPrefix(mimeType, "video/") {
		var videoVariants *shared.VideoVariants
		var w, h *int
		var vidErr error
		videoVariants, w, h, vidErr = r.generateVideoVariants(storagePath, ext, role)
		if vidErr != nil {
			fmt.Println("WARN: video variant generation failed:", vidErr)
		} else {
			variants = &shared.FileVariants{Video: videoVariants}
			width, height = w, h
		}
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
			MimeType:    mimeType,
			Size:        file.Size,
			Name:        file.Filename,
			Width:       width,
			Height:      height,
			Variants:    variants,
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
