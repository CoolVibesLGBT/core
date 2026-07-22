package repositories

import (
	"context"
	"core/application/ports"
	"core/constants"
	domainmoderation "core/domain/moderation"
	"core/models"
	"core/models/post"
	modelutils "core/models/utils"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestModerationWritesUseSharedTargetAdvisoryLock(t *testing.T) {
	db := newDryRunTaxonomyDB(t).Session(&gorm.Session{SkipDefaultTransaction: true})
	targetID := uuid.New()
	tx := lockModerationTarget(db, models.EngagementContentableTypePost, targetID)
	if tx.Error != nil {
		t.Fatalf("moderation target lock error = %v", tx.Error)
	}
	lockSQL := strings.ToLower(tx.Statement.SQL.String())
	if !strings.Contains(lockSQL, "pg_advisory_xact_lock") || !strings.Contains(lockSQL, "hashtextextended") {
		t.Fatalf("moderation write is not target-serialized: %s", tx.Statement.SQL.String())
	}
	if moderationTargetLockKey(models.EngagementContentableTypePost, targetID) == moderationTargetLockKey(models.EngagementContentableTypeUser, targetID) {
		t.Fatal("moderation lock key must include target type")
	}
}

func TestModerationQueueQuerySupportsMixedAndUserTargets(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := NewModerationRepository(db)

	var reports []models.Report
	mixed := repo.reportQueueQuery(context.Background(), ports.ModerationReportFilter{Status: domainmoderation.StatusPending}).Find(&reports)
	if mixed.Error != nil {
		t.Fatalf("mixed query error = %v", mixed.Error)
	}
	mixedSQL := strings.ToLower(mixed.Statement.SQL.String())
	if strings.Contains(mixedSQL, "contentable_type =") {
		t.Fatalf("unfiltered queue should include multiple target types: %s", mixed.Statement.SQL.String())
	}
	if !strings.Contains(mixedSQL, "reports.status") {
		t.Fatalf("queue status filter missing: %s", mixed.Statement.SQL.String())
	}

	userQuery := repo.reportQueueQuery(context.Background(), ports.ModerationReportFilter{
		ContentableType: domainmoderation.TargetUser,
		UserPublicID:    77,
	}).Find(&reports)
	if userQuery.Error != nil {
		t.Fatalf("user query error = %v", userQuery.Error)
	}
	userSQL := strings.ToLower(userQuery.Statement.SQL.String())
	for _, fragment := range []string{"moderation_targets", "reports.contentable_type", "moderation_targets.public_id"} {
		if !strings.Contains(userSQL, fragment) {
			t.Fatalf("user queue query missing %q: %s", fragment, userQuery.Statement.SQL.String())
		}
	}
}

func TestMapModerationViewPreservesModelJSONContract(t *testing.T) {
	reportID := uuid.New()
	reporterID := uuid.New()
	reportModel := models.Report{
		ID:              reportID,
		ContentableID:   uuid.New(),
		ContentableType: models.EngagementContentableTypePost,
		ReporterID:      reporterID,
		Reporter: &models.User{
			ID:          reporterID,
			PublicID:    99,
			UserName:    "reporter",
			DisplayName: "Reporter",
		},
		ReportKindKey: models.ReportKindSpam,
		ReportKind: &models.ReportKind{
			Key:          models.ReportKindSpam,
			DisplayOrder: 1,
			Name:         modelutils.LocalizedString{"en": "Spam"},
			Description:  modelutils.LocalizedString{"en": "Unwanted content"},
		},
		Reason: "spam details",
		Status: models.ReportStatusPending,
	}

	reportView, err := mapModerationReportView(reportModel)
	if err != nil {
		t.Fatalf("map report view: %v", err)
	}
	reportJSON, err := json.Marshal(reportView)
	if err != nil {
		t.Fatalf("marshal report view: %v", err)
	}
	var reportPayload struct {
		ID              string `json:"id"`
		ContentableType string `json:"contentable_type"`
		Status          string `json:"status"`
		Reporter        struct {
			PublicID string `json:"public_id"`
			UserName string `json:"username"`
		} `json:"reporter"`
	}
	if err := json.Unmarshal(reportJSON, &reportPayload); err != nil {
		t.Fatalf("decode report view: %v", err)
	}
	if reportPayload.ID != reportID.String() || reportPayload.ContentableType != string(domainmoderation.TargetPost) || reportPayload.Status != string(domainmoderation.StatusPending) {
		t.Fatalf("report payload = %#v", reportPayload)
	}
	if reportPayload.Reporter.PublicID != "99" || reportPayload.Reporter.UserName != "reporter" {
		t.Fatalf("reporter payload = %#v", reportPayload.Reporter)
	}
	assertJSONEquivalent(t, reportModel, reportView)

	postModel := post.Post{ID: uuid.New(), PublicID: 42, Published: true}
	postView, err := mapModerationView[ports.ModerationPostView](postModel)
	if err != nil {
		t.Fatalf("map post view: %v", err)
	}
	postJSON, err := json.Marshal(postView)
	if err != nil {
		t.Fatalf("marshal post view: %v", err)
	}
	var postPayload struct {
		PublicID  string `json:"public_id"`
		Published bool   `json:"published"`
	}
	if err := json.Unmarshal(postJSON, &postPayload); err != nil {
		t.Fatalf("decode post view: %v", err)
	}
	if postPayload.PublicID != "42" || !postPayload.Published {
		t.Fatalf("post payload = %#v", postPayload)
	}
	assertJSONEquivalent(t, postModel, postView)
}

func TestResolveReportSameStatusStillAppliesVisibilityAndResolutionIntegration(t *testing.T) {
	db := eventRSVPIntegrationDB(t)
	for _, table := range []any{&models.Report{}, &models.ReportKind{}, &post.Post{}, &models.User{}} {
		if !db.Migrator().HasTable(table) {
			t.Skip("moderation schema is not migrated in TEST_DATABASE_URL")
		}
	}

	basePublicID := time.Now().UTC().UnixNano()
	author := models.User{
		ID: uuid.New(), PublicID: basePublicID, Domain: models.CoolVibes,
		UserName: "moderation-author-" + uuid.NewString(), DisplayName: "Author", UserRole: constants.UserRoleUser,
	}
	reporter := models.User{
		ID: uuid.New(), PublicID: basePublicID + 1, Domain: models.CoolVibes,
		UserName: "moderation-reporter-" + uuid.NewString(), DisplayName: "Reporter", UserRole: constants.UserRoleUser,
	}
	moderator := models.User{
		ID: uuid.New(), PublicID: basePublicID + 2, Domain: models.CoolVibes,
		UserName: "moderation-reviewer-" + uuid.NewString(), DisplayName: "Reviewer", UserRole: constants.UserRoleModerator,
	}
	if err := db.Omit(clause.Associations).Create(&[]models.User{author, reporter, moderator}).Error; err != nil {
		t.Fatalf("create moderation users: %v", err)
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.ReportKind{Key: models.ReportKindSpam}).Error; err != nil {
		t.Fatalf("ensure report kind: %v", err)
	}
	now := time.Now().UTC()
	reportedPost := post.Post{
		ID: uuid.New(), PublicID: basePublicID + 3, AuthorID: author.ID, Domain: models.CoolVibes,
		PostKind: post.PostKindPost, Published: true, PublishedAt: &now,
	}
	if err := db.Omit(clause.Associations).Create(&reportedPost).Error; err != nil {
		t.Fatalf("create reported post: %v", err)
	}
	report := models.Report{
		ID: uuid.New(), ContentableID: reportedPost.ID, ContentableType: models.EngagementContentableTypePost,
		ReporterID: reporter.ID, ReportKindKey: models.ReportKindSpam, Status: models.ReportStatusActioned,
	}
	if err := db.Omit(clause.Associations).Create(&report).Error; err != nil {
		t.Fatalf("create actioned report: %v", err)
	}

	hide := false
	view, err := NewModerationRepository(db).ResolveReport(context.Background(), ports.ModerationResolveInput{
		ReportID: report.ID, Status: domainmoderation.StatusActioned, ReviewedByID: moderator.ID,
		Resolution: "hidden on idempotent retry", PublishPost: &hide,
	})
	if err != nil {
		t.Fatalf("ResolveReport(same status) error = %v", err)
	}
	if view.Status != domainmoderation.StatusActioned || view.Resolution != "hidden on idempotent retry" {
		t.Fatalf("resolved report = %#v", view)
	}

	var persistedPost post.Post
	if err := db.First(&persistedPost, "id = ?", reportedPost.ID).Error; err != nil {
		t.Fatalf("reload reported post: %v", err)
	}
	if persistedPost.Published || persistedPost.PublishedAt != nil {
		t.Fatalf("same-status resolution did not hide post: %#v", persistedPost)
	}
}

func TestConcurrentResolveAndHideShareTargetLockIntegration(t *testing.T) {
	db := locationRaceIntegrationDB(t)
	for _, table := range []any{&models.Report{}, &models.ReportKind{}, &post.Post{}, &models.User{}} {
		if !db.Migrator().HasTable(table) {
			t.Skip("moderation schema is not migrated in TEST_DATABASE_URL")
		}
	}

	basePublicID := time.Now().UTC().UnixNano()
	author := models.User{ID: uuid.New(), PublicID: basePublicID, Domain: models.CoolVibes, UserName: "moderation-race-author-" + uuid.NewString(), DisplayName: "Author", UserRole: constants.UserRoleUser}
	reporter := models.User{ID: uuid.New(), PublicID: basePublicID + 1, Domain: models.CoolVibes, UserName: "moderation-race-reporter-" + uuid.NewString(), DisplayName: "Reporter", UserRole: constants.UserRoleUser}
	moderator := models.User{ID: uuid.New(), PublicID: basePublicID + 2, Domain: models.CoolVibes, UserName: "moderation-race-reviewer-" + uuid.NewString(), DisplayName: "Reviewer", UserRole: constants.UserRoleModerator}
	users := []models.User{author, reporter, moderator}
	if err := db.Omit(clause.Associations).Create(&users).Error; err != nil {
		t.Fatalf("create moderation race users: %v", err)
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.ReportKind{Key: models.ReportKindSpam}).Error; err != nil {
		t.Fatalf("ensure report kind: %v", err)
	}
	now := time.Now().UTC()
	reportedPost := post.Post{ID: uuid.New(), PublicID: basePublicID + 3, AuthorID: author.ID, Domain: models.CoolVibes, PostKind: post.PostKindPost, Published: true, PublishedAt: &now}
	if err := db.Omit(clause.Associations).Create(&reportedPost).Error; err != nil {
		t.Fatalf("create moderation race post: %v", err)
	}
	report := models.Report{ID: uuid.New(), ContentableID: reportedPost.ID, ContentableType: models.EngagementContentableTypePost, ReporterID: reporter.ID, ReportKindKey: models.ReportKindSpam, Status: models.ReportStatusPending}
	if err := db.Omit(clause.Associations).Create(&report).Error; err != nil {
		t.Fatalf("create moderation race report: %v", err)
	}
	t.Cleanup(func() {
		db.Unscoped().Where("id = ?", report.ID).Delete(&models.Report{})
		db.Unscoped().Where("id = ?", reportedPost.ID).Delete(&post.Post{})
		db.Unscoped().Where("id IN ?", []uuid.UUID{author.ID, reporter.ID, moderator.ID}).Delete(&models.User{})
	})

	repo := NewModerationRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := make(chan struct{})
	errs := make(chan error, 2)
	var writers sync.WaitGroup
	writers.Add(2)
	go func() {
		defer writers.Done()
		<-start
		hide := false
		_, err := repo.ResolveReport(ctx, ports.ModerationResolveInput{ReportID: report.ID, Status: domainmoderation.StatusActioned, ReviewedByID: moderator.ID, Resolution: "resolved and hidden", PublishPost: &hide})
		errs <- err
	}()
	go func() {
		defer writers.Done()
		<-start
		_, err := repo.SetPostPublished(ctx, reportedPost.PublicID, false, moderator.ID, "directly hidden")
		errs <- err
	}()
	close(start)
	writers.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent moderation write error = %v", err)
		}
	}

	var persistedReport models.Report
	if err := db.First(&persistedReport, "id = ?", report.ID).Error; err != nil {
		t.Fatalf("reload moderation race report: %v", err)
	}
	var persistedPost post.Post
	if err := db.First(&persistedPost, "id = ?", reportedPost.ID).Error; err != nil {
		t.Fatalf("reload moderation race post: %v", err)
	}
	if persistedReport.Status != models.ReportStatusActioned || persistedPost.Published {
		t.Fatalf("concurrent moderation result report=%s published=%v", persistedReport.Status, persistedPost.Published)
	}
}

