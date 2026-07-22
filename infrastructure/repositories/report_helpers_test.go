package repositories

import (
	"context"
	"core/application/ports"
	"core/constants"
	"core/models"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

func TestCreateReportValidatesKindAndIsSequentiallyIdempotent(t *testing.T) {
	db := eventRSVPIntegrationDB(t)
	if !db.Migrator().HasTable(&models.Report{}) || !db.Migrator().HasTable(&models.ReportKind{}) {
		t.Skip("reporting schema is not migrated in TEST_DATABASE_URL")
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.ReportKind{Key: models.ReportKindSpam}).Error; err != nil {
		t.Fatalf("ensure report kind: %v", err)
	}
	reporter := models.User{
		ID: uuid.New(), PublicID: time.Now().UnixNano(), Domain: models.CoolVibes,
		UserName: "reporter-" + uuid.NewString(), DisplayName: "Reporter", UserRole: constants.UserRoleUser,
	}
	if err := db.Omit(clause.Associations).Create(&reporter).Error; err != nil {
		t.Fatalf("create reporter: %v", err)
	}

	target := models.User{
		ID: uuid.New(), PublicID: time.Now().UnixNano() + 1, Domain: models.CoolVibes,
		UserName: "report-target-" + uuid.NewString(), DisplayName: "Target", UserRole: constants.UserRoleUser,
	}
	if err := db.Omit(clause.Associations).Create(&target).Error; err != nil {
		t.Fatalf("create report target: %v", err)
	}
	targetID := target.ID
	if err := createReport(context.Background(), db, targetID, models.EngagementContentableTypeUser, reporter.ID, " spam ", " details "); err != nil {
		t.Fatalf("createReport() error = %v", err)
	}
	if err := createReport(context.Background(), db, targetID, models.EngagementContentableTypeUser, reporter.ID, models.ReportKindSpam, "updated details"); err != nil {
		t.Fatalf("repeat createReport() error = %v", err)
	}

	var reports []models.Report
	if err := db.Where("reporter_id = ? AND contentable_id = ? AND status = ?", reporter.ID, targetID, models.ReportStatusPending).Find(&reports).Error; err != nil {
		t.Fatalf("load reports: %v", err)
	}
	if len(reports) != 1 || reports[0].Reason != "updated details" || reports[0].ReportKindKey != models.ReportKindSpam {
		t.Fatalf("idempotent reports = %#v", reports)
	}

	err := createReport(context.Background(), db, targetID, models.EngagementContentableTypeUser, reporter.ID, "not_seeded", "")
	if !errors.Is(err, ports.ErrInvalidReportKind) {
		t.Fatalf("invalid kind error = %v", err)
	}

	err = createReport(context.Background(), db, uuid.New(), models.EngagementContentableTypeUser, reporter.ID, models.ReportKindSpam, "")
	if !errors.Is(err, ports.ErrReportTargetNotFound) {
		t.Fatalf("missing target error = %v", err)
	}
}
