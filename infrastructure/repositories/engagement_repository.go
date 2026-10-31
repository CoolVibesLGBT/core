package repositories

import (
	"context"
	"core/application/types"
	"core/constants"
	domainuser "core/domain/user"
	domainwallet "core/domain/wallet"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"core/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EngagementRepository struct {
	db *gorm.DB
}

// GORM resource row locks coordinate aggregate creation across application
// instances. The process lock keeps tests and non-PostgreSQL adapters safe.
var fallbackEngagementViewLock sync.Mutex

func NewEngagementRepository(
	db *gorm.DB,
) *EngagementRepository {
	return &EngagementRepository{
		db: db,
	}
}

func (r *EngagementRepository) SocialNotificationsTransactional() bool { return true }

func (r *EngagementRepository) DB() *gorm.DB {
	return r.db
}

func insertViewDetailOnce(tx *gorm.DB, detail *models.EngagementDetail) *gorm.DB {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "dedupe_key"}},
		DoNothing: true,
	}).Create(detail)
}

func lockViewAggregate(tx *gorm.DB, lockKey string) *gorm.DB {
	return lockTransactionResource(tx, lockKey)
}

func engagementAggregateLockKey(contentableType models.EngagementContentableType, contentableID uuid.UUID) string {
	return fmt.Sprintf("engagement:%s:%s", contentableType, contentableID)
}

func incrementViewAggregate(tx *gorm.DB, engagementID uuid.UUID, countKey string) *gorm.DB {
	countsExpression := gorm.Expr(
		"jsonb_set(COALESCE(counts, '{}'::jsonb), ARRAY[?]::text[], to_jsonb(COALESCE((counts ->> ?)::bigint, 0) + 1), true)",
		countKey,
		countKey,
	)
	return tx.Model(&models.Engagement{}).
		Where("id = ?", engagementID).
		Updates(map[string]interface{}{
			"counts":     countsExpression,
			"updated_at": time.Now().UTC(),
		})
}

func lockUserEngagementAggregates(tx *gorm.DB, actorID, targetID uuid.UUID) error {
	ids := []uuid.UUID{actorID, targetID}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	for _, id := range ids {
		if err := lockViewAggregate(tx, engagementAggregateLockKey(models.EngagementContentableTypeUser, id)).Error; err != nil {
			return err
		}
	}
	return nil
}

