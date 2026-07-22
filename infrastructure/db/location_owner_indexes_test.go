package db

import (
	modelutils "core/models/utils"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestLocationOwnerIndexDefinitionIsUniqueAndSoftDeleteAware(t *testing.T) {
	definitions := locationOwnerIndexDefinitions()
	if len(definitions) != 1 {
		t.Fatalf("location owner index count = %d, want 1", len(definitions))
	}

	definition := definitions[0]
	if definition.Name != "uidx_locations_active_owner" || definition.Table != "locations" || !definition.Unique {
		t.Fatalf("unexpected location owner index: %#v", definition)
	}
	if strings.Join(definition.Columns, ",") != "contentable_type,contentable_id" {
		t.Fatalf("location owner index columns = %#v", definition.Columns)
	}
	if definition.Condition != "deleted_at IS NULL" {
		t.Fatalf("location owner index condition = %q", definition.Condition)
	}
}

func TestDiscoveryIndexesIncludeSharedSpatialAndUserCursorIndexes(t *testing.T) {
	definitions := discoveryIndexDefinitions()
	if len(definitions) != 2 {
		t.Fatalf("discovery index count = %d, want 2", len(definitions))
	}

	spatial := definitions[0]
	if spatial.Name != "idx_locations_active_point_gist" || spatial.Table != "locations" || spatial.Using != "gist" {
		t.Fatalf("unexpected spatial discovery index: %#v", spatial)
	}
	if strings.Join(spatial.Columns, ",") != "location_point" || spatial.Condition != "deleted_at IS NULL AND location_point IS NOT NULL" {
		t.Fatalf("unexpected spatial discovery predicate: %#v", spatial)
	}

	users := definitions[1]
	if users.Name != "idx_users_active_domain_public_id" || strings.Join(users.Columns, ",") != "domain,public_id" {
		t.Fatalf("unexpected user discovery index: %#v", users)
	}
	if users.Condition != "deleted_at IS NULL" {
		t.Fatalf("unexpected user discovery predicate: %#v", users)
	}
}

func TestMigrateLocationOwnerUniquenessSafelyDeduplicatesAndIsIdempotentIntegration(t *testing.T) {
	database := migrationIntegrationDB(t).Session(&gorm.Session{SkipDefaultTransaction: true})
	if !database.Migrator().HasTable(&modelutils.Location{}) {
		t.Skip("locations table is not migrated in TEST_DATABASE_URL")
	}

	// The index may already exist in a migrated test database. Drop it only in
	// this rollback-only test transaction so the historical duplicate fixture
	// can be installed safely.
	if err := database.Exec("DROP INDEX IF EXISTS uidx_locations_active_owner").Error; err != nil {
		t.Fatalf("drop location owner index fixture: %v", err)
	}

	ownerID := uuid.New()
	baseTime := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	older := modelutils.Location{
		ID:              uuid.New(),
		ContentableType: modelutils.LocationOwnerUser,
		ContentableID:   ownerID,
		CreatedAt:       baseTime,
		UpdatedAt:       baseTime,
	}
	newer := modelutils.Location{
		ID:              uuid.New(),
		ContentableType: modelutils.LocationOwnerUser,
		ContentableID:   ownerID,
		CreatedAt:       baseTime.Add(time.Minute),
		UpdatedAt:       baseTime.Add(time.Minute),
	}
	if err := database.Create(&older).Error; err != nil {
		t.Fatalf("create older duplicate location: %v", err)
	}
	if err := database.Create(&newer).Error; err != nil {
		t.Fatalf("create newer duplicate location: %v", err)
	}

	if err := MigrateLocationOwnerUniqueness(database); err != nil {
		t.Fatalf("first location owner migration: %v", err)
	}
	if err := MigrateLocationOwnerUniqueness(database); err != nil {
		t.Fatalf("idempotent location owner migration: %v", err)
	}

	var active []modelutils.Location
	if err := database.Where("contentable_type = ? AND contentable_id = ?", modelutils.LocationOwnerUser, ownerID).Find(&active).Error; err != nil {
		t.Fatalf("load active locations: %v", err)
	}
	if len(active) != 1 || active[0].ID != newer.ID {
		t.Fatalf("active locations = %#v, want newest id %s", active, newer.ID)
	}

	var total int64
	if err := database.Unscoped().Model(&modelutils.Location{}).
		Where("contentable_type = ? AND contentable_id = ?", modelutils.LocationOwnerUser, ownerID).
		Count(&total).Error; err != nil {
		t.Fatalf("count all locations: %v", err)
	}
	if total != 2 {
		t.Fatalf("all location rows = %d, want 2 retained rows", total)
	}
}
