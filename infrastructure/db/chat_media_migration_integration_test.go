package db

import (
	"core/constants"
	"core/models"
	"core/models/media"
	"core/models/post"
	modelutils "core/models/utils"
	"gorm.io/gorm/logger"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func migrationIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	schema := "migration_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	for _, extension := range []string{"uuid-ossp", "postgis"} {
		if err := EnableExtension(admin, extension); err != nil {
			t.Fatal(err)
		}
	}
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = admin.Exec("DROP SCHEMA " + schema + " CASCADE").Error
		pool, _ := admin.DB()
		_ = pool.Close()
	})
	if strings.Contains(dsn, "://") {
		parsed, err := url.Parse(dsn)
		if err != nil {
			t.Fatal(err)
		}
		query := parsed.Query()
		query.Set("search_path", schema+",public")
		parsed.RawQuery = query.Encode()
		dsn = parsed.String()
	} else {
		dsn += " search_path=" + schema + ",public"
	}
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true, IgnoreRelationshipsWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := database.DB()
	t.Cleanup(func() { _ = pool.Close() })
	if err := database.AutoMigrate(&models.User{}, &models.Engagement{}, &models.EngagementDetail{},
		&media.Media{}, &modelutils.FileMetadata{}, &post.Post{}, &modelutils.Location{}); err != nil {
		t.Fatal(err)
	}
	tx := database.Begin()
	if tx.Error != nil {
		t.Fatalf("begin test transaction: %v", tx.Error)
	}
	t.Cleanup(func() { tx.Rollback() })
	return tx
}

func TestMigrateLegacyChatMediaOwnershipIsIdempotentAndPreservesFile(t *testing.T) {
	database := migrationIntegrationDB(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	userID := uuid.New()
	chatID := uuid.New()
	messageID := uuid.New()
	contentableType := string(post.PostKindChat)
	user := models.User{
		ID: userID, PublicID: now.UnixNano(), Domain: models.CoolVibes,
		UserName: "media-migration-" + uuid.NewString(), DisplayName: "Migration", UserRole: constants.UserRoleUser,
	}
	if err := database.Omit(clause.Associations).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	message := post.Post{
		ID: messageID, PublicID: now.UnixNano() + 1, PostKind: post.PostKindMessage,
		Domain: models.CoolVibes, ContentableID: &chatID, ContentableType: &contentableType,
		AuthorID: userID, Published: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := database.Omit(clause.Associations).Create(&message).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}

	physicalPath := filepath.Join(t.TempDir(), "legacy-chat.jpg")
	if err := os.WriteFile(physicalPath, []byte("legacy-image"), 0o600); err != nil {
		t.Fatal(err)
	}
	fileID := uuid.New()
	file := modelutils.FileMetadata{
		ID: fileID, URL: "/static/legacy-chat.jpg", StoragePath: physicalPath,
		MimeType: "image/jpeg", Size: 12, Name: "legacy-chat.jpg", CreatedAt: now,
	}
	legacy := media.Media{
		ID: uuid.New(), PublicID: now.UnixNano() + 2, FileID: fileID,
		OwnerID: messageID, OwnerType: media.OwnerChat, UserID: userID,
		Role: media.RoleChatMedia, ProcessingStatus: media.ProcessingStatusReady,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := database.Omit(clause.Associations).Create(&file).Error; err != nil {
		t.Fatalf("create file metadata: %v", err)
	}
	if err := database.Omit(clause.Associations).Create(&legacy).Error; err != nil {
		t.Fatalf("create legacy media: %v", err)
	}

	if err := MigrateLegacyChatMediaOwnership(database); err != nil {
		t.Fatalf("first migration: %v", err)
	}
	if err := MigrateLegacyChatMediaOwnership(database); err != nil {
		t.Fatalf("second migration: %v", err)
	}
	var migrated media.Media
	if err := database.First(&migrated, "id = ?", legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if migrated.OwnerType != media.OwnerPost || migrated.OwnerID != messageID {
		t.Fatalf("unexpected migrated ownership: type=%q id=%s", migrated.OwnerType, migrated.OwnerID)
	}
	var fileCount int64
	if err := database.Model(&modelutils.FileMetadata{}).Where("id = ? AND storage_path = ?", fileID, physicalPath).Count(&fileCount).Error; err != nil || fileCount != 1 {
		t.Fatalf("file metadata changed, count=%d error=%v", fileCount, err)
	}
	if data, err := os.ReadFile(physicalPath); err != nil || string(data) != "legacy-image" {
		t.Fatalf("physical file changed, data=%q error=%v", data, err)
	}
}
