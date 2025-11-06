package shared

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type FileMetadata struct {
	ID          uuid.UUID     `gorm:"type:uuid;primaryKey" json:"id"`
	URL         string        `gorm:"size:512;not null" json:"url"`          // CDN veya public URL
	StoragePath string        `gorm:"size:512;not null" json:"storage_path"` // CDN veya local path
	MimeType    string        `gorm:"size:128;not null" json:"mime_type"`    // "image/png", "video/mp4" vs.
	Size        int64         `gorm:"not null" json:"size"`                  // Bytes cinsinden
	Name        string        `gorm:"size:255" json:"name"`                  // Orijinal dosya adı
	Width       *int          `gorm:"null" json:"width,omitempty"`           // Resim/video için
	Height      *int          `gorm:"null" json:"height,omitempty"`          // Resim/video için
	Duration    *float64      `gorm:"null" json:"duration,omitempty"`        // Ses/video için saniye cinsinden
	Variants    *FileVariants `gorm:"type:jsonb" json:"variants,omitempty"`
	CreatedAt   time.Time     `json:"created_at"` // Oluşturulma zamanı
}

// FileVariants hem görsel hem video varyantlarını kapsar
type FileVariants struct {
	Image *ImageVariants `json:"image,omitempty"`
	Video *VideoVariants `json:"video,omitempty"`
}

// Görsel varyantları
type ImageVariants struct {
	Icon      *VariantInfo `json:"icon,omitempty"`      // 128x128//profil
	Thumbnail *VariantInfo `json:"thumbnail,omitempty"` // 120x180//story
	Small     *VariantInfo `json:"small,omitempty"`     // 480x480
	Medium    *VariantInfo `json:"medium,omitempty"`    // 720x720
	Large     *VariantInfo `json:"large,omitempty"`     // 1080x1080
	Original  *VariantInfo `json:"original,omitempty"`
}

// Video varyantları
type VideoVariants struct {
	Poster  *VariantInfo `json:"poster,omitempty"`  // İlk frame veya cover
	Low     *VariantInfo `json:"low,omitempty"`     // 480p
	Medium  *VariantInfo `json:"medium,omitempty"`  // 720p
	High    *VariantInfo `json:"high,omitempty"`    // 1080p
	Preview *VariantInfo `json:"preview,omitempty"` // Sessiz kısa loop (splash, anim icon vs.)
}

// Her varyant için ortak bilgiler
type VariantInfo struct {
	URL      string   `json:"url"`
	Width    *int     `json:"width,omitempty"`
	Height   *int     `json:"height,omitempty"`
	Duration *float64 `json:"duration,omitempty"`
	Format   string   `json:"format,omitempty"`
	Size     int64    `json:"size,omitempty"`
}

func (FileMetadata) TableName() string {
	return "file_metadata"
}

// Scanner interface implementasyonu
func (fv *FileVariants) Scan(value interface{}) error {
	if value == nil {
		*fv = FileVariants{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan FileVariants: expected []byte but got %T", value)
	}

	err := json.Unmarshal(bytes, fv)
	if err != nil {
		return fmt.Errorf("failed to unmarshal FileVariants: %w", err)
	}
	return nil
}

// Valuer interface implementasyonu
func (fv FileVariants) Value() (driver.Value, error) {
	if fv.Image == nil && fv.Video == nil {
		return nil, nil
	}

	bytes, err := json.Marshal(fv)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal FileVariants: %w", err)
	}
	return bytes, nil
}