func loadOrCreateEngagementAggregate(
	tx *gorm.DB,
	contentableID uuid.UUID,
	contentableType models.EngagementContentableType,
) (*models.Engagement, error) {
	var engagement models.Engagement
	result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("contentable_id = ? AND contentable_type = ?", contentableID, contentableType).
		Limit(1).
		Find(&engagement)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 1 {
		return &engagement, nil
	}

	now := time.Now().UTC()
	engagement = models.Engagement{
		ID:              uuid.New(),
		ContentableID:   contentableID,
		ContentableType: contentableType,
		Counts:          datatypes.JSON([]byte("{}")),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := tx.Create(&engagement).Error; err != nil {
		return nil, err
	}
	return &engagement, nil
}

func loadUserEngagementAggregate(tx *gorm.DB, userID uuid.UUID) (*models.Engagement, error) {
	var engagement models.Engagement
	result := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("contentable_id = ? AND contentable_type = ?", userID, models.EngagementContentableTypeUser).
		Limit(1).
		Find(&engagement)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &engagement, nil
}

func ensureUserEngagementAggregate(tx *gorm.DB, aggregate *models.Engagement, userID uuid.UUID) (*models.Engagement, error) {
	if aggregate != nil {
		return aggregate, nil
	}

	aggregate = &models.Engagement{
		ID:              uuid.New(),
		ContentableID:   userID,
		ContentableType: models.EngagementContentableTypeUser,
		Counts:          datatypes.JSON([]byte("{}")),
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	if err := tx.Create(aggregate).Error; err != nil {
		return nil, err
	}
	return aggregate, nil
}

func reciprocalInteractionDetailExists(
	tx *gorm.DB,
	aggregate *models.Engagement,
	engagerID uuid.UUID,
	engageeID uuid.UUID,
	kind models.EngagementKind,
) (bool, error) {
	if aggregate == nil {
		return false, nil
	}

	var count int64
	err := tx.Model(&models.EngagementDetail{}).
		Where("engagement_id = ? AND engager_id = ? AND engagee_id = ? AND kind = ?", aggregate.ID, engagerID, engageeID, kind).
		Count(&count).Error
	return count > 0, err
}

func reciprocalInteractionDedupeKey(interaction domainuser.InteractionKind, direction string, kind models.EngagementKind, actorID, targetID uuid.UUID) string {
	return fmt.Sprintf("user-interaction:%s:%s:%s:%s:%s", interaction, direction, kind, actorID, targetID)
}

func syncEngagementKindCount(tx *gorm.DB, engagementID uuid.UUID, kind models.EngagementKind) error {
	keys, ok := models.EngagementCountKeys[kind]
	if !ok || keys.CountKey == "" {
		return constants.ErrEngagementCounterMissing.Format(kind)
	}

	var count int64
	if err := tx.Model(&models.EngagementDetail{}).
		Where("engagement_id = ? AND kind = ?", engagementID, kind).
		Count(&count).Error; err != nil {
		return err
	}

	if tx.Name() == "postgres" {
		countsExpression := gorm.Expr(
			"jsonb_set(COALESCE(counts, '{}'::jsonb), ARRAY[?]::text[], to_jsonb(CAST(? AS bigint)), true)",
			keys.CountKey,
			count,
		)
		result := tx.Model(&models.Engagement{}).
			Where("id = ?", engagementID).
			Updates(map[string]interface{}{
				"counts":     countsExpression,
				"updated_at": time.Now().UTC(),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return constants.ErrEngagementAggregateNotUpdated
		}
		return nil
	}

	var aggregate models.Engagement
	if err := tx.Select("id", "counts").First(&aggregate, "id = ?", engagementID).Error; err != nil {
		return err
	}
	counts := make(map[string]interface{})
	if len(aggregate.Counts) > 0 && string(aggregate.Counts) != "null" {
		if err := json.Unmarshal(aggregate.Counts, &counts); err != nil {
			return err
		}
	}
	counts[keys.CountKey] = count
	payload, err := json.Marshal(counts)
	if err != nil {
		return err
	}
	return tx.Model(&models.Engagement{}).
		Where("id = ?", engagementID).
		Updates(map[string]interface{}{"counts": datatypes.JSON(payload), "updated_at": time.Now().UTC()}).Error
}

func setReciprocalInteractionDetail(
	tx *gorm.DB,
	aggregate *models.Engagement,
	engagerID uuid.UUID,
	engageeID uuid.UUID,
	kind models.EngagementKind,
	dedupeKey string,
	enabled bool,
) error {
	if aggregate == nil {
		if enabled {
			return constants.ErrEngagementAggregateRequired
		}
		return nil
	}

	var existing []models.EngagementDetail
	if err := tx.
		Where("engagement_id = ? AND engager_id = ? AND engagee_id = ? AND kind = ?", aggregate.ID, engagerID, engageeID, kind).
		Order("created_at ASC, id ASC").
		Find(&existing).Error; err != nil {
		return err
	}

	mutated := false
	if enabled {
		if len(existing) == 0 {
			detail := models.EngagementDetail{
				ID:           uuid.New(),
				EngagementID: aggregate.ID,
				DedupeKey:    &dedupeKey,
				EngagerID:    engagerID,
				EngageeID:    engageeID,
				Kind:         kind,
				CreatedAt:    time.Now().UTC(),
				UpdatedAt:    time.Now().UTC(),
			}
			insert := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "dedupe_key"}},
				DoNothing: true,
			}).Create(&detail)
			if insert.Error != nil {
				return insert.Error
			}
			if insert.RowsAffected != 1 {
				return constants.ErrReciprocalInteractionConflict
			}
			mutated = true
		} else if len(existing) > 1 {
			duplicateIDs := make([]uuid.UUID, 0, len(existing)-1)
			for _, detail := range existing[1:] {
				duplicateIDs = append(duplicateIDs, detail.ID)
			}
			if err := tx.Delete(&models.EngagementDetail{}, "id IN ?", duplicateIDs).Error; err != nil {
				return err
			}
			mutated = true
		}
	} else if len(existing) > 0 {
		ids := make([]uuid.UUID, 0, len(existing))
		for _, detail := range existing {
			ids = append(ids, detail.ID)
		}
		if err := tx.Delete(&models.EngagementDetail{}, "id IN ?", ids).Error; err != nil {
			return err
		}
		mutated = true
	}

	if !mutated {
		return nil
	}
	return syncEngagementKindCount(tx, aggregate.ID, kind)
}

func ensureSocialInteractionAllowed(tx *gorm.DB, actorID, targetID uuid.UUID, kind domainuser.InteractionKind) error {
	var users []matchUserIdentity
	if err := tx.Table("users").Select("id", "domain", "privacy_level").
		Where("id IN ? AND deleted_at IS NULL AND is_bot = FALSE", []uuid.UUID{actorID, targetID}).
		Where("user_role NOT IN ?", []constants.UserRole{constants.UserRoleBanned, constants.UserRoleDeleted, constants.UserRolePending}).
		Find(&users).Error; err != nil {
		return err
	}
	if len(users) != 2 || users[0].Domain != users[1].Domain {
		return domainuser.ErrInteractionUnavailable
	}
	for _, user := range users {
		if user.ID == targetID && kind == domainuser.InteractionLike && user.PrivacyLevel != constants.PrivacyPublic {
			return domainuser.ErrInteractionUnavailable
		}
	}
	for _, pair := range [][2]uuid.UUID{{actorID, targetID}, {targetID, actorID}} {
		blocked, err := userMatchDetailExists(tx, pair[0], pair[1], models.EngagementKindBlocking)
		if err != nil {
			return err
		}
		if blocked {
			return domainuser.ErrInteractionBlocked
		}
	}
	return nil
}

var blockedRelationshipKinds = []models.EngagementKind{
	models.EngagementKindFollowing, models.EngagementKindFollower,
	models.EngagementKindSubscribing, models.EngagementKindSubscribedBy,
	models.EngagementKindLikeGiven, models.EngagementKindLikeReceived,
	models.EngagementKindDislikeGiven, models.EngagementKindDisLikeReceived,
	models.EngagementKindMatched,
}

