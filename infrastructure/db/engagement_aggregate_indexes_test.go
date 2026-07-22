package db

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"core/constants"
	"core/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestEngagementAggregateIndexDefinitionEnforcesOwnerIdentity(t *testing.T) {
	definitions := engagementAggregateIndexDefinitions()
	if len(definitions) != 1 {
		t.Fatalf("engagement aggregate index count = %d, want 1", len(definitions))
	}
	definition := definitions[0]
	if definition.Name != "uidx_engagements_contentable" || definition.Table != "engagements" || !definition.Unique {
		t.Fatalf("unexpected engagement aggregate index: %#v", definition)
	}
	if strings.Join(definition.Columns, ",") != "contentable_type,contentable_id" {
		t.Fatalf("engagement owner index columns = %#v", definition.Columns)
	}
}

func TestMergeEngagementAggregateCountsPreservesTotals(t *testing.T) {
	ownerID := uuid.New()
	aggregates := []models.Engagement{
		{
			ContentableID: ownerID,
			Counts: datatypes.JSON([]byte(`{
				"tip_count": 1,
				"tip_amount": "2.25",
				"custom_projection": {"kept": true}
			}`)),
		},
		{
			ContentableID: ownerID,
			Counts: datatypes.JSON([]byte(`{
				"tip_count": 2,
				"tip_amount": "1.75",
				"view_count": 3,
				"custom_projection": {"kept": false}
			}`)),
		},
	}

	merged, err := mergeEngagementAggregateCounts(aggregates)
	if err != nil {
		t.Fatalf("mergeEngagementAggregateCounts() error = %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(merged, &result); err != nil {
		t.Fatalf("decode merged counts: %v", err)
	}
	if result["tip_count"] != float64(3) || result["tip_amount"] != "4" || result["view_count"] != float64(3) {
		t.Fatalf("merged known counts = %#v", result)
	}
	custom, ok := result["custom_projection"].(map[string]interface{})
	if !ok || custom["kept"] != true {
		t.Fatalf("canonical unknown projection was not preserved: %#v", result["custom_projection"])
	}
}

func TestMergeEngagementAggregateCountsRejectsFractionalCounter(t *testing.T) {
	_, err := mergeEngagementAggregateCounts([]models.Engagement{{
		Counts: datatypes.JSON([]byte(`{"tip_count":"1.5"}`)),
	}})
	if err == nil || !strings.Contains(err.Error(), "non-negative integer") {
		t.Fatalf("fractional count error = %v", err)
	}
}

func TestEngagementJSONDecimalRejectsScientificNotation(t *testing.T) {
	if _, err := engagementJSONDecimal(json.RawMessage(`"1e2147483647"`)); err == nil {
		t.Fatal("engagementJSONDecimal() accepted unsafe scientific notation")
	}
}

func TestMigrateEngagementAggregateUniquenessMergesHistoricalRowsIntegration(t *testing.T) {
	database := migrationIntegrationDB(t).Session(&gorm.Session{SkipDefaultTransaction: true})
	if !database.Migrator().HasTable(&models.Engagement{}) || !database.Migrator().HasTable(&models.EngagementDetail{}) {
		t.Skip("engagement schema is not migrated in TEST_DATABASE_URL")
	}
	if err := database.Exec("DROP INDEX IF EXISTS uidx_engagements_contentable").Error; err != nil {
		t.Fatalf("drop aggregate index fixture: %v", err)
	}
	if err := database.Exec("CREATE INDEX uidx_engagements_contentable ON engagements (contentable_type)").Error; err != nil {
		t.Fatalf("create malformed aggregate index fixture: %v", err)
	}

	basePublicID := time.Now().UTC().UnixNano()
	actor := models.User{
		ID: uuid.New(), PublicID: basePublicID, Domain: models.CoolVibes,
		UserName: "aggregate-migration-actor-" + uuid.NewString(), DisplayName: "Actor", UserRole: constants.UserRoleUser,
	}
	target := models.User{
		ID: uuid.New(), PublicID: basePublicID + 1, Domain: models.CoolVibes,
		UserName: "aggregate-migration-target-" + uuid.NewString(), DisplayName: "Target", UserRole: constants.UserRoleUser,
	}
	if err := database.Omit(clause.Associations).Create(&[]models.User{actor, target}).Error; err != nil {
		t.Fatalf("create migration users: %v", err)
	}

	ownerID := uuid.New()
	first := models.Engagement{
		ID: uuid.New(), ContentableID: ownerID, ContentableType: models.EngagementContentableTypePost,
		Counts:    datatypes.JSON([]byte(`{"tip_count":1,"tip_amount":"2.25"}`)),
		CreatedAt: time.Now().UTC().Add(-time.Minute),
	}
	second := models.Engagement{
		ID: uuid.New(), ContentableID: ownerID, ContentableType: models.EngagementContentableTypePost,
		Counts:    datatypes.JSON([]byte(`{"tip_count":2,"tip_amount":"1.75"}`)),
		CreatedAt: time.Now().UTC(),
	}
	if err := database.Omit(clause.Associations).Create(&[]models.Engagement{first, second}).Error; err != nil {
		t.Fatalf("create duplicate aggregates: %v", err)
	}
	details := []models.EngagementDetail{
		{ID: uuid.New(), EngagementID: first.ID, EngagerID: actor.ID, EngageeID: target.ID, Kind: models.EngagementKindTip},
		{ID: uuid.New(), EngagementID: second.ID, EngagerID: actor.ID, EngageeID: target.ID, Kind: models.EngagementKindTip},
	}
	if err := database.Omit(clause.Associations).Create(&details).Error; err != nil {
		t.Fatalf("create duplicate aggregate details: %v", err)
	}

	if err := MigrateEngagementAggregateUniqueness(database); err != nil {
		t.Fatalf("migrate aggregate uniqueness: %v", err)
	}
	if err := MigrateEngagementAggregateUniqueness(database); err != nil {
		t.Fatalf("idempotent aggregate migration: %v", err)
	}

	var aggregates []models.Engagement
	if err := database.Where("contentable_id = ? AND contentable_type = ?", ownerID, models.EngagementContentableTypePost).Find(&aggregates).Error; err != nil {
		t.Fatalf("load canonical aggregate: %v", err)
	}
	if len(aggregates) != 1 || aggregates[0].ID != first.ID {
		t.Fatalf("canonical aggregates = %#v, want oldest %s", aggregates, first.ID)
	}
	var detailCount int64
	if err := database.Model(&models.EngagementDetail{}).Where("engagement_id = ?", first.ID).Count(&detailCount).Error; err != nil {
		t.Fatalf("count moved details: %v", err)
	}
	if detailCount != 2 {
		t.Fatalf("moved detail count = %d, want 2", detailCount)
	}
	var counts map[string]interface{}
	if err := json.Unmarshal(aggregates[0].Counts, &counts); err != nil {
		t.Fatalf("decode canonical counts: %v", err)
	}
	if counts["tip_count"] != float64(3) || counts["tip_amount"] != "4" {
		t.Fatalf("canonical counts = %#v", counts)
	}
}
