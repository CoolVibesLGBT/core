package repositories

import (
	"context"
	"core/constants"
	"core/models"
	"core/models/post"
	postpayloads "core/models/post/payloads"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestEventAttendeePreloadSelectsOnlySafeProfileProjection(t *testing.T) {
	db := newDryRunTaxonomyDB(t).Session(&gorm.Session{SkipDefaultTransaction: true})
	var attendees []postpayloads.EventAttendee
	tx := preloadEventAttendees(db).Find(&attendees)
	if tx.Error != nil {
		t.Fatalf("preloadEventAttendees() error = %v", tx.Error)
	}
	sql := strings.ToLower(tx.Statement.SQL.String())
	for _, fragment := range []string{"users.public_id", "users.user_name", "users.display_name", "avatar_url", "left join users"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("safe attendee projection missing %q: %s", fragment, tx.Statement.SQL.String())
		}
	}
	for _, forbidden := range []string{"users.email", "users.password"} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("attendee projection leaks %q: %s", forbidden, tx.Statement.SQL.String())
		}
	}
}

func TestEventRSVPUsesEventLevelAdvisoryLock(t *testing.T) {
	db := newDryRunTaxonomyDB(t).Session(&gorm.Session{SkipDefaultTransaction: true})
	tx := lockEventRSVP(db, 123)
	if tx.Error != nil {
		t.Fatalf("lockEventRSVP() error = %v", tx.Error)
	}
	sql := strings.ToLower(tx.Statement.SQL.String())
	if !strings.Contains(sql, "pg_advisory_xact_lock") || !strings.Contains(sql, "hashtextextended") {
		t.Fatalf("RSVP write is not event-serialized: %s", tx.Statement.SQL.String())
	}
}

func eventRSVPIntegrationDB(t *testing.T) *gorm.DB {
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

func TestSetEventRSVPChangesTogglesRepairsDuplicatesAndCounts(t *testing.T) {
	db := eventRSVPIntegrationDB(t)
	if err := db.AutoMigrate(&postpayloads.Event{}, &postpayloads.EventAttendee{}); err != nil {
		t.Fatalf("migrate event RSVP schema: %v", err)
	}

	basePublicID := time.Now().UTC().UnixNano()
	viewer := models.User{
		ID: uuid.New(), PublicID: basePublicID, Domain: models.CoolVibes,
		UserName: "rsvp-viewer-" + uuid.NewString(), DisplayName: "RSVP Viewer", UserRole: constants.UserRoleUser,
	}
	if err := db.Omit(clause.Associations).Create(&viewer).Error; err != nil {
		t.Fatalf("create RSVP user: %v", err)
	}
	contentableType := string(post.PostKindEvent)
	eventPost := post.Post{
		ID: uuid.New(), PublicID: basePublicID + 1, AuthorID: viewer.ID, Domain: models.CoolVibes,
		PostKind: post.PostKindEvent, ContentableType: &contentableType, Published: true,
	}
	if err := db.Omit(clause.Associations).Create(&eventPost).Error; err != nil {
		t.Fatalf("create event post: %v", err)
	}
	event := postpayloads.Event{ID: uuid.New(), PostID: eventPost.ID, Status: "scheduled"}
	if err := db.Omit(clause.Associations).Create(&event).Error; err != nil {
		t.Fatalf("create event: %v", err)
	}

	repo := &PostRepository{db: db}
	going := postpayloads.EventAttendanceGoing
	result, err := repo.SetEventRSVP(context.Background(), eventPost.PublicID, viewer.ID, &going)
	if err != nil || result.Status == nil || *result.Status != going {
		t.Fatalf("SetEventRSVP(going) = %#v, %v", result, err)
	}
	if result.Counts.Going != 1 || len(result.Attendees) != 1 || result.Attendees[0].DisplayName != viewer.DisplayName {
		t.Fatalf("unexpected going result: %#v", result)
	}

	maybe := postpayloads.EventAttendanceMaybe
	result, err = repo.SetEventRSVP(context.Background(), eventPost.PublicID, viewer.ID, &maybe)
	if err != nil || result.Status == nil || *result.Status != maybe || result.Counts.Maybe != 1 || result.Counts.Going != 0 {
		t.Fatalf("SetEventRSVP(maybe) = %#v, %v", result, err)
	}
	result, err = repo.SetEventRSVP(context.Background(), eventPost.PublicID, viewer.ID, &maybe)
	if err != nil || result.Status != nil || len(result.Attendees) != 0 || result.Counts.Maybe != 0 {
		t.Fatalf("SetEventRSVP(toggle clear) = %#v, %v", result, err)
	}

	now := time.Now().UTC()
	duplicates := []postpayloads.EventAttendee{
		{ID: uuid.New(), EventID: event.ID, UserID: viewer.ID, Status: postpayloads.EventAttendanceMaybe, JoinedAt: now},
		{ID: uuid.New(), EventID: event.ID, UserID: viewer.ID, Status: postpayloads.EventAttendanceNotGoing, JoinedAt: now.Add(-time.Minute)},
	}
	if err := db.Create(&duplicates).Error; err != nil {
		t.Fatalf("create historical duplicate RSVPs: %v", err)
	}
	result, err = repo.SetEventRSVP(context.Background(), eventPost.PublicID, viewer.ID, &going)
	if err != nil || result.Status == nil || *result.Status != going || result.Counts.Going != 1 {
		t.Fatalf("SetEventRSVP(repair duplicate) = %#v, %v", result, err)
	}
	var rowCount int64
	if err := db.Model(&postpayloads.EventAttendee{}).Where("event_id = ? AND user_id = ?", event.ID, viewer.ID).Count(&rowCount).Error; err != nil {
		t.Fatalf("count repaired RSVPs: %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("RSVP rows after repair = %d; want 1", rowCount)
	}
}