func clearBlockedUserRelationships(tx *gorm.DB, actor, target *models.Engagement) error {
	first, second := friendshipPair(actor.ContentableID, target.ContentableID)
	friendship, err := findFriendshipDetail(tx, first, second, false)
	if err != nil {
		return err
	}
	if friendship != nil {
		details, err := decodeFriendshipDetails(friendship.Details)
		if err != nil {
			return err
		}
		if details.Status == domainuser.FriendshipPending || details.Status == domainuser.FriendshipAccepted {
			details.Status = domainuser.FriendshipRemoved
			payload, err := json.Marshal(details)
			if err != nil {
				return err
			}
			if err := tx.Model(&models.EngagementDetail{}).Where("id = ?", friendship.ID).
				Updates(map[string]any{"details": datatypes.JSON(payload), "updated_at": time.Now().UTC()}).Error; err != nil {
				return err
			}
		}
		if err := resolveFriendshipNotification(tx, details.PublicID, details.Status); err != nil {
			return err
		}
	}
	if err := tx.Where("engagement_id IN ?", []uuid.UUID{actor.ID, target.ID}).
		Where("(engager_id = ? AND engagee_id = ?) OR (engager_id = ? AND engagee_id = ?)",
			actor.ContentableID, target.ContentableID, target.ContentableID, actor.ContentableID).
		Where("kind IN ?", blockedRelationshipKinds).
		Delete(&models.EngagementDetail{}).Error; err != nil {
		return err
	}
	kinds := append(append([]models.EngagementKind{}, blockedRelationshipKinds...), models.EngagementKindBlocking, models.EngagementKindBlockedBy)
	if err := syncUserAggregateKinds(tx, actor, kinds); err != nil {
		return err
	}
	return syncUserAggregateKinds(tx, target, kinds)
}

// ApplyReciprocalUserInteraction atomically writes the actor-facing and the
// reciprocal target-facing relationship rows. Explicit state intents are
// idempotent; toggle intents resolve their next state while both user
// engagement aggregates are locked in the same transaction.
func (r *EngagementRepository) ApplyReciprocalUserInteraction(
	ctx context.Context,
	actorID uuid.UUID,
	targetID uuid.UUID,
	intent domainuser.InteractionStateIntent,
) (domainuser.InteractionStateTransition, error) {
	if actorID == uuid.Nil || targetID == uuid.Nil {
		return domainuser.InteractionStateTransition{}, constants.ErrInteractionUserIdentifiersRequired
	}
	if actorID == targetID {
		return domainuser.InteractionStateTransition{}, fmt.Errorf("%w: %s", domainuser.ErrSelfInteraction, intent.Interaction())
	}

	pair, err := intent.EngagementPair()
	if err != nil {
		return domainuser.InteractionStateTransition{}, err
	}
	givenKind := models.EngagementKind(pair.Given)
	receivedKind := models.EngagementKind(pair.Received)
	for _, kind := range []models.EngagementKind{givenKind, receivedKind} {
		if _, ok := models.EngagementCountKeys[kind]; !ok {
			return domainuser.InteractionStateTransition{}, constants.ErrEngagementCounterMissing.Format(kind)
		}
	}

	isPostgres := r.db.Name() == "postgres"
	if !isPostgres {
		fallbackEngagementViewLock.Lock()
		defer fallbackEngagementViewLock.Unlock()
	}

	var transition domainuser.InteractionStateTransition
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if intent.Interaction() == domainuser.InteractionBlock {
			if err := lockPrivatePhotoAccessPair(tx, actorID, targetID); err != nil {
				return err
			}
		}
		if isPostgres {
			if err := lockUserEngagementAggregates(tx, actorID, targetID); err != nil {
				return err
			}
		}

		actorAggregate, err := loadUserEngagementAggregate(tx, actorID)
		if err != nil {
			return err
		}
		targetAggregate, err := loadUserEngagementAggregate(tx, targetID)
		if err != nil {
			return err
		}

		current, err := reciprocalInteractionDetailExists(tx, actorAggregate, actorID, targetID, givenKind)
		if err != nil {
			return err
		}
		transition, err = intent.Transition(current)
		if err != nil {
			return err
		}
		if transition.Enabled && intent.Interaction() != domainuser.InteractionBlock {
			if err := ensureSocialInteractionAllowed(tx, actorID, targetID, intent.Interaction()); err != nil {
				return err
			}
		}

		if transition.Enabled || intent.Interaction() == domainuser.InteractionLike {
			actorAggregate, err = ensureUserEngagementAggregate(tx, actorAggregate, actorID)
			if err != nil {
				return err
			}
			targetAggregate, err = ensureUserEngagementAggregate(tx, targetAggregate, targetID)
			if err != nil {
				return err
			}
		}
		if intent.Interaction() == domainuser.InteractionLike {
			if err := setUserReaction(tx, actorAggregate, targetAggregate, pair, transition.Enabled); err != nil {
				return err
			}
			transition.Matched, transition.NewMatch, err = syncMutualUserMatch(tx, actorAggregate, targetAggregate)
			if err != nil {
				return err
			}
			if err := syncMatchingAggregateCounts(tx, actorAggregate); err != nil {
				return err
			}
			if err := syncMatchingAggregateCounts(tx, targetAggregate); err != nil {
				return err
			}
			if transition.NewMatch {
				return enqueueMatchNotifications(tx, actorID, targetID)
			}
			return nil
		}

		if err := setReciprocalInteractionDetail(
			tx,
			actorAggregate,
			actorID,
			targetID,
			givenKind,
			reciprocalInteractionDedupeKey(intent.Interaction(), "given", givenKind, actorID, targetID),
			transition.Enabled,
		); err != nil {
			return err
		}
		if err := setReciprocalInteractionDetail(
			tx,
			targetAggregate,
			targetID,
			actorID,
			receivedKind,
			reciprocalInteractionDedupeKey(intent.Interaction(), "received", receivedKind, actorID, targetID),
			transition.Enabled,
		); err != nil {
			return err
		}
		if intent.Interaction() == domainuser.InteractionBlock && transition.Enabled {
			if err := clearBlockedUserRelationships(tx, actorAggregate, targetAggregate); err != nil {
				return err
			}
			// Commit the block and grant revocations together. A failed cleanup
			// must never allow old approvals to reappear after a later unblock.
			return (&PrivatePhotoRepository{db: tx}).RevokePrivatePhotoAccessBetween(ctx, actorID, targetID, time.Now().UTC())
		}
		if intent.Interaction() == domainuser.InteractionFollow && transition.Changed {
			return enqueueFollowNotifications(tx, actorID, targetID, transition.Enabled)
		}
		return nil
	})
	if err != nil {
		return domainuser.InteractionStateTransition{}, err
	}
	return transition, nil
}

