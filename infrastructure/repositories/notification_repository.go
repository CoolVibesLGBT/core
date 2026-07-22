package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"core/application/ports"
	"core/helpers"
	push "core/infrastructure/push"
	"core/models"
	"core/models/notifications"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationRepository struct {
	db               *gorm.DB
	snowFlakeNode    *helpers.Node
	newWebPushSender webPushSenderFactory
}

func (r *NotificationRepository) DB() *gorm.DB {
	return r.db
}

func (r *NotificationRepository) Node() *helpers.Node {
	return r.snowFlakeNode
}

func NewNotificationRepository(db *gorm.DB, snowFlakeNode *helpers.Node) *NotificationRepository {
	return &NotificationRepository{
		db:               db,
		snowFlakeNode:    snowFlakeNode,
		newWebPushSender: defaultWebPushSenderFactory,
	}
}

func (r *NotificationRepository) NotifyPrivatePhotoAccessRequested(ctx context.Context, owner, viewer ports.PrivatePhotoUser, request ports.PrivatePhotoAccessRecord) error {
	sender, receiver, err := r.privatePhotoNotificationUsers(ctx, viewer.ID, owner.ID)
	if err != nil {
		return err
	}
	title := "Private photo request"
	body := "@" + viewer.UserName + " requested access to your private photos."
	payload := notifications.NotificationPayload{
		Title: title,
		Body:  body,
		Data: map[string]any{
			"request_id":    fmt.Sprint(request.PublicID),
			"owner_id":      fmt.Sprint(owner.PublicID),
			"viewer_id":     fmt.Sprint(viewer.PublicID),
			"status":        request.Status,
			"access_status": request.Status,
		},
	}
	return r.SendNotificationToUser(sender, receiver, notifications.NotificationTypePrivatePhotoAccessRequest, title, body, payload)
}

func (r *NotificationRepository) NotifyPrivatePhotoAccessResponded(ctx context.Context, owner, viewer ports.PrivatePhotoUser, request ports.PrivatePhotoAccessRecord) error {
	sender, receiver, err := r.privatePhotoNotificationUsers(ctx, owner.ID, viewer.ID)
	if err != nil {
		return err
	}
	title := "Private photo request updated"
	body := "@" + owner.UserName + " " + string(request.Status) + " your private photo request."
	payload := notifications.NotificationPayload{
		Title: title,
		Body:  body,
		Data: map[string]any{
			"request_id":    fmt.Sprint(request.PublicID),
			"owner_id":      fmt.Sprint(owner.PublicID),
			"viewer_id":     fmt.Sprint(viewer.PublicID),
			"status":        request.Status,
			"access_status": request.Status,
		},
	}
	updateErr := r.resolvePrivatePhotoRequestNotification(ctx, owner.ID, request)
	sendErr := r.SendNotificationToUser(sender, receiver, notifications.NotificationTypePrivatePhotoAccessResponse, title, body, payload)
	return errors.Join(updateErr, sendErr)
}

func (r *NotificationRepository) resolvePrivatePhotoRequestNotification(ctx context.Context, ownerID uuid.UUID, request ports.PrivatePhotoAccessRecord) error {
	var items []notifications.Notification
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND type = ?", ownerID, notifications.NotificationTypePrivatePhotoAccessRequest).
		Order("created_at DESC").
		Find(&items).Error; err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, item := range items {
		data, ok := item.Payload.Data.(map[string]any)
		if !ok || fmt.Sprint(data["request_id"]) != fmt.Sprint(request.PublicID) {
			continue
		}
		if fmt.Sprint(data["status"]) != "pending" && fmt.Sprint(data["access_status"]) != "pending" {
			continue
		}
		data["status"] = request.Status
		data["access_status"] = request.Status
		payload := item.Payload
		payload.Data = data
		return r.db.WithContext(ctx).Model(&notifications.Notification{}).
			Where("id = ?", item.ID).
			Updates(map[string]any{
				"payload": payload,
				"is_read": true,
				"read_at": now,
			}).Error
	}
	return nil
}

