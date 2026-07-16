package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	aggregateLockKey := fmt.Sprintf("engagement:%s:%s", contentableType, contentableID)
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

		var engagement models.Engagement
		err := tx.
			Where("contentable_id = ? AND contentable_type = ?", contentableID, contentableType).
			First(&engagement).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			engagement = models.Engagement{
				ID:              uuid.New(),
				ContentableID:   contentableID,
				ContentableType: contentableType,
				Counts:          datatypes.JSON([]byte("{}")),
			}
			if err := tx.Create(&engagement).Error; err != nil {
				return err
			}
		} else if err != nil {
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
		// 1. Engagement kaydını kontrol et
		var engagement models.Engagement
		err := tx.Where("id = ?", detail.EngagementID).First(&engagement).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("engagement record not found for engagement_id: " + detail.EngagementID.String())
		} else if err != nil {
			return err
		}

		// 2. Detayı oluştur
		if err := tx.Create(detail).Error; err != nil {
			return err
		}

		// 3. Engagement.Counts güncelle
		counts := map[string]interface{}{}
		if err := json.Unmarshal(engagement.Counts, &counts); err != nil {
			return err
		}

		keys, ok := models.EngagementCountKeys[models.EngagementKind(detail.Kind)]
		if !ok {
			return errors.New("unknown engagement kind: " + string(detail.Kind))
		}

		// Count artır
		if counts[keys.CountKey] == nil {
			counts[keys.CountKey] = int64(0)
		}
		countVal, _ := counts[keys.CountKey].(float64)
		counts[keys.CountKey] = int64(countVal) + 1

		// Amount artır (varsa)
		if keys.AmountKey != "" && detail.Details != nil {
			var detailsMap map[string]interface{}
			if err := json.Unmarshal(detail.Details, &detailsMap); err == nil {
				if amtVal, found := detailsMap["amount"]; found {
					amtDecimal, err := decimal.NewFromString(amtVal.(string))
					if err == nil {
						var currentAmount decimal.Decimal
						if val, ok := counts[keys.AmountKey]; ok {
							switch v := val.(type) {
							case float64:
								currentAmount = decimal.NewFromFloat(v)
							case string:
								currentAmount, _ = decimal.NewFromString(v)
							default:
								currentAmount = decimal.Zero
							}
						}
						newAmount := currentAmount.Add(amtDecimal)
						counts[keys.AmountKey] = newAmount.String()
					}
				}
			}
		}

		newCounts, err := json.Marshal(counts)
		if err != nil {
			return err
		}

		engagement.Counts = newCounts
		engagement.UpdatedAt = time.Now()

		if err := tx.Model(&models.Engagement{}).Where("id = ?", engagement.ID).Update("counts", engagement.Counts).Error; err != nil {
			return err
		}

		return nil
	})
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
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var detail models.EngagementDetail
		if err := tx.Where("id = ?", detailID).First(&detail).Error; err != nil {
			fmt.Println("RemoveEngagementDetail:Err:1", err)
			return err
		}

		var engagement models.Engagement
		if err := tx.Where("id = ?", detail.EngagementID).First(&engagement).Error; err != nil {
			fmt.Println("RemoveEngagementDetail:Err:2", err)
			return err
		}

		counts := map[string]interface{}{}
		if err := json.Unmarshal(engagement.Counts, &counts); err != nil {
			return err
		}

		keys, ok := models.EngagementCountKeys[models.EngagementKind(detail.Kind)]
		if !ok {
			return errors.New("unknown engagement kind: " + string(detail.Kind))
		}

		// Decrement count
		if counts[keys.CountKey] == nil {
			counts[keys.CountKey] = int64(0)
		}
		countVal, _ := counts[keys.CountKey].(float64)
		newCount := int64(countVal) - 1
		if newCount < 0 {
			newCount = 0
		}
		counts[keys.CountKey] = newCount

		// Decrement amount if applicable
		if keys.AmountKey != "" && detail.Details != nil {
			var detailsMap map[string]interface{}
			if err := json.Unmarshal(detail.Details, &detailsMap); err == nil {
				if amtVal, found := detailsMap["amount"]; found {
					amtDecimal, err := decimal.NewFromString(amtVal.(string))
					if err == nil {
						var currentAmount decimal.Decimal
						if val, ok := counts[keys.AmountKey]; ok {
							switch v := val.(type) {
							case float64:
								currentAmount = decimal.NewFromFloat(v)
							case string:
								currentAmount, _ = decimal.NewFromString(v)
							default:
								currentAmount = decimal.Zero
							}
						}

						newAmount := currentAmount.Sub(amtDecimal)
						if newAmount.IsNegative() {
							newAmount = decimal.Zero
						}
						counts[keys.AmountKey] = newAmount.String()
					}
				}
			}
		}

		// Marshal counts back
		newCounts, err := json.Marshal(counts)
		if err != nil {
			return err
		}
		engagement.Counts = newCounts
		engagement.UpdatedAt = time.Now()

		if err := tx.Model(&models.Engagement{}).Where("id = ?", engagement.ID).Update("counts", engagement.Counts).Error; err != nil {
			return err
		}

		// Delete detail
		if err := tx.Delete(&models.EngagementDetail{}, "id = ?", detailID).Error; err != nil {
			return err
		}

		return nil
	})
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
	// Engagement kaydını al veya oluştur
	var engagement models.Engagement

	/*
		err := r.db.WithContext(ctx).
			Where("contentable_id = ? AND contentable_type = ?", contentableID, contentableType).
			First(&engagement).Error
	*/
	err := r.db.WithContext(ctx).
		Where(&models.Engagement{
			ContentableID:   contentableID,
			ContentableType: contentableType,
		}).
		First(&engagement).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		engagement = models.Engagement{
			ID:              uuid.New(),
			ContentableID:   contentableID,
			ContentableType: contentableType,
			Counts:          datatypes.JSON([]byte("{}")),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if err := r.db.WithContext(ctx).Create(&engagement).Error; err != nil {
			return true, err
		}
	} else if err != nil {
		return false, err
	}

	// EngagementDetail kontrolü
	var existingDetail models.EngagementDetail
	//err = r.db.WithContext(ctx).
	//	Where("engagement_id = ? AND engager_id = ? AND engagee_id = ? AND kind = ?", engagement.ID, engagerID, engageeID, kind).
	// First(&existingDetail).Error

	err = r.db.WithContext(ctx).
		Where(&models.EngagementDetail{
			EngagementID: engagement.ID,
			EngagerID:    engagerID,
			EngageeID:    engageeID,
			Kind:         kind,
		}).
		First(&existingDetail).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Yoksa oluştur (toggle ON)
		newDetail := models.EngagementDetail{
			ID:           uuid.New(),
			EngagementID: engagement.ID,
			EngagerID:    engagerID,
			EngageeID:    engageeID, // İçeriğin sahibi (target user)
			Kind:         kind,
			CreatedAt:    time.Now(),
		}
		return true, r.CreateEngagementDetail(ctx, &newDetail)
	} else if err != nil {
		return false, err
	} else {
		// Var ise sil (toggle OFF)
		return false, r.RemoveEngagementDetail(ctx, existingDetail.ID)
	}
}