// RecordViewOnce records an authenticated user's view of a single piece of
// content at most once. The nullable unique dedupe key leaves existing,
// repeatable engagement kinds untouched while making view insertion safe when
// multiple requests race. The aggregate is incremented only after this request
// wins that insert race.
func (r *EngagementRepository) RecordViewOnce(
	ctx context.Context,
	engagerID uuid.UUID,
	engageeID uuid.UUID,
	kind models.EngagementKind,
	contentableID uuid.UUID,
	contentableType models.EngagementContentableType,
) (bool, error) {
	if engagerID == uuid.Nil || engageeID == uuid.Nil || contentableID == uuid.Nil {
		return false, constants.ErrViewIdentifiersRequired
	}
	if engagerID == engageeID {
		return false, nil
	}
	if r == nil || r.db == nil {
		return false, constants.ErrEngagementRepositoryNotConfigured
	}
	if kind != models.EngagementKindView && kind != models.EngagementKindViewGiven && kind != models.EngagementKindViewReceived {
		return false, constants.ErrUnsupportedViewEngagementKind.Format(kind)
	}

	if contentableType == models.EngagementContentableTypeUser {
		return r.recordProfileViewOnce(ctx, engagerID, engageeID)
	}
	countKeys, ok := models.EngagementCountKeys[kind]
	if !ok || countKeys.CountKey == "" {
		return false, constants.ErrViewEngagementCounterMissing.Format(kind)
	}

	dedupeKey := fmt.Sprintf("view:%s:%s:%s", contentableType, contentableID, engagerID)
	aggregateLockKey := engagementAggregateLockKey(contentableType, contentableID)
	isPostgres := r.db.Name() == "postgres"
	if !isPostgres {
		fallbackEngagementViewLock.Lock()
		defer fallbackEngagementViewLock.Unlock()
	}

	counted := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockUserEngagementAggregates(tx, engagerID, engageeID); err != nil {
			return err
		}
		if err := ensureSocialInteractionAllowed(tx, engagerID, engageeID, domainuser.InteractionFollow); err != nil {
			return err
		}
		if isPostgres {
			if err := lockViewAggregate(tx, aggregateLockKey).Error; err != nil {
				return err
			}
		}

		engagement, err := loadOrCreateEngagementAggregate(tx, contentableID, contentableType)
		if err != nil {
			return err
		}

		detail := models.EngagementDetail{
			ID:           uuid.New(),
			EngagementID: engagement.ID,
			DedupeKey:    &dedupeKey,
			EngagerID:    engagerID,
			EngageeID:    engageeID,
			Kind:         kind,
		}
		insertDetail := insertViewDetailOnce(tx, &detail)
		if insertDetail.Error != nil {
			return insertDetail.Error
		}
		if insertDetail.RowsAffected == 0 {
			return nil
		}

		updateAggregate := incrementViewAggregate(tx, engagement.ID, countKeys.CountKey)
		if updateAggregate.Error != nil {
			return updateAggregate.Error
		}
		if updateAggregate.RowsAffected != 1 {
			return constants.ErrViewAggregateNotUpdated
		}

		counted = true
		return nil
	})

	return counted, err
}

func (r *EngagementRepository) CreateEngagementDetail(ctx context.Context, detail *models.EngagementDetail) error {
	if detail == nil {
		return constants.ErrEngagementDetailRequired
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return createEngagementDetailInTransaction(tx, detail)
	})
}