func (r *NotificationRepository) privatePhotoNotificationUsers(ctx context.Context, senderID, receiverID uuid.UUID) (models.User, models.User, error) {
	var users []models.User
	if err := r.db.WithContext(ctx).Where("id IN ?", []uuid.UUID{senderID, receiverID}).Find(&users).Error; err != nil {
		return models.User{}, models.User{}, err
	}
	var sender, receiver models.User
	for _, user := range users {
		switch user.ID {
		case senderID:
			sender = user
		case receiverID:
			receiver = user
		}
	}
	if sender.ID == uuid.Nil || receiver.ID == uuid.Nil {
		return models.User{}, models.User{}, ports.ErrNotFound
	}
	return sender, receiver, nil
}

func (r *NotificationRepository) GetAllSubscriptions() ([]models.Subscription, error) {
	var users []models.User
	err := r.db.Find(&users).Error
	if err != nil {
		return nil, err
	}

	var allSubs []models.Subscription
	for _, user := range users {
		subs, err := decodePushSubscriptions(user.Subscriptions)
		if err == nil {
			allSubs = append(allSubs, subs...)
		}
	}

	return allSubs, nil
}

func (r *NotificationRepository) CreateNotification(senderUser uuid.UUID, receiverUser uuid.UUID, notifType, title, message string, payload notifications.NotificationPayload) (*notifications.Notification, error) {
	notification := &notifications.Notification{
		ID:        uuid.New(),
		SenderID:  &senderUser,
		UserID:    receiverUser,
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

func (r *NotificationRepository) SendNotificationToUser(sender models.User, receiver models.User, notificationType string, notificationTitle string, notificationMessage string, payload notifications.NotificationPayload) error {

	_, err := r.CreateNotification(sender.ID, receiver.ID, notificationType, notificationTitle, notificationMessage, payload)
	if err != nil {
		return fmt.Errorf("notification cannot be saved: %w", err)
	}

	var subscriptions []models.Subscription
	if len(receiver.Subscriptions) == 0 {
		return fmt.Errorf("user has no subscriptions")
	}

	subscriptions, err = decodePushSubscriptions(receiver.Subscriptions)
	if err != nil {
		return fmt.Errorf("failed to unmarshal subscriptions: %w", err)
	}
	if len(subscriptions) == 0 {
		return fmt.Errorf("user has no subscriptions")
	}

	// Payload'u JSON string haline getir
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	deliveryCtx, cancelDelivery := context.WithTimeout(context.Background(), notificationPushBatchTimeout)
	defer cancelDelivery()

	vapidKeyInfo, err := helpers.CreateVapidKeys(r.db.WithContext(deliveryCtx))
	if err != nil {
		return fmt.Errorf("failed to get vapid key: %w", err)
	}
	opts := push.NewOptions().
		ApplyKeys(vapidKeyInfo.PublicKey, vapidKeyInfo.PrivateKey).
		SetDeliveryTimeout(push.DefaultDeliveryTimeout)

	newSender := r.newWebPushSender
	if newSender == nil {
		newSender = defaultWebPushSenderFactory
	}
	pb, err := newSender(opts)
	if err != nil {
		return fmt.Errorf("failed to create push service: %w", err)
	}

	attempted, failed := deliverWebPushBatch(
		deliveryCtx,
		pb,
		subscriptions,
		payloadBytes,
		notificationPushMaxConcurrent,
	)
	if failed > 0 || attempted < len(subscriptions) {
		log.Printf(
			"[Notification] push delivery incomplete: attempted=%d total=%d failed=%d deadline_exceeded=%t",
			attempted,
			len(subscriptions),
			failed,
			errors.Is(deliveryCtx.Err(), context.DeadlineExceeded),
		)
	}

	return nil
}

func decodePushSubscriptions(raw []byte) ([]models.Subscription, error) {
	items, err := helpers.DecodeJSONItems(raw)
	if err != nil {
		return nil, err
	}

	subscriptions := make([]models.Subscription, 0, len(items))
	for _, item := range items {
		var sub models.Subscription
		if err := json.Unmarshal(item, &sub); err != nil {
			continue
		}

		if strings.TrimSpace(sub.Endpoint) == "" {
			continue
		}
		if strings.TrimSpace(sub.Keys.Auth) == "" || strings.TrimSpace(sub.Keys.P256dh) == "" {
			continue
		}

		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
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

func (r *NotificationRepository) MarkNotificationAsRead(notificationID string) error {
	id, err := uuid.Parse(notificationID)
	if err != nil {
		return err
	}
	now := time.Now()
	return r.db.Model(&notifications.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}