func (r *EngagementRepository) GetEngagements(ctx context.Context, contentableType models.EngagementContentableType, contentableId uuid.UUID, engagementKind models.EngagementKind, cursor *time.Time, limit int) ([]models.EngagementDetail, *time.Time, error) {
	engagement, err := r.GetEngagement(ctx, contentableId, contentableType)
	if err != nil {

		return []models.EngagementDetail{}, nil, err
	}

	return r.GetEngagementDetailsWithCursor(ctx, engagement.ID, &engagementKind, cursor, limit)
}

func (r *EngagementRepository) AddEngagement(ctx context.Context, engagerID uuid.UUID, engageeID uuid.UUID, kind models.EngagementKind, contentableID uuid.UUID, contentableType models.EngagementContentableType) error {
	// Engagement kaydını al veya oluştur
	var engagement models.Engagement

	err := r.db.WithContext(ctx).
		Where(&models.Engagement{
			ContentableID:   contentableID,
			ContentableType: contentableType,
		}).
		First(&engagement).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		engagement = models.Engagement{
			ID:              uuid.New(),
			ContentableID:   contentableID,
			ContentableType: contentableType,
			Counts:          datatypes.JSON([]byte("{}")),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if err := r.db.WithContext(ctx).Create(&engagement).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Yeni EngagementDetail oluştur (toggle yok, hep ekle)
	newDetail := models.EngagementDetail{
		ID:           uuid.New(),
		EngagementID: engagement.ID,
		EngagerID:    engagerID,
		EngageeID:    engageeID,
		Kind:         kind,
		CreatedAt:    time.Now(),
	}

	return r.CreateEngagementDetail(ctx, &newDetail)
}

func (r *EngagementRepository) AddTip(ctx context.Context, engagerID uuid.UUID, engageeID uuid.UUID, tipAmount decimal.Decimal, contentableID uuid.UUID, contentableType models.EngagementContentableType, kind models.EngagementKind) error {
	// kind mutlaka tipler arasında olmalı: EngagementKindTipGiven veya EngagementKindTipReceived

	// Engagement kaydını al veya oluştur
	var engagement models.Engagement

	err := r.db.WithContext(ctx).
		Where(&models.Engagement{
			ContentableID:   contentableID,
			ContentableType: contentableType,
		}).
		First(&engagement).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		engagement = models.Engagement{
			ID:              uuid.New(),
			ContentableID:   contentableID,
			ContentableType: contentableType,
			Counts:          datatypes.JSON([]byte("{}")),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if err := r.db.WithContext(ctx).Create(&engagement).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Details JSON içine amount koy
	detailsMap := map[string]string{
		"amount": tipAmount.String(),
	}
	detailsJSON, err := json.Marshal(detailsMap)
	if err != nil {
		return err
	}

	newDetail := models.EngagementDetail{
		ID:           uuid.New(),
		EngagementID: engagement.ID,
		EngagerID:    engagerID,
		EngageeID:    engageeID,
		Kind:         kind, // örn. models.EngagementKindTipGiven veya TipReceived
		Details:      datatypes.JSON(detailsJSON),
		CreatedAt:    time.Now(),
	}

	return r.CreateEngagementDetail(ctx, &newDetail)
}