func createEngagementDetailInTransaction(tx *gorm.DB, detail *models.EngagementDetail) error {
	if tx == nil {
		return constants.ErrTransactionRequired
	}
	if detail == nil {
		return constants.ErrEngagementDetailRequired
	}

	var engagement models.Engagement
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", detail.EngagementID).
		First(&engagement).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return constants.ErrEngagementRecordNotFound.Format(detail.EngagementID.String())
	}
	if err != nil {
		return err
	}

	create := tx
	if detail.DedupeKey != nil {
		create = create.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "dedupe_key"}},
			DoNothing: true,
		})
	}
	result := create.Create(detail)
	if result.Error != nil {
		return result.Error
	}
	if detail.DedupeKey != nil && result.RowsAffected == 0 {
		return errEngagementDetailAlreadyExists
	}

	counts := map[string]interface{}{}
	if len(engagement.Counts) > 0 {
		if err := json.Unmarshal(engagement.Counts, &counts); err != nil {
			return err
		}
	}
	if counts == nil {
		counts = make(map[string]interface{})
	}

	keys, ok := models.EngagementCountKeys[detail.Kind]
	if !ok {
		return constants.ErrUnknownEngagementKind.Format(string(detail.Kind))
	}
	countVal, _ := counts[keys.CountKey].(float64)
	counts[keys.CountKey] = int64(countVal) + 1

	if keys.AmountKey != "" && len(detail.Details) > 0 {
		var detailsMap map[string]interface{}
		if err := json.Unmarshal(detail.Details, &detailsMap); err != nil {
			return err
		}
		amountText, ok := detailsMap["amount"].(string)
		if !ok {
			return constants.ErrEngagementAmountInvalid
		}
		amount, err := decimal.NewFromString(amountText)
		if err != nil {
			return err
		}

		currentAmount := decimal.Zero
		switch value := counts[keys.AmountKey].(type) {
		case float64:
			currentAmount = decimal.NewFromFloat(value)
		case string:
			currentAmount, err = decimal.NewFromString(value)
			if err != nil {
				return err
			}
		}
		counts[keys.AmountKey] = currentAmount.Add(amount).String()
	}

	newCounts, err := json.Marshal(counts)
	if err != nil {
		return err
	}
	return tx.Model(&models.Engagement{}).
		Where("id = ?", engagement.ID).
		Updates(map[string]interface{}{
			"counts":     datatypes.JSON(newCounts),
			"updated_at": time.Now().UTC(),
		}).Error
}

// GetEngagement fetches engagement aggregate record by contentable id/type
func (r *EngagementRepository) GetEngagement(ctx context.Context, contentableID uuid.UUID, contentableType models.EngagementContentableType) (*models.Engagement, error) {
	var engagement models.Engagement
	if err := r.db.WithContext(ctx).Where("contentable_id = ? AND contentable_type = ?", contentableID, contentableType).First(&engagement).Error; err != nil {
		return nil, err
	}
	return &engagement, nil
}

func (r *EngagementRepository) GetEngagementDetails(ctx context.Context, engagementID uuid.UUID, kind *models.EngagementKind) ([]models.EngagementDetail, error) {
	var details []models.EngagementDetail
	// Base filter
	filters := models.EngagementDetail{
		EngagementID: engagementID,
	}
	// If kind provided, apply it
	if kind != nil {
		filters.Kind = *kind
	}
	err := r.db.WithContext(ctx).Where(&filters).Order("created_at DESC").Find(&details).Error
	if err != nil {
		return nil, err
	}

	return details, nil
}

func (r *EngagementRepository) GetEngagementDetailsWithCursor(ctx context.Context, engagementID uuid.UUID, kind *models.EngagementKind, cursor *types.MatchListCursor, limit int) ([]models.EngagementDetail, *types.MatchListCursor, error) {
	filters := models.EngagementDetail{
		EngagementID: engagementID,
	}
	if kind != nil {
		filters.Kind = *kind
	}
	q := r.db.WithContext(ctx).Model(&models.EngagementDetail{}).Where(&filters)
	return engagementDetailsPage(q, cursor, limit)
}

func engagementDetailsPage(query *gorm.DB, cursor *types.MatchListCursor, limit int) ([]models.EngagementDetail, *types.MatchListCursor, error) {
	var details []models.EngagementDetail
	q := query.Select("engagement_details.*").
		Preload("Engager.Avatar.File").
		Preload("Engagee.Avatar.File")
	if cursor != nil {
		if cursor.DetailID == uuid.Nil {
			q = q.Where("engagement_details.created_at < ?", cursor.OccurredAt)
		} else {
			q = q.Where("engagement_details.created_at < ? OR (engagement_details.created_at = ? AND engagement_details.id < ?)", cursor.OccurredAt, cursor.OccurredAt, cursor.DetailID)
		}
	}
	if limit <= 0 {
		limit = constants.DEFAULT_LIMIT
	}
	if limit > constants.MAXIMUM_LIMIT {
		limit = constants.MAXIMUM_LIMIT
	}
	err := q.Order("engagement_details.created_at DESC, engagement_details.id DESC").
		Limit(limit + 1).
		Find(&details).
		Error

	if err != nil {
		return []models.EngagementDetail{}, nil, err
	}
	if len(details) == 0 {
		return []models.EngagementDetail{}, nil, nil
	}
	if len(details) <= limit {
		return details, nil, nil
	}
	details = details[:limit]
	last := details[len(details)-1]
	return details, &types.MatchListCursor{OccurredAt: last.CreatedAt, DetailID: last.ID}, nil
}

