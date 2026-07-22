package repositories

import (
	"context"
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

// PostgreSQL advisory locks coordinate aggregate creation across application
// instances. The process lock keeps tests and non-PostgreSQL adapters safe.
var fallbackEngagementViewLock sync.Mutex

var errEngagementDetailAlreadyExists = errors.New("engagement detail already exists")

func NewEngagementRepository(
	db *gorm.DB,
) *EngagementRepository {
	return &EngagementRepository{
		db: db,
	}
}

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
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", lockKey)
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
		return fmt.Errorf("missing engagement counter for kind %s", kind)
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
			return errors.New("engagement aggregate was not updated")
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
			return errors.New("engagement aggregate is required")
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
				return errors.New("reciprocal interaction detail conflict")
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
		return domainuser.InteractionStateTransition{}, errors.New("interaction user identifiers are required")
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
			return domainuser.InteractionStateTransition{}, fmt.Errorf("missing engagement counter for kind %s", kind)
		}
	}

	isPostgres := r.db.Name() == "postgres"
	if !isPostgres {
		fallbackEngagementViewLock.Lock()
		defer fallbackEngagementViewLock.Unlock()
	}

	var transition domainuser.InteractionStateTransition
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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

		if transition.Enabled {
			actorAggregate, err = ensureUserEngagementAggregate(tx, actorAggregate, actorID)
			if err != nil {
				return err
			}
			targetAggregate, err = ensureUserEngagementAggregate(tx, targetAggregate, targetID)
			if err != nil {
				return err
			}
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
		return setReciprocalInteractionDetail(
			tx,
			targetAggregate,
			targetID,
			actorID,
			receivedKind,
			reciprocalInteractionDedupeKey(intent.Interaction(), "received", receivedKind, actorID, targetID),
			transition.Enabled,
		)
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
		return false, errors.New("view identifiers are required")
	}
	if engagerID == engageeID {
		return false, nil
	}
	if kind != models.EngagementKindView && kind != models.EngagementKindViewGiven && kind != models.EngagementKindViewReceived {
		return false, fmt.Errorf("unsupported view engagement kind: %s", kind)
	}

	countKeys, ok := models.EngagementCountKeys[kind]
	if !ok || countKeys.CountKey == "" {
		return false, fmt.Errorf("missing counter for view engagement kind: %s", kind)
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
			return errors.New("view aggregate was not updated")
		}

		counted = true
		return nil
	})

	return counted, err
}

func (r *EngagementRepository) CreateEngagementDetail(ctx context.Context, detail *models.EngagementDetail) error {
	if detail == nil {
		return errors.New("detail is nil")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return createEngagementDetailInTransaction(tx, detail)
	})
}

