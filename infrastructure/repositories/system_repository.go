package repositories

import (
	"context"
	"core/application/ports"
	"core/helpers"
	"core/models"
	"core/models/payment"
	eventkinds "core/models/post/payloads"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SystemRepository struct {
	db *gorm.DB
}

func NewSystemRepository(db *gorm.DB) *SystemRepository {
	return &SystemRepository{db: db}
}

func (r *SystemRepository) GetPreferences(ctx context.Context) (models.PreferencesData, error) {
	var preferences models.PreferencesData
	err := r.db.WithContext(ctx).Model(&models.Preferences{}).Select("data").First(&preferences).Error
	return preferences, err
}

func (r *SystemRepository) GetEventKinds(ctx context.Context) ([]eventkinds.EventKind, error) {
	var items []eventkinds.EventKind
	err := r.db.WithContext(ctx).Model(&eventkinds.EventKind{}).Order("display_order ASC").Find(&items).Error
	return items, err
}

func (r *SystemRepository) GetReportKinds(ctx context.Context) ([]models.ReportKind, error) {
	var items []models.ReportKind
	err := r.db.WithContext(ctx).Model(&models.ReportKind{}).Order("display_order ASC").Find(&items).Error
	return items, err
}

func (r *SystemRepository) GetVapidPublicKey(ctx context.Context) (string, error) {
	key, err := helpers.CreateVapidKeys(r.db.WithContext(ctx))
	if err != nil {
		return "", err
	}
	return key.PublicKey, nil
}

func (r *SystemRepository) SaveVapidSubscription(ctx context.Context, userID uuid.UUID, newSub models.Subscription) error {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return err
	}

	items, err := helpers.DecodeJSONItems(user.Subscriptions)
	if err != nil {
		items = nil
	}

	exists := false
	for _, item := range items {
		var sub models.Subscription
		if err := json.Unmarshal(item, &sub); err != nil {
			continue
		}
		if sub.Endpoint == newSub.Endpoint {
			exists = true
			break
		}
	}

	if !exists {
		rawNewSub, err := json.Marshal(newSub)
		if err != nil {
			return err
		}
		items = append(items, rawNewSub)
	}

	subsJSON, err := json.Marshal(items)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Update("subscriptions", datatypes.JSON(subsJSON)).Error
}

func (r *SystemRepository) GetPaymentMethod(ctx context.Context) (*payment.PaymentMethod, error) {
	var pm payment.PaymentMethod
	if err := r.db.WithContext(ctx).First(&pm).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}
	return &pm, nil
}