// RemoveEngagementDetail deletes an engagement detail and decrements the count/amount in aggregate
func (r *EngagementRepository) RemoveEngagementDetail(ctx context.Context, detailID uuid.UUID) error {
	if r == nil || r.db == nil {
		return constants.ErrEngagementRepositoryNotConfigured
	}
	if detailID == uuid.Nil {
		return constants.ErrEngagementDetailIDRequired
	}

	// Every other aggregate writer takes the canonical owner lock before row
	// locks. Keep the same order here; the process lock is the non-PostgreSQL
	// equivalent and must span the transaction commit.
	if r.db.Name() != "postgres" {
		fallbackEngagementViewLock.Lock()
		defer fallbackEngagementViewLock.Unlock()
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return removeEngagementDetailInTransaction(tx, detailID)
	})
}

type engagementDetailOwner struct {
	EngagementID    uuid.UUID
	ContentableID   uuid.UUID
	ContentableType models.EngagementContentableType
}

// lockEngagementDetailOwnerInTransaction resolves the canonical aggregate
// owner without taking a row lock, then acquires locks in the global order:
// owner resource row -> aggregate row -> detail row. Resolving first is safe:
// detail ownership is immutable, and the final locked read detects a racing
// delete instead of mutating a different aggregate.
func lockEngagementDetailOwnerInTransaction(tx *gorm.DB, detailID uuid.UUID) (*models.Engagement, *models.EngagementDetail, error) {
	if tx == nil {
		return nil, nil, constants.ErrTransactionRequired
	}

	var owner engagementDetailOwner
	err := tx.Model(&models.EngagementDetail{}).
		Select("engagement_details.engagement_id, engagements.contentable_id, engagements.contentable_type").
		Joins("JOIN engagements ON engagements.id = engagement_details.engagement_id").
		Where("engagement_details.id = ?", detailID).
		Take(&owner).Error
	if err != nil {
		return nil, nil, err
	}
	if owner.EngagementID == uuid.Nil || owner.ContentableID == uuid.Nil || owner.ContentableType == "" {
		return nil, nil, constants.ErrEngagementDetailOwnerInvalid
	}

	if tx.Name() == "postgres" {
		if err := lockViewAggregate(tx, engagementAggregateLockKey(owner.ContentableType, owner.ContentableID)).Error; err != nil {
			return nil, nil, err
		}
	}

	var engagement models.Engagement
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&engagement, "id = ?", owner.EngagementID).Error; err != nil {
		return nil, nil, err
	}

	var detail models.EngagementDetail
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&detail, "id = ? AND engagement_id = ?", detailID, engagement.ID).Error; err != nil {
		return nil, nil, err
	}
	return &engagement, &detail, nil
}

func removeEngagementDetailInTransaction(tx *gorm.DB, detailID uuid.UUID) error {
	engagement, detail, err := lockEngagementDetailOwnerInTransaction(tx, detailID)
	if err != nil {
		return err
	}
	return removeLockedEngagementDetailInTransaction(tx, engagement, detail)
}

// removeLockedEngagementDetailInTransaction applies the aggregate mutation
// after the caller has locked the canonical owner, aggregate row, and detail
// row in that order.
func removeLockedEngagementDetailInTransaction(tx *gorm.DB, engagement *models.Engagement, detail *models.EngagementDetail) error {
	if tx == nil {
		return constants.ErrTransactionRequired
	}
	if engagement == nil || engagement.ID == uuid.Nil {
		return constants.ErrEngagementAggregateRequired
	}
	if detail == nil || detail.ID == uuid.Nil || detail.EngagementID != engagement.ID {
		return constants.ErrEngagementDetailAggregateMismatch
	}

	counts := make(map[string]interface{})
	if len(engagement.Counts) > 0 && string(engagement.Counts) != "null" {
		if err := json.Unmarshal(engagement.Counts, &counts); err != nil {
			return err
		}
	}
	keys, ok := models.EngagementCountKeys[detail.Kind]
	if !ok {
		return constants.ErrUnknownEngagementKind.Format(string(detail.Kind))
	}

	countValue := int64(0)
	switch value := counts[keys.CountKey].(type) {
	case float64:
		countValue = int64(value)
	case string:
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return constants.ErrEngagementCountInvalid.Format(value, err)
		}
		countValue = parsed
	}
	if countValue > 0 {
		countValue--
	}
	counts[keys.CountKey] = countValue

	if keys.AmountKey != "" && len(detail.Details) > 0 {
		var detailsMap map[string]interface{}
		if err := json.Unmarshal(detail.Details, &detailsMap); err != nil {
			return err
		}
		amountText, ok := detailsMap["amount"].(string)
		if !ok {
			return constants.ErrEngagementAmountInvalid
		}
		amount, err := decimal.NewFromString(amountText)
		if err != nil {
			return err
		}
		currentAmount := decimal.Zero
		switch value := counts[keys.AmountKey].(type) {
		case float64:
			currentAmount = decimal.NewFromFloat(value)
		case string:
			currentAmount, err = decimal.NewFromString(value)
			if err != nil {
				return err
			}
		}
		currentAmount = currentAmount.Sub(amount)
		if currentAmount.IsNegative() {
			currentAmount = decimal.Zero
		}
		counts[keys.AmountKey] = currentAmount.String()
	}

	newCounts, err := json.Marshal(counts)
	if err != nil {
		return err
	}
	if err := tx.Model(&models.Engagement{}).
		Where("id = ?", engagement.ID).
		Updates(map[string]interface{}{
			"counts":     datatypes.JSON(newCounts),
			"updated_at": time.Now().UTC(),
		}).Error; err != nil {
		return err
	}
	return tx.Delete(&models.EngagementDetail{}, "id = ?", detail.ID).Error
}

