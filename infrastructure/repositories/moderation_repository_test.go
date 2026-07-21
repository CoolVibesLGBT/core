package repositories

import (
	"context"
	"core/application/ports"
	"core/models"
	"strings"
	"testing"
)

func TestModerationQueueQuerySupportsMixedAndUserTargets(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := NewModerationRepository(db)

	var reports []models.Report
	mixed := repo.reportQueueQuery(context.Background(), ports.ModerationReportFilter{Status: models.ReportStatusPending}).Find(&reports)
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
		ContentableType: models.EngagementContentableTypeUser,
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
