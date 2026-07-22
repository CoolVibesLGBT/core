package repositories

import (
	"context"
	"core/constants"
	"core/models"
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestViewWriteSQLIsConflictSafeAndAtomic(t *testing.T) {
	db := newDryRunTaxonomyDB(t).Session(&gorm.Session{SkipDefaultTransaction: true})
	lock := lockViewAggregate(db, "engagement:post:"+uuid.NewString())
	if lock.Error != nil {
		t.Fatalf("lockViewAggregate() error = %v", lock.Error)
	}
	lockSQL := strings.ToLower(lock.Statement.SQL.String())
	if !strings.Contains(lockSQL, "pg_advisory_xact_lock") || !strings.Contains(lockSQL, "hashtextextended") {
		t.Fatalf("view aggregate creation is not transaction-locked: %s", lock.Statement.SQL.String())
	}

	dedupeKey := "view:post:" + uuid.NewString() + ":" + uuid.NewString()
	detail := models.EngagementDetail{
		ID:           uuid.New(),
		EngagementID: uuid.New(),
		DedupeKey:    &dedupeKey,
		EngagerID:    uuid.New(),
		EngageeID:    uuid.New(),
		Kind:         models.EngagementKindView,
	}

	insert := insertViewDetailOnce(db, &detail)
	if insert.Error != nil {
		t.Fatalf("insertViewDetailOnce() error = %v", insert.Error)
	}
	insertSQL := strings.ToUpper(insert.Statement.SQL.String())
	if !strings.Contains(insertSQL, `ON CONFLICT ("DEDUPE_KEY") DO NOTHING`) {
		t.Fatalf("view detail insert is not conflict-safe: %s", insert.Statement.SQL.String())
	}

	update := incrementViewAggregate(db, uuid.New(), "view_count")
	if update.Error != nil {
		t.Fatalf("incrementViewAggregate() error = %v", update.Error)
	}
	updateSQL := strings.ToLower(update.Statement.SQL.String())
	for _, fragment := range []string{"jsonb_set", "counts ->>", "+ 1"} {
		if !strings.Contains(updateSQL, fragment) {
			t.Fatalf("view aggregate update must atomically increment JSONB count; missing %q in %s", fragment, update.Statement.SQL.String())
		}
	}
}

func TestEngagementAggregateLockKeyIsCanonicalForEveryWriteKind(t *testing.T) {
	contentableID := uuid.New()
	got := engagementAggregateLockKey(models.EngagementContentableTypePost, contentableID)
	want := "engagement:post:" + contentableID.String()
	if got != want {
		t.Fatalf("engagementAggregateLockKey() = %q, want %q", got, want)
	}
}

func TestPublicPostEngagementDetailsExcludeViewerIdentities(t *testing.T) {
	db := newDryRunTaxonomyDB(t).Session(&gorm.Session{SkipDefaultTransaction: true})
	var details []models.EngagementDetail
	query := preloadPublicEngagementDetails(db).Find(&details)
	if query.Error != nil {
		t.Fatalf("preloadPublicEngagementDetails() error = %v", query.Error)
	}

	sql := strings.ToLower(query.Statement.SQL.String())
	if !strings.Contains(sql, "kind <>") {
		t.Fatalf("public engagement preload does not filter view details: %s", query.Statement.SQL.String())
	}
	if len(query.Statement.Vars) != 1 || query.Statement.Vars[0] != models.EngagementKindView {
		t.Fatalf("public engagement preload filter vars = %#v; want view", query.Statement.Vars)
	}
}

func TestRecordViewOnceSkipsSelfBeforeDatabaseAccess(t *testing.T) {
	userID := uuid.New()
	repo := NewEngagementRepository(nil)
	counted, err := repo.RecordViewOnce(context.Background(), userID, userID, models.EngagementKindViewReceived, userID, models.EngagementContentableTypeUser)
	if err != nil || counted {
		t.Fatalf("RecordViewOnce(self) = %v, %v; want false, nil", counted, err)
	}
}

func engagementViewIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" && os.Getenv("ENV") == "test" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin test transaction: %v", tx.Error)
	}
	t.Cleanup(func() { tx.Rollback() })
	return tx
}