func (r *EngagementRepository) HasUserEngaged(ctx context.Context, engagerID uuid.UUID, engageeID uuid.UUID, kind models.EngagementKind) (bool, error) {
	var count int64

	/*
		err := r.db.WithContext(ctx).
			Model(&models.EngagementDetail{}).
			Where("engager_id = ? AND engagee_id = ? AND kind = ?", engagerID, engageeID, kind).
			Count(&count).Error
	*/
	err := r.db.WithContext(ctx).
		Model(&models.EngagementDetail{}).
		Where(&models.EngagementDetail{
			EngagerID: engagerID,
			EngageeID: engageeID,
			Kind:      kind,
		}).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// userID,       // Kimin içeriği (post, video, vs) bu? İçeriğin sahibi (target user)
// engagerID,    // Etkileşimi yapan kullanıcı (engager) //takip eden
// engageeID,	// Etkilesimi alan kullanici ornegin: takip edilen
func (r *EngagementRepository) ToggleEngagement(ctx context.Context, engagerID uuid.UUID, engageeID uuid.UUID, kind models.EngagementKind, contentableID uuid.UUID, contentableType models.EngagementContentableType) (bool, error) {
	if r == nil || r.db == nil {
		return false, constants.ErrEngagementRepositoryNotConfigured
	}
	if engagerID == uuid.Nil || engageeID == uuid.Nil || contentableID == uuid.Nil || contentableType == "" {
		return false, constants.ErrEngagementIdentifiersRequired
	}
	keys, ok := models.EngagementCountKeys[kind]
	if !ok {
		return false, constants.ErrUnknownEngagementKind.Format(string(kind))
	}
	if keys.AmountKey != "" {
		return false, constants.ErrAmountEngagementToggleDenied
	}

	isPostgres := r.db.Name() == "postgres"
	if !isPostgres {
		fallbackEngagementViewLock.Lock()
		defer fallbackEngagementViewLock.Unlock()
	}

	enabled := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if isPostgres {
			if err := lockViewAggregate(tx, engagementAggregateLockKey(contentableType, contentableID)).Error; err != nil {
				return err
			}
		}
		engagement, err := loadOrCreateEngagementAggregate(tx, contentableID, contentableType)
		if err != nil {
			return err
		}

		var existingDetail models.EngagementDetail
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("engagement_id = ? AND engager_id = ? AND engagee_id = ? AND kind = ?", engagement.ID, engagerID, engageeID, kind).
			First(&existingDetail).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			now := time.Now().UTC()
			detail := models.EngagementDetail{
				ID:           uuid.New(),
				EngagementID: engagement.ID,
				EngagerID:    engagerID,
				EngageeID:    engageeID,
				Kind:         kind,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			if err := createEngagementDetailInTransaction(tx, &detail); err != nil {
				return err
			}
			enabled = true
			return nil
		}
		if err != nil {
			return err
		}
		return removeEngagementDetailInTransaction(tx, existingDetail.ID)
	})
	return enabled, err
}

func (r *EngagementRepository) GetEngagements(ctx context.Context, contentableType models.EngagementContentableType, contentableId uuid.UUID, engagementKind models.EngagementKind, cursor *types.MatchListCursor, limit int, viewers ...uuid.UUID) ([]models.EngagementDetail, *types.MatchListCursor, error) {
	engagement, err := r.GetEngagement(ctx, contentableId, contentableType)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return []models.EngagementDetail{}, nil, nil
	}
	if err != nil {

		return []models.EngagementDetail{}, nil, err
	}

	viewerID := contentableId
	if len(viewers) > 0 {
		viewerID = viewers[0]
	}
	query := r.db.WithContext(ctx).Model(&models.EngagementDetail{}).
		Where("engagement_details.engagement_id = ? AND engagement_details.kind = ?", engagement.ID, engagementKind)
	if engagementKind == models.EngagementKindFollowing || engagementKind == models.EngagementKindFollower {
		query = query.Joins("JOIN users ON users.id = engagement_details.engagee_id").
			Where("users.deleted_at IS NULL AND users.is_bot = FALSE").
			Where("users.user_role NOT IN ?", []constants.UserRole{constants.UserRoleBanned, constants.UserRoleDeleted, constants.UserRolePending}).
			Where("users.domain = (?)", userDomainSubquery(r.db, contentableId))
		query = userBlockScope(query, viewerID, "users")
		if viewerID != contentableId {
			query = profileViewerScope(query, viewerID)
		}
	}
	return engagementDetailsPage(query, cursor, limit)
}

