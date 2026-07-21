package reportkinds

import (
	report "core/models"
	"errors"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestReportKindSeedsAreCompleteAndUnique(t *testing.T) {
	seeds := reportKindSeeds()
	if len(seeds) != 37 {
		t.Fatalf("report kind count = %d, want 37", len(seeds))
	}
	seen := make(map[string]struct{}, len(seeds))
	for _, seed := range seeds {
		if seed.Key == "" {
			t.Fatal("report kind has an empty key")
		}
		if !report.IsStandardReportKind(seed.Key) {
			t.Fatalf("seeded report kind %q is missing from legacy compatibility", seed.Key)
		}
		if _, exists := seen[seed.Key]; exists {
			t.Fatalf("duplicate report kind %q", seed.Key)
		}
		seen[seed.Key] = struct{}{}
	}
}

func TestSeedReportKindsPropagatesDatabaseErrors(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "host=localhost user=test password=test dbname=test sslmode=disable",
		PreferSimpleProtocol: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open dry-run database: %v", err)
	}
	wantErr := errors.New("forced report-kind write failure")
	if err := db.Callback().Create().Before("gorm:create").Register("test:fail_report_kind_create", func(tx *gorm.DB) {
		_ = tx.AddError(wantErr)
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}

	err = seedReportKinds(db, reportKindSeeds()[:1])
	if !errors.Is(err, wantErr) {
		t.Fatalf("seed error = %v, want %v", err, wantErr)
	}
}
