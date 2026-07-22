package repositories

import (
	"context"
	"core/constants"
	domainuser "core/domain/user"
	"core/models"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestApplyReciprocalUserInteractionRejectsSelfBeforeDatabaseAccess(t *testing.T) {
	intent, err := domainuser.NewSetInteractionState(domainuser.InteractionFollow, true)
	if err != nil {
		t.Fatalf("NewSetInteractionState() error = %v", err)
	}
	userID := uuid.New()
	repo := NewEngagementRepository(nil)

	_, err = repo.ApplyReciprocalUserInteraction(context.Background(), userID, userID, intent)
	if !errors.Is(err, domainuser.ErrSelfInteraction) {
		t.Fatalf("ApplyReciprocalUserInteraction(self) error = %v; want ErrSelfInteraction", err)
	}
}

func TestReciprocalInteractionDedupeKeysAreDirectional(t *testing.T) {
	actorID := uuid.New()
	targetID := uuid.New()
	given := reciprocalInteractionDedupeKey(domainuser.InteractionFollow, "given", models.EngagementKindFollowing, actorID, targetID)
	received := reciprocalInteractionDedupeKey(domainuser.InteractionFollow, "received", models.EngagementKindFollower, actorID, targetID)
	if given == received {
		t.Fatalf("given and received dedupe keys must differ: %q", given)
	}
	if given != reciprocalInteractionDedupeKey(domainuser.InteractionFollow, "given", models.EngagementKindFollowing, actorID, targetID) {
		t.Fatal("reciprocal interaction dedupe key is not deterministic")
	}
	like := reciprocalInteractionDedupeKey(domainuser.InteractionLike, "given", models.EngagementKindLikeGiven, actorID, targetID)
	dislike := reciprocalInteractionDedupeKey(domainuser.InteractionLike, "given", models.EngagementKindDislikeGiven, actorID, targetID)
	if like == dislike {
		t.Fatalf("like and dislike dedupe keys must differ: %q", like)
	}
}

func TestApplyReciprocalUserInteractionKeepsLikeAndDislikeKeysDistinctIntegration(t *testing.T) {
	db := engagementViewIntegrationDB(t)
	if err := db.AutoMigrate(&models.Engagement{}, &models.EngagementDetail{}); err != nil {
		t.Fatalf("migrate reciprocal engagement schema: %v", err)
	}

	actor, target := createReciprocalTestUsers(t, db)
	repo := NewEngagementRepository(db)
	for _, positive := range []bool{true, false} {
		intent, err := domainuser.NewToggleReactionState(positive)
		if err != nil {
			t.Fatalf("NewToggleReactionState(%v) error = %v", positive, err)
		}
		transition, err := repo.ApplyReciprocalUserInteraction(context.Background(), actor.ID, target.ID, intent)
		if err != nil || !transition.Enabled || !transition.Changed {
			t.Fatalf("apply reaction %v = %+v, %v; want enabled and changed", positive, transition, err)
		}
	}

	assertReciprocalDetailCount(t, db, actor.ID, target.ID, models.EngagementKindLikeGiven, 1)
	assertReciprocalDetailCount(t, db, target.ID, actor.ID, models.EngagementKindLikeReceived, 1)
	assertReciprocalDetailCount(t, db, actor.ID, target.ID, models.EngagementKindDislikeGiven, 1)
	assertReciprocalDetailCount(t, db, target.ID, actor.ID, models.EngagementKindDisLikeReceived, 1)
}

func TestApplyReciprocalUserInteractionIsIdempotentIntegration(t *testing.T) {
	db := engagementViewIntegrationDB(t)
	if err := db.AutoMigrate(&models.Engagement{}, &models.EngagementDetail{}); err != nil {
		t.Fatalf("migrate reciprocal engagement schema: %v", err)
	}

	actor, target := createReciprocalTestUsers(t, db)
	repo := NewEngagementRepository(db)
	enable, err := domainuser.NewSetInteractionState(domainuser.InteractionFollow, true)
	if err != nil {
		t.Fatalf("NewSetInteractionState(enable) error = %v", err)
	}

	first, err := repo.ApplyReciprocalUserInteraction(context.Background(), actor.ID, target.ID, enable)
	if err != nil || !first.Enabled || !first.Changed {
		t.Fatalf("first enable = %+v, %v; want enabled and changed", first, err)
	}
	second, err := repo.ApplyReciprocalUserInteraction(context.Background(), actor.ID, target.ID, enable)
	if err != nil || !second.Enabled || second.Changed {
		t.Fatalf("second enable = %+v, %v; want enabled and unchanged", second, err)
	}

	assertReciprocalDetailCount(t, db, actor.ID, target.ID, models.EngagementKindFollowing, 1)
	assertReciprocalDetailCount(t, db, target.ID, actor.ID, models.EngagementKindFollower, 1)
	assertUserEngagementCount(t, db, actor.ID, "following_count", 1)
	assertUserEngagementCount(t, db, target.ID, "follower_count", 1)

	disable, err := domainuser.NewSetInteractionState(domainuser.InteractionFollow, false)
	if err != nil {
		t.Fatalf("NewSetInteractionState(disable) error = %v", err)
	}
	first, err = repo.ApplyReciprocalUserInteraction(context.Background(), actor.ID, target.ID, disable)
	if err != nil || first.Enabled || !first.Changed {
		t.Fatalf("first disable = %+v, %v; want disabled and changed", first, err)
	}
	second, err = repo.ApplyReciprocalUserInteraction(context.Background(), actor.ID, target.ID, disable)
	if err != nil || second.Enabled || second.Changed {
		t.Fatalf("second disable = %+v, %v; want disabled and unchanged", second, err)
	}

	assertReciprocalDetailCount(t, db, actor.ID, target.ID, models.EngagementKindFollowing, 0)
	assertReciprocalDetailCount(t, db, target.ID, actor.ID, models.EngagementKindFollower, 0)
	assertUserEngagementCount(t, db, actor.ID, "following_count", 0)
	assertUserEngagementCount(t, db, target.ID, "follower_count", 0)
}

func TestApplyReciprocalUserInteractionRollsBackBothSidesIntegration(t *testing.T) {
	db := engagementViewIntegrationDB(t)
	if err := db.AutoMigrate(&models.Engagement{}, &models.EngagementDetail{}); err != nil {
		t.Fatalf("migrate reciprocal engagement schema: %v", err)
	}

	actor, target := createReciprocalTestUsers(t, db)
	targetAggregate := models.Engagement{
		ID:              uuid.New(),
		ContentableID:   target.ID,
		ContentableType: models.EngagementContentableTypeUser,
		Counts:          datatypes.JSON([]byte("{}")),
	}
	if err := db.Create(&targetAggregate).Error; err != nil {
		t.Fatalf("create target aggregate: %v", err)
	}

	conflictingKey := reciprocalInteractionDedupeKey(domainuser.InteractionBlock, "received", models.EngagementKindBlockedBy, actor.ID, target.ID)
	rogue := models.EngagementDetail{
		ID:           uuid.New(),
		EngagementID: targetAggregate.ID,
		DedupeKey:    &conflictingKey,
		EngagerID:    target.ID,
		EngageeID:    actor.ID,
		Kind:         models.EngagementKindLikeReceived,
	}
	if err := db.Create(&rogue).Error; err != nil {
		t.Fatalf("create conflicting detail: %v", err)
	}

	intent, err := domainuser.NewSetInteractionState(domainuser.InteractionBlock, true)
	if err != nil {
		t.Fatalf("NewSetInteractionState() error = %v", err)
	}
	_, err = NewEngagementRepository(db).ApplyReciprocalUserInteraction(context.Background(), actor.ID, target.ID, intent)
	if err == nil {
		t.Fatal("expected reciprocal detail conflict")
	}

	assertReciprocalDetailCount(t, db, actor.ID, target.ID, models.EngagementKindBlocking, 0)
	assertReciprocalDetailCount(t, db, target.ID, actor.ID, models.EngagementKindBlockedBy, 0)
	var actorAggregates int64
	if err := db.Model(&models.Engagement{}).
		Where("contentable_id = ? AND contentable_type = ?", actor.ID, models.EngagementContentableTypeUser).
		Count(&actorAggregates).Error; err != nil {
		t.Fatalf("count actor aggregates: %v", err)
	}
	if actorAggregates != 0 {
		t.Fatalf("failed reciprocal transaction persisted %d actor aggregates; want 0", actorAggregates)
	}
}

func createReciprocalTestUsers(t *testing.T, db *gorm.DB) (models.User, models.User) {
	t.Helper()
	basePublicID := time.Now().UTC().UnixNano()
	actor := models.User{
		ID: uuid.New(), PublicID: basePublicID, Domain: models.CoolVibes,
		UserName: "reciprocal-actor-" + uuid.NewString(), DisplayName: "Actor", UserRole: constants.UserRoleUser,
	}
	target := models.User{
		ID: uuid.New(), PublicID: basePublicID + 1, Domain: models.CoolVibes,
		UserName: "reciprocal-target-" + uuid.NewString(), DisplayName: "Target", UserRole: constants.UserRoleUser,
	}
	if err := db.Omit(clause.Associations).Create(&[]models.User{actor, target}).Error; err != nil {
		t.Fatalf("create reciprocal test users: %v", err)
	}
	return actor, target
}

func assertReciprocalDetailCount(t *testing.T, db *gorm.DB, engagerID, engageeID uuid.UUID, kind models.EngagementKind, expected int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.EngagementDetail{}).
		Where("engager_id = ? AND engagee_id = ? AND kind = ?", engagerID, engageeID, kind).
		Count(&count).Error; err != nil {
		t.Fatalf("count reciprocal details: %v", err)
	}
	if count != expected {
		t.Fatalf("%s detail count = %d; want %d", kind, count, expected)
	}
}

func assertUserEngagementCount(t *testing.T, db *gorm.DB, userID uuid.UUID, key string, expected float64) {
	t.Helper()
	var aggregate models.Engagement
	if err := db.Where("contentable_id = ? AND contentable_type = ?", userID, models.EngagementContentableTypeUser).
		First(&aggregate).Error; err != nil {
		t.Fatalf("load user engagement aggregate: %v", err)
	}
	counts := make(map[string]interface{})
	if err := json.Unmarshal(aggregate.Counts, &counts); err != nil {
		t.Fatalf("decode engagement counts: %v", err)
	}
	if got := counts[key]; got != expected {
		t.Fatalf("%s = %#v; want %v", key, got, expected)
	}
}