func (r *EngagementRepository) AddEngagement(ctx context.Context, engagerID uuid.UUID, engageeID uuid.UUID, kind models.EngagementKind, contentableID uuid.UUID, contentableType models.EngagementContentableType) error {
	if r == nil || r.db == nil {
		return constants.ErrEngagementRepositoryNotConfigured
	}
	if engagerID == uuid.Nil || engageeID == uuid.Nil || contentableID == uuid.Nil || contentableType == "" {
		return constants.ErrEngagementIdentifiersRequired
	}
	keys, ok := models.EngagementCountKeys[kind]
	if !ok {
		return constants.ErrUnknownEngagementKind.Format(string(kind))
	}
	if keys.AmountKey != "" {
		return constants.ErrAmountEngagementRequiresAddTip
	}

	isPostgres := r.db.Name() == "postgres"
	if !isPostgres {
		fallbackEngagementViewLock.Lock()
		defer fallbackEngagementViewLock.Unlock()
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if isPostgres {
			if err := lockViewAggregate(tx, engagementAggregateLockKey(contentableType, contentableID)).Error; err != nil {
				return err
			}
		}
		engagement, err := loadOrCreateEngagementAggregate(tx, contentableID, contentableType)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		detail := models.EngagementDetail{
			ID:           uuid.New(),
			EngagementID: engagement.ID,
			EngagerID:    engagerID,
			EngageeID:    engageeID,
			Kind:         kind,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		return createEngagementDetailInTransaction(tx, &detail)
	})
}

func (r *EngagementRepository) AddTip(ctx context.Context, engagerID uuid.UUID, engageeID uuid.UUID, tipAmount decimal.Decimal, contentableID uuid.UUID, contentableType models.EngagementContentableType, kind models.EngagementKind) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return addTipInTransaction(tx, engagerID, engageeID, tipAmount, contentableID, contentableType, kind)
	})
}

type amountEngagementConfig struct {
	dedupeKey *string
	details   map[string]string
}

type amountEngagementOption func(*amountEngagementConfig)

func withAmountEngagementDedupeKey(key string) amountEngagementOption {
	return func(config *amountEngagementConfig) {
		config.dedupeKey = &key
	}
}

func withAmountEngagementDetail(key, value string) amountEngagementOption {
	return func(config *amountEngagementConfig) {
		config.details[key] = value
	}
}

func addTipInTransaction(tx *gorm.DB, engagerID uuid.UUID, engageeID uuid.UUID, tipAmount decimal.Decimal, contentableID uuid.UUID, contentableType models.EngagementContentableType, kind models.EngagementKind, options ...amountEngagementOption) error {
	if tx == nil {
		return constants.ErrTransactionRequired
	}
	if engagerID == uuid.Nil || engageeID == uuid.Nil || contentableID == uuid.Nil || contentableType == "" {
		return constants.ErrAmountEngagementIdentifiersRequired
	}
	keys, ok := models.EngagementCountKeys[kind]
	if !ok {
		return constants.ErrUnknownEngagementKind.Format(string(kind))
	}
	if keys.AmountKey == "" {
		return constants.ErrEngagementAmountUnsupported.Format(string(kind))
	}
	if err := domainwallet.ValidateMoneyRepresentation(tipAmount); err != nil {
		return err
	}
	if tipAmount.Coefficient().Sign() <= 0 {
		return domainwallet.ErrInvalidAmount
	}

	if tx.Name() == "postgres" {
		lockKey := engagementAggregateLockKey(contentableType, contentableID)
		if err := lockViewAggregate(tx, lockKey).Error; err != nil {
			return err
		}
	} else {
		fallbackEngagementViewLock.Lock()
		defer fallbackEngagementViewLock.Unlock()
	}

	engagement, err := loadOrCreateEngagementAggregate(tx, contentableID, contentableType)
	if err != nil {
		return err
	}

	newDetail, err := newAmountEngagementDetail(engagement.ID, engagerID, engageeID, tipAmount, kind, options...)
	if err != nil {
		return err
	}

	return createEngagementDetailInTransaction(tx, newDetail)
}

func newAmountEngagementDetail(engagementID, engagerID, engageeID uuid.UUID, amount decimal.Decimal, kind models.EngagementKind, options ...amountEngagementOption) (*models.EngagementDetail, error) {
	config := amountEngagementConfig{
		details: map[string]string{"amount": amount.String()},
	}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	detailsJSON, err := json.Marshal(config.details)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	detail := &models.EngagementDetail{
		ID:           uuid.New(),
		EngagementID: engagementID,
		EngagerID:    engagerID,
		EngageeID:    engageeID,
		Kind:         kind,
		DedupeKey:    config.dedupeKey,
		Details:      datatypes.JSON(detailsJSON),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	return detail, nil
}