func TestConcurrentCreateReportAndHideCannotLeavePendingReportIntegration(t *testing.T) {
	db := locationRaceIntegrationDB(t)
	for _, table := range []any{&models.Report{}, &models.ReportKind{}, &post.Post{}, &models.User{}} {
		if !db.Migrator().HasTable(table) {
			t.Skip("moderation schema is not migrated in TEST_DATABASE_URL")
		}
	}

	basePublicID := time.Now().UTC().UnixNano()
	author := models.User{ID: uuid.New(), PublicID: basePublicID, Domain: models.CoolVibes, UserName: "report-hide-author-" + uuid.NewString(), DisplayName: "Author", UserRole: constants.UserRoleUser}
	reporter := models.User{ID: uuid.New(), PublicID: basePublicID + 1, Domain: models.CoolVibes, UserName: "report-hide-reporter-" + uuid.NewString(), DisplayName: "Reporter", UserRole: constants.UserRoleUser}
	moderator := models.User{ID: uuid.New(), PublicID: basePublicID + 2, Domain: models.CoolVibes, UserName: "report-hide-reviewer-" + uuid.NewString(), DisplayName: "Reviewer", UserRole: constants.UserRoleModerator}
	users := []models.User{author, reporter, moderator}
	if err := db.Omit(clause.Associations).Create(&users).Error; err != nil {
		t.Fatalf("create report/hide users: %v", err)
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.ReportKind{Key: models.ReportKindSpam}).Error; err != nil {
		t.Fatalf("ensure report kind: %v", err)
	}
	now := time.Now().UTC()
	reportedPost := post.Post{ID: uuid.New(), PublicID: basePublicID + 3, AuthorID: author.ID, Domain: models.CoolVibes, PostKind: post.PostKindPost, Published: true, PublishedAt: &now}
	if err := db.Omit(clause.Associations).Create(&reportedPost).Error; err != nil {
		t.Fatalf("create report/hide post: %v", err)
	}
	t.Cleanup(func() {
		db.Unscoped().Where("contentable_id = ?", reportedPost.ID).Delete(&models.Report{})
		db.Unscoped().Where("id = ?", reportedPost.ID).Delete(&post.Post{})
		db.Unscoped().Where("id IN ?", []uuid.UUID{author.ID, reporter.ID, moderator.ID}).Delete(&models.User{})
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := make(chan struct{})
	reportErr := make(chan error, 1)
	hideErr := make(chan error, 1)
	go func() {
		<-start
		reportErr <- createReport(ctx, db, reportedPost.ID, models.EngagementContentableTypePost, reporter.ID, models.ReportKindSpam, "concurrent report")
	}()
	go func() {
		<-start
		_, err := NewModerationRepository(db).SetPostPublished(ctx, reportedPost.PublicID, false, moderator.ID, "concurrent hide")
		hideErr <- err
	}()
	close(start)

	if err := <-hideErr; err != nil {
		t.Fatalf("hide post error: %v", err)
	}
	if err := <-reportErr; err != nil && !errors.Is(err, ports.ErrReportTargetNotFound) {
		t.Fatalf("create report error: %v", err)
	}

	var pending int64
	if err := db.Model(&models.Report{}).
		Where("contentable_id = ? AND contentable_type = ? AND status = ?", reportedPost.ID, models.EngagementContentableTypePost, models.ReportStatusPending).
		Count(&pending).Error; err != nil {
		t.Fatalf("count pending reports: %v", err)
	}
	if pending != 0 {
		t.Fatalf("hidden post has %d pending reports", pending)
	}
}

func assertJSONEquivalent(t *testing.T, expected, actual any) {
	t.Helper()
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("marshal expected JSON: %v", err)
	}
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("marshal actual JSON: %v", err)
	}

	var expectedValue any
	if err := json.Unmarshal(expectedJSON, &expectedValue); err != nil {
		t.Fatalf("decode expected JSON: %v", err)
	}
	var actualValue any
	if err := json.Unmarshal(actualJSON, &actualValue); err != nil {
		t.Fatalf("decode actual JSON: %v", err)
	}
	if !reflect.DeepEqual(expectedValue, actualValue) {
		t.Fatalf("JSON contract changed\nexpected: %s\nactual:   %s", expectedJSON, actualJSON)
	}
}
