package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"core/application/ports"
	domainmedia "core/domain/media"
	"core/helpers"
	"core/models"
	"core/models/notifications"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	return r.SendNotificationToUser(ctx, sender, receiver, notifications.NotificationTypePrivatePhotoAccessRequest, title, body, payload)
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
	sendErr := r.SendNotificationToUser(ctx, sender, receiver, notifications.NotificationTypePrivatePhotoAccessResponse, title, body, payload)
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
		if fmt.Sprint(data["status"]) != string(domainmedia.PrivatePhotoAccessPending) && fmt.Sprint(data["access_status"]) != string(domainmedia.PrivatePhotoAccessPending) {
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

func (r *NotificationRepository) CreateNotification(ctx context.Context, senderUser uuid.UUID, receiverUser uuid.UUID, notifType, title, message string, payload notifications.NotificationPayload) (item *notifications.Notification, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		item, err = persistQueuedNotification(tx, senderUser, receiverUser, notifType, title, message, payload)
		return err
	})
	return item, err
}

func (r *NotificationRepository) SendNotificationToUser(ctx context.Context, sender models.User, receiver models.User, notificationType string, notificationTitle string, notificationMessage string, payload notifications.NotificationPayload) error {
	_, err := r.CreateNotification(ctx, sender.ID, receiver.ID, notificationType, notificationTitle, notificationMessage, payload)
	return err
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

func (r *NotificationRepository) FetchAndMarkShownNotifications(ctx context.Context, userID uuid.UUID, limit int) ([]notifications.Notification, error) {
	items := make([]notifications.Notification, 0)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock selection and acknowledgement together; concurrent polls claim
		// disjoint rows instead of showing the same notification twice.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("user_id = ? AND is_shown = FALSE AND deleted_at IS NULL", userID).
			Order("created_at DESC, id DESC").Limit(normalizedMatchLimit(limit)).Find(&items).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		now := time.Now().UTC()
		ids := make([]uuid.UUID, len(items))
		for i := range items {
			ids[i] = items[i].ID
			items[i].IsShown, items[i].ShownAt = true, &now
		}
		return tx.Model(&notifications.Notification{}).Where("id IN ?", ids).
			Updates(map[string]any{"is_shown": true, "shown_at": now}).Error
	})
	if err != nil {
		return nil, err
	}
	return items, nil
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
