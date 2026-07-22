package repositories

import (
	"context"
	modelutils "core/models/utils"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestUpsertLocationUsesOneAtomicPartialConflictStatement(t *testing.T) {
	database := newDryRunTaxonomyDB(t).Session(&gorm.Session{SkipDefaultTransaction: true})
	location := &modelutils.Location{
		ContentableType: modelutils.LocationOwnerUser,
		ContentableID:   uuid.New(),
	}

	result := upsertLocation(database, location)
	if result.Error != nil {
		t.Fatalf("upsertLocation() error = %v", result.Error)
	}

	sql := strings.ToLower(strings.Join(strings.Fields(result.Statement.SQL.String()), " "))
	for _, required := range []string{
		`insert into "locations"`,
		`on conflict ("contentable_type","contentable_id") where deleted_at is null do update`,
		`returning "id"`,
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("location upsert SQL is missing %q: %s", required, result.Statement.SQL.String())
		}
	}
	if strings.Contains(sql, "select ") {
		t.Fatalf("location upsert must not use a read-before-write query: %s", result.Statement.SQL.String())
	}
}

func TestUpsertLocationRejectsNilInput(t *testing.T) {
	database := newDryRunTaxonomyDB(t).Session(&gorm.Session{SkipDefaultTransaction: true})
	if result := upsertLocation(database, nil); result.Error == nil {
		t.Fatal("upsertLocation(nil) error = nil")
	}
}

func TestUpsertLocationConcurrentWritersKeepOneActiveOwnerIntegration(t *testing.T) {
	database := locationRaceIntegrationDB(t)
	if !database.Migrator().HasTable(&modelutils.Location{}) {
		t.Skip("locations table is not migrated in TEST_DATABASE_URL")
	}

	var indexName sql.NullString
	if err := database.Raw("SELECT to_regclass('public.uidx_locations_active_owner')").Scan(&indexName).Error; err != nil {
		t.Fatalf("check location owner index: %v", err)
	}
	if !indexName.Valid {
		t.Skip("uidx_locations_active_owner is not migrated in TEST_DATABASE_URL")
	}

	ownerID := uuid.New()
	cleanup := func() {
		database.Unscoped().Where("contentable_type = ? AND contentable_id = ?", modelutils.LocationOwnerUser, ownerID).
			Delete(&modelutils.Location{})
	}
	cleanup()
	t.Cleanup(cleanup)

	const writerCount = 16
	errorsByWriter := make(chan error, writerCount)
	var writers sync.WaitGroup
	for writer := 0; writer < writerCount; writer++ {
		writer := writer
		writers.Add(1)
		go func() {
			defer writers.Done()
			city := fmt.Sprintf("city-%d", writer)
			location := &modelutils.Location{
				ContentableType: modelutils.LocationOwnerUser,
				ContentableID:   ownerID,
				City:            &city,
			}
			errorsByWriter <- upsertLocation(database.WithContext(context.Background()), location).Error
		}()
	}
	writers.Wait()
	close(errorsByWriter)
	for err := range errorsByWriter {
		if err != nil {
			t.Fatalf("concurrent upsert error = %v", err)
		}
	}

	var activeCount int64
	if err := database.Model(&modelutils.Location{}).
		Where("contentable_type = ? AND contentable_id = ?", modelutils.LocationOwnerUser, ownerID).
		Count(&activeCount).Error; err != nil {
		t.Fatalf("count active locations: %v", err)
	}
	if activeCount != 1 {
		t.Fatalf("active locations after %d writers = %d, want 1", writerCount, activeCount)
	}
}

func locationRaceIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" && os.Getenv("ENV") == "test" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open location race database: %v", err)
	}
	return database
}
