package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"coolvibes/models"
	"coolvibes/services/socket"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type EngagementRepository struct {
	db            *gorm.DB
	socketService *socket.SocketService
}

func NewEngagementRepository(
	db *gorm.DB,
	socketService *socket.SocketService,
) *EngagementRepository {
	return &EngagementRepository{
		db:            db,
		socketService: socketService,
	}
}

func (r *EngagementRepository) DB() *gorm.DB {
	return r.db
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
func (r *EngagementRepository) GetEngagement(ctx context.Context, contentableID uuid.UUID, contentableType string) (*models.Engagement, error) {
	var engagement models.Engagement
	if err := r.db.WithContext(ctx).Where("contentable_id = ? AND contentable_type = ?", contentableID, contentableType).First(&engagement).Error; err != nil {
		return nil, err
	}
	return &engagement, nil
}

// ListEngagementDetails lists all engagement details for a given engagement ID, optionally filtered by kind
func (r *EngagementRepository) ListEngagementDetails(ctx context.Context, engagementID uuid.UUID, kind *string) ([]models.EngagementDetail, error) {
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

func (r *EngagementRepository) HasUserEngaged(ctx context.Context, engagementID uuid.UUID, userID uuid.UUID, kind models.EngagementKind) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.EngagementDetail{}).
		Where("engagement_id = ? AND user_id = ? AND kind = ?", engagementID, userID, kind).
		Count(&count).Error

	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// userID,       // Kimin içeriği (post, video, vs) bu? İçeriğin sahibi (target user)
// engagerID,    // Etkileşimi yapan kullanıcı (engager)
func (r *EngagementRepository) ToggleEngagement(ctx context.Context, userID, engagerID uuid.UUID, kind models.EngagementKind, contentableID uuid.UUID, contentableType string) (bool, error) {
	// Engagement kaydını al veya oluştur
	var engagement models.Engagement
	err := r.db.WithContext(ctx).
		Where("contentable_id = ? AND contentable_type = ?", contentableID, contentableType).
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
	err = r.db.WithContext(ctx).
		Where("engagement_id = ? AND engager_id = ? AND kind = ?", engagement.ID, engagerID, kind).
		First(&existingDetail).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Yoksa oluştur (toggle ON)
		newDetail := models.EngagementDetail{
			ID:           uuid.New(),
			EngagementID: engagement.ID,
			EngagerID:    engagerID,
			RecipientID:  userID, // İçeriğin sahibi (target user)
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
