package db

import (
	"core/models/media"
	modelutils "core/models/utils"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

func TestEnsureMediaVisibilityConstraintRejectsPublicProtectedMedia(t *testing.T) {
	database := migrationIntegrationDB(t)
	if err := database.Exec("ALTER TABLE medias DROP CONSTRAINT IF EXISTS chk_medias_protected_visibility").Error; err != nil {
		t.Fatalf("drop test constraint: %v", err)
	}

	if err := EnsureMediaVisibilityConstraint(database); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := EnsureMediaVisibilityConstraint(database); err != nil {
		t.Fatalf("second call must be idempotent: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	file := modelutils.FileMetadata{
		ID: uuid.New(), URL: "/static/protected-test", StoragePath: "./static/protected-test",
		MimeType: "image/jpeg", Name: "protected-test.jpg", CreatedAt: now,
	}
	if err := database.Omit(clause.Associations).Create(&file).Error; err != nil {
		t.Fatalf("create file metadata: %v", err)
	}

	item := media.Media{
		ID: uuid.New(), PublicID: now.UnixNano(), FileID: file.ID,
		OwnerID: uuid.New(), OwnerType: media.OwnerUser, UserID: uuid.New(), Role: media.RolePrivatePhoto,
		IsPublic: true, ProcessingStatus: media.ProcessingStatusReady, CreatedAt: now, UpdatedAt: now,
	}
	if err := database.Omit(clause.Associations).Create(&item).Error; err == nil {
		t.Fatal("database constraint accepted public private-photo media")
	}
}
