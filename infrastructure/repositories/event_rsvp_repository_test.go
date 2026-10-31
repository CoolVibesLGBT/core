package repositories

import (
	"context"
	"core/constants"
	"core/models"
	"core/models/post"
	postpayloads "core/models/post/payloads"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestEventRSVPUsesEventLevelRowLock(t *testing.T) {
	db := newDryRunTaxonomyDB(t).Session(&gorm.Session{SkipDefaultTransaction: true})
	tx := lockEventRSVP(db, 123)
	if tx.Error != nil {
		t.Fatalf("lockEventRSVP() error = %v", tx.Error)
	}
	sql := strings.ToLower(tx.Statement.SQL.String())
	if !strings.Contains(sql, "transaction_locks") || !strings.Contains(sql, "for update") {
		t.Fatalf("RSVP write is not event-serialized: %s", tx.Statement.SQL.String())
	}
}

func eventRSVPIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := privatePhotoIntegrationDB(t)
	if err := db.AutoMigrate(&post.Post{}, &postpayloads.Event{}, &postpayloads.EventAttendee{}, &models.ReportKind{}, &models.Report{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestSetEventRSVPTogglesAndCounts(t *testing.T) {
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

	result, err = repo.SetEventRSVP(context.Background(), eventPost.PublicID, viewer.ID, &going)
	if err != nil || result.Status == nil || *result.Status != going || result.Counts.Going != 1 {
		t.Fatalf("SetEventRSVP(re-RSVP going) = %#v, %v", result, err)
	}
	var rowCount int64
	if err := db.Model(&postpayloads.EventAttendee{}).Where("event_id = ? AND user_id = ?", event.ID, viewer.ID).Count(&rowCount).Error; err != nil {
		t.Fatalf("count RSVPs: %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("RSVP rows = %d; want 1", rowCount)
	}
}

// TestSetEventRSVPRejectsDuplicateAttendeeRows locks in the guarantee the
// old dedup-on-every-write loop used to provide manually: uidx_event_attendee_user
// now makes a second (event, user) row impossible at the database level.
func TestSetEventRSVPRejectsDuplicateAttendeeRows(t *testing.T) {
	db := eventRSVPIntegrationDB(t)
	basePublicID := time.Now().UTC().UnixNano()
	viewer := models.User{
		ID: uuid.New(), PublicID: basePublicID, Domain: models.CoolVibes,
		UserName: "rsvp-dup-" + uuid.NewString(), DisplayName: "RSVP Dup", UserRole: constants.UserRoleUser,
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

	first := postpayloads.EventAttendee{ID: uuid.New(), EventID: event.ID, UserID: viewer.ID, Status: postpayloads.EventAttendanceMaybe, JoinedAt: time.Now().UTC()}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first attendee row: %v", err)
	}
	second := postpayloads.EventAttendee{ID: uuid.New(), EventID: event.ID, UserID: viewer.ID, Status: postpayloads.EventAttendanceGoing, JoinedAt: time.Now().UTC()}
	if err := db.Create(&second).Error; err == nil {
		t.Fatal("database accepted a second attendee row for the same (event, user)")
	}
}
