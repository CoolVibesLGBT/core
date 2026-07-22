package db

import (
	"core/models/media"
	modelutils "core/models/utils"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

func TestMigrateProtectedMediaVisibilityBackfillsAndConstrains(t *testing.T) {
	database := migrationIntegrationDB(t)
	if err := database.Exec("ALTER TABLE medias DROP CONSTRAINT IF EXISTS chk_medias_protected_visibility").Error; err != nil {
		t.Fatalf("drop test constraint: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	protectedRoles := []media.MediaRole{
		media.RolePrivatePhoto,
		media.RoleChatImage,
		media.RoleChatMedia,
		media.RoleChatVideo,
	}
	items := make([]media.Media, 0, len(protectedRoles))
	for index, role := range protectedRoles {
		file := modelutils.FileMetadata{
			ID: uuid.New(), URL: "/static/protected-test", StoragePath: "./static/protected-test",
			MimeType: "image/jpeg", Name: "protected-test.jpg", CreatedAt: now,
		}
		if err := database.Omit(clause.Associations).Create(&file).Error; err != nil {
			t.Fatalf("create file metadata: %v", err)
		}
		item := media.Media{
			ID: uuid.New(), PublicID: now.UnixNano() + int64(index) + 1, FileID: file.ID,
			OwnerID: uuid.New(), OwnerType: media.OwnerUser, UserID: uuid.New(), Role: role,
			IsPublic: true, ProcessingStatus: media.ProcessingStatusReady, CreatedAt: now, UpdatedAt: now,
		}
		if err := database.Omit(clause.Associations).Create(&item).Error; err != nil {
			t.Fatalf("create legacy %q media: %v", role, err)
		}
		items = append(items, item)
	}

	if err := MigrateProtectedMediaVisibility(database); err != nil {
		t.Fatalf("first migration: %v", err)
	}
	if err := MigrateProtectedMediaVisibility(database); err != nil {
		t.Fatalf("second migration: %v", err)
	}
	for _, item := range items {
		var isPublic bool
		if err := database.Model(&media.Media{}).Select("is_public").Where("id = ?", item.ID).Scan(&isPublic).Error; err != nil {
			t.Fatalf("read migrated %q media: %v", item.Role, err)
		}
		if isPublic {
			t.Fatalf("migrated protected role %q remained public", item.Role)
		}
	}

	if err := database.Model(&media.Media{}).
		Where("id = ?", items[0].ID).
		UpdateColumn("is_public", true).Error; err == nil {
		t.Fatal("database constraint accepted public private-photo media")
	}
}
