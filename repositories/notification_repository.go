package repositories

import (
	"coolvibes/helpers"
	"coolvibes/models"
	"coolvibes/models/notifications"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	push "coolvibes/push"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationRepository struct {
	db            *gorm.DB
	snowFlakeNode *helpers.Node
}

func (r *NotificationRepository) DB() *gorm.DB {
	return r.db
}

func (r *NotificationRepository) Node() *helpers.Node {
	return r.snowFlakeNode
}

func NewNotificationRepository(db *gorm.DB, snowFlakeNode *helpers.Node) *NotificationRepository {
	return &NotificationRepository{db: db, snowFlakeNode: snowFlakeNode}
}

func (r *NotificationRepository) GetAllSubscriptions() ([]models.Subscription, error) {
	var users []models.User
	err := r.db.Find(&users).Error
	if err != nil {
		return nil, err
	}

	var allSubs []models.Subscription
	for _, user := range users {
		var subs []models.Subscription
		if len(user.Subscriptions) > 0 {
			if err := json.Unmarshal(user.Subscriptions, &subs); err == nil {
				allSubs = append(allSubs, subs...)
			}
		}
	}

	return allSubs, nil
}

func (r *NotificationRepository) CreateNotification(userID uuid.UUID, notifType, title, message string, payload notifications.NotificationPayload) (*notifications.Notification, error) {
	notification := &notifications.Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		Message:   message,
		Payload:   payload,
		IsRead:    false,
		IsShown:   false,
		CreatedAt: time.Now(),
	}

	if err := r.db.Create(notification).Error; err != nil {
		return nil, err
	}

	return notification, nil
}

func (r *NotificationRepository) SendNotificationToUser(user models.User, notificationType string, notificationTitle string, notificationMessage string, payload notifications.NotificationPayload) error {
	// Kullanıcının kayıtlı subscriptionlarını json'dan ayıkla

	notification, err := r.CreateNotification(user.ID, notificationType, notificationTitle, notificationMessage, payload)
	if err != nil {
		return fmt.Errorf("notification cannot be saved: %w", err)
	}

	fmt.Println(notification.ID)

	var subscriptions []models.Subscription
	if len(user.Subscriptions) == 0 {
		return fmt.Errorf("user has no subscriptions")
	}

	err = json.Unmarshal(user.Subscriptions, &subscriptions)
	if err != nil {
		return fmt.Errorf("failed to unmarshal subscriptions: %w", err)
	}

	// Payload'u JSON string haline getir
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	vapidKeyInfo, err := helpers.CreateVapidKeys(r.db)
	if err != nil {
		return fmt.Errorf("failed to get vapid key: %w", err)
	}

	// Her subscription için push notification gönder
	for _, sub := range subscriptions {
		// subscription'dan push.Subscription yapısına dönüştür
		pushSub := &push.Subscription{
			Endpoint: sub.Endpoint,
			Keys: push.Keys{
				P256dh: sub.Keys.P256dh, // küçük d ile
				Auth:   sub.Keys.Auth,
			},
		}

		// push.Options oluştur
		options := &push.Options{
			Subscriber:      "mailto:ersanyakit@gmail.com", // İstersen helpers'dan da çekebilirsin
			VAPIDPublicKey:  vapidKeyInfo.PublicKey,
			VAPIDPrivateKey: vapidKeyInfo.PrivateKey,
			TTL:             60,
		}

		// Push bildirimi gönder
		resp, err := push.SendNotification(payloadBytes, pushSub, options)
		if err != nil {
			fmt.Printf("Failed to send notification to %s: %v\n", sub.Endpoint, err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			fmt.Printf("Unexpected status code %d when sending to %s\n", resp.StatusCode, sub.Endpoint)
		}
	}

	return nil
}

func (r *NotificationRepository) FetchAndMarkShownNotifications(userID uuid.UUID, limit int) ([]notifications.Notification, error) {
	var notificationList []notifications.Notification

	// 1. Gösterilmemiş bildirimleri çek
	err := r.db.
		Where("user_id = ? AND is_shown = false", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&notificationList).Error
	if err != nil {
		return nil, err
	}

	// 2. Çekilen bildirimlerin IDsini topla
	var ids []uuid.UUID
	for _, n := range notificationList {
		ids = append(ids, n.ID)
	}

	// 3. Eğer varsa, bu bildirimleri 'shown' olarak işaretle
	if len(ids) > 0 {
		err = r.db.Model(&notifications.Notification{}).
			Where("id IN ?", ids).
			Update("is_shown", true).Error
		if err != nil {
			return nil, err
		}
	}

	return notificationList, nil
}