func TestRecordViewOnceIsIdempotentIntegration(t *testing.T) {
	db := engagementViewIntegrationDB(t)
	if err := db.AutoMigrate(&models.Engagement{}, &models.EngagementDetail{}); err != nil {
		t.Fatalf("migrate engagement view schema: %v", err)
	}

	basePublicID := time.Now().UTC().UnixNano()
	viewer := models.User{
		ID: uuid.New(), PublicID: basePublicID, Domain: models.CoolVibes,
		UserName: "view-test-viewer-" + uuid.NewString(), DisplayName: "Viewer", UserRole: constants.UserRoleUser,
	}
	target := models.User{
		ID: uuid.New(), PublicID: basePublicID + 1, Domain: models.CoolVibes,
		UserName: "view-test-target-" + uuid.NewString(), DisplayName: "Target", UserRole: constants.UserRoleUser,
	}
	if err := db.Omit(clause.Associations).Create(&[]models.User{viewer, target}).Error; err != nil {
		t.Fatalf("create view test users: %v", err)
	}

	repo := NewEngagementRepository(db)
	first, err := repo.RecordViewOnce(context.Background(), viewer.ID, target.ID, models.EngagementKindViewReceived, target.ID, models.EngagementContentableTypeUser)
	if err != nil || !first {
		t.Fatalf("first RecordViewOnce() = %v, %v; want true, nil", first, err)
	}
	second, err := repo.RecordViewOnce(context.Background(), viewer.ID, target.ID, models.EngagementKindViewReceived, target.ID, models.EngagementContentableTypeUser)
	if err != nil || second {
		t.Fatalf("second RecordViewOnce() = %v, %v; want false, nil", second, err)
	}

	var aggregate models.Engagement
	if err := db.Where("contentable_id = ? AND contentable_type = ?", target.ID, models.EngagementContentableTypeUser).First(&aggregate).Error; err != nil {
		t.Fatalf("load profile view aggregate: %v", err)
	}
	var counts map[string]interface{}
	if err := json.Unmarshal(aggregate.Counts, &counts); err != nil {
		t.Fatalf("decode aggregate counts: %v", err)
	}
	if got := counts["view_received_count"]; got != float64(1) {
		t.Fatalf("view_received_count = %#v; want 1", got)
	}

	var detailCount int64
	if err := db.Model(&models.EngagementDetail{}).
		Where("engagement_id = ? AND kind = ?", aggregate.ID, models.EngagementKindViewReceived).
		Count(&detailCount).Error; err != nil {
		t.Fatalf("count view details: %v", err)
	}
	if detailCount != 1 {
		t.Fatalf("view detail count = %d; want 1", detailCount)
	}
}

func TestFirstViewAndFirstAmountWriteShareOneAggregateIntegration(t *testing.T) {
	db := locationRaceIntegrationDB(t)
	if !db.Migrator().HasTable(&models.User{}) || !db.Migrator().HasTable(&models.Engagement{}) {
		t.Skip("engagement schema is not migrated in TEST_DATABASE_URL")
	}
	var indexName sql.NullString
	if err := db.Raw("SELECT to_regclass('public.uidx_engagements_contentable')").Scan(&indexName).Error; err != nil {
		t.Fatalf("check engagement aggregate index: %v", err)
	}
	if !indexName.Valid {
		t.Skip("uidx_engagements_contentable is not migrated in TEST_DATABASE_URL")
	}

	basePublicID := time.Now().UTC().UnixNano()
	viewer := models.User{
		ID: uuid.New(), PublicID: basePublicID, Domain: models.CoolVibes,
		UserName: "mixed-viewer-" + uuid.NewString(), DisplayName: "Viewer", UserRole: constants.UserRoleUser,
	}
	target := models.User{
		ID: uuid.New(), PublicID: basePublicID + 1, Domain: models.CoolVibes,
		UserName: "mixed-target-" + uuid.NewString(), DisplayName: "Target", UserRole: constants.UserRoleUser,
	}
	if err := db.Omit(clause.Associations).Create(&[]models.User{viewer, target}).Error; err != nil {
		t.Fatalf("create mixed engagement users: %v", err)
	}
	t.Cleanup(func() {
		var aggregateIDs []uuid.UUID
		db.Model(&models.Engagement{}).
			Where("contentable_id = ? AND contentable_type = ?", target.ID, models.EngagementContentableTypeUser).
			Pluck("id", &aggregateIDs)
		if len(aggregateIDs) > 0 {
			db.Where("engagement_id IN ?", aggregateIDs).Delete(&models.EngagementDetail{})
			db.Where("id IN ?", aggregateIDs).Delete(&models.Engagement{})
		}
		db.Unscoped().Where("id IN ?", []uuid.UUID{viewer.ID, target.ID}).Delete(&models.User{})
	})

	repo := NewEngagementRepository(db)
	errorsByWriter := make(chan error, 2)
	var writers sync.WaitGroup
	writers.Add(2)
	go func() {
		defer writers.Done()
		_, err := repo.RecordViewOnce(context.Background(), viewer.ID, target.ID, models.EngagementKindViewReceived, target.ID, models.EngagementContentableTypeUser)
		errorsByWriter <- err
	}()
	go func() {
		defer writers.Done()
		errorsByWriter <- repo.AddTip(context.Background(), viewer.ID, target.ID, decimal.NewFromInt(1), target.ID, models.EngagementContentableTypeUser, models.EngagementKindTip)
	}()
	writers.Wait()
	close(errorsByWriter)
	for err := range errorsByWriter {
		if err != nil {
			t.Fatalf("mixed aggregate writer error = %v", err)
		}
	}

	var aggregateCount int64
	if err := db.Model(&models.Engagement{}).
		Where("contentable_id = ? AND contentable_type = ?", target.ID, models.EngagementContentableTypeUser).
		Count(&aggregateCount).Error; err != nil {
		t.Fatalf("count mixed aggregates: %v", err)
	}
	if aggregateCount != 1 {
		t.Fatalf("first view + amount created %d aggregates, want 1", aggregateCount)
	}
}