func createEngagementDetailInTransaction(tx *gorm.DB, detail *models.EngagementDetail) error {
	if tx == nil {
		return errors.New("transaction is nil")
	}
	if detail == nil {
		return errors.New("detail is nil")
	}

	var engagement models.Engagement
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", detail.EngagementID).
		First(&engagement).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("engagement record not found for engagement_id: " + detail.EngagementID.String())
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
		return errors.New("unknown engagement kind: " + string(detail.Kind))
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
			return errors.New("engagement amount must be a decimal string")
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

// ListEngagementDetails lists all engagement details for a given engagement ID, optionally filtered by kind
func (r *EngagementRepository) ListEngagementDetailsDeprecated(ctx context.Context, engagementID uuid.UUID, kind *string) ([]models.EngagementDetail, error) {
	var details []models.EngagementDetail
	query := r.db.WithContext(ctx).Where("engagement_id = ?", engagementID)
	if kind != nil {
		query = query.Where("kind = ?", *kind)
	}
	if err := query.Order("created_at desc").Find(&details).Error; err != nil {
		return nil, err
	}
	return details, nil
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

func (r *EngagementRepository) GetEngagementDetailsWithCursor(ctx context.Context, engagementID uuid.UUID, kind *models.EngagementKind, cursor *time.Time, limit int) ([]models.EngagementDetail, *time.Time, error) {
	var details []models.EngagementDetail
	filters := models.EngagementDetail{
		EngagementID: engagementID,
	}
	if kind != nil {
		filters.Kind = *kind
	}
	q := r.db.WithContext(ctx).Model(&models.EngagementDetail{}).
		Preload("Engager.Avatar.File").
		Preload("Engagee.Avatar.File").
		Where(&filters)
	if cursor != nil {
		q = q.Where("created_at < ?", *cursor)
	}
	if limit <= 0 {
		limit = 100
	}
	err := q.Order("created_at DESC").
		Limit(limit).
		Find(&details).
		Error

	if err != nil {
		return []models.EngagementDetail{}, nil, err
	}
	if len(details) == 0 {
		return []models.EngagementDetail{}, nil, nil
	}
	last := details[len(details)-1].CreatedAt
	nextCursor := &last

	return details, nextCursor, nil
}

// RemoveEngagementDetail deletes an engagement detail and decrements the count/amount in aggregate
func (r *EngagementRepository) RemoveEngagementDetail(ctx context.Context, detailID uuid.UUID) error {
	if r == nil || r.db == nil {
		return errors.New("engagement repository is not configured")
	}
	if detailID == uuid.Nil {
		return errors.New("engagement detail identifier is required")
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
// owner advisory lock -> aggregate row -> detail row. Resolving first is safe:
// detail ownership is immutable, and the final locked read detects a racing
// delete instead of mutating a different aggregate.
func lockEngagementDetailOwnerInTransaction(tx *gorm.DB, detailID uuid.UUID) (*models.Engagement, *models.EngagementDetail, error) {
	if tx == nil {
		return nil, nil, errors.New("transaction is nil")
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
		return nil, nil, errors.New("engagement detail owner is invalid")
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
		return errors.New("transaction is nil")
	}
	if engagement == nil || engagement.ID == uuid.Nil {
		return errors.New("engagement aggregate is required")
	}
	if detail == nil || detail.ID == uuid.Nil || detail.EngagementID != engagement.ID {
		return errors.New("locked engagement detail does not belong to aggregate")
	}

	counts := make(map[string]interface{})
	if len(engagement.Counts) > 0 && string(engagement.Counts) != "null" {
		if err := json.Unmarshal(engagement.Counts, &counts); err != nil {
			return err
		}
	}
	keys, ok := models.EngagementCountKeys[detail.Kind]
	if !ok {
		return errors.New("unknown engagement kind: " + string(detail.Kind))
	}

	countValue := int64(0)
	switch value := counts[keys.CountKey].(type) {
	case float64:
		countValue = int64(value)
	case string:
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid engagement count %q: %w", value, err)
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
			return errors.New("engagement amount must be a decimal string")
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
		return false, errors.New("engagement repository is not configured")
	}
	if engagerID == uuid.Nil || engageeID == uuid.Nil || contentableID == uuid.Nil || contentableType == "" {
		return false, errors.New("engagement identifiers are required")
	}
	keys, ok := models.EngagementCountKeys[kind]
	if !ok {
		return false, errors.New("unknown engagement kind: " + string(kind))
	}
	if keys.AmountKey != "" {
		return false, errors.New("amount engagement kinds cannot be toggled")
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

func (r *EngagementRepository) GetEngagements(ctx context.Context, contentableType models.EngagementContentableType, contentableId uuid.UUID, engagementKind models.EngagementKind, cursor *time.Time, limit int) ([]models.EngagementDetail, *time.Time, error) {
	engagement, err := r.GetEngagement(ctx, contentableId, contentableType)
	if err != nil {

		return []models.EngagementDetail{}, nil, err
	}

	return r.GetEngagementDetailsWithCursor(ctx, engagement.ID, &engagementKind, cursor, limit)
}

func (r *EngagementRepository) AddEngagement(ctx context.Context, engagerID uuid.UUID, engageeID uuid.UUID, kind models.EngagementKind, contentableID uuid.UUID, contentableType models.EngagementContentableType) error {
	if r == nil || r.db == nil {
		return errors.New("engagement repository is not configured")
	}
	if engagerID == uuid.Nil || engageeID == uuid.Nil || contentableID == uuid.Nil || contentableType == "" {
		return errors.New("engagement identifiers are required")
	}
	keys, ok := models.EngagementCountKeys[kind]
	if !ok {
		return errors.New("unknown engagement kind: " + string(kind))
	}
	if keys.AmountKey != "" {
		return errors.New("amount engagement kinds require AddTip")
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
		return errors.New("transaction is nil")
	}
	if engagerID == uuid.Nil || engageeID == uuid.Nil || contentableID == uuid.Nil || contentableType == "" {
		return errors.New("amount engagement identifiers are required")
	}
	keys, ok := models.EngagementCountKeys[kind]
	if !ok {
		return errors.New("unknown engagement kind: " + string(kind))
	}
	if keys.AmountKey == "" {
		return errors.New("engagement kind does not support an amount: " + string(kind))
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
