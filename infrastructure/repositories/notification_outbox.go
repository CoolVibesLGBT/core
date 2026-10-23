package repositories

import (
	"context"
	"core/constants"
	domainmedia "core/domain/media"
	domainuser "core/domain/user"
	"core/helpers"
	"core/infrastructure/push"
	"core/models"
	"core/models/notifications"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func persistQueuedNotification(tx *gorm.DB, sender, receiver uuid.UUID, kind, title, body string, payload notifications.NotificationPayload) (*notifications.Notification, error) {
	now := time.Now().UTC()
	id := uuid.New()
	payload.Title, payload.Body, payload.Tag = title, body, id.String()
	item := &notifications.Notification{ID: id, SenderID: &sender, UserID: receiver, Type: kind,
		Title: title, Message: body, Payload: payload, CreatedAt: now}
	if err := tx.Omit(clause.Associations).Create(item).Error; err != nil {
		return nil, err
	}
	job := notifications.Delivery{ID: id, AvailableAt: now, CreatedAt: now, UpdatedAt: now}
	if err := tx.Create(&job).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func enqueueProfileVisitNotification(tx *gorm.DB, viewerID, targetID uuid.UUID) error {
	var viewer, target models.User
	if err := tx.Select("id", "public_id", "user_name").First(&viewer, "id = ?", viewerID).Error; err != nil {
		return err
	}
	if err := tx.Select("id", "default_language").First(&target, "id = ?", targetID).Error; err != nil {
		return err
	}
	title, body := "Profile visit", "@"+viewer.UserName+" viewed your profile."
	if strings.EqualFold(strings.Split(strings.ReplaceAll(target.DefaultLanguage, "_", "-"), "-")[0], "tr") {
		title, body = "Profil ziyareti", "@"+viewer.UserName+" profilinize baktı."
	}
	payload := notifications.NotificationPayload{
		URL:  "/" + viewer.UserName,
		Data: map[string]string{"user_id": strconv.FormatInt(viewer.PublicID, 10)},
	}
	_, err := persistQueuedNotification(tx, viewerID, targetID, notifications.NotificationTypeProfileVisit, title, body, payload)
	return err
}

func enqueueMatchNotifications(tx *gorm.DB, first, second uuid.UUID) error {
	var users []models.User
	if err := tx.Select("id", "public_id").Where("id IN ?", []uuid.UUID{first, second}).Find(&users).Error; err != nil {
		return err
	}
	if len(users) != 2 {
		return constants.ErrMatchNotificationParticipantsMissing
	}
	for i := range users {
		sender, receiver := users[i], users[1-i]
		payload := notifications.NotificationPayload{Data: map[string]string{"match_user_id": strconv.FormatInt(sender.PublicID, 10)}}
		if _, err := persistQueuedNotification(tx, sender.ID, receiver.ID, notifications.NotificationTypeNewMatch, "It's a Match!", "You have a new match.", payload); err != nil {
			return err
		}
	}
	return nil
}

func enqueueFollowNotifications(tx *gorm.DB, actor, target uuid.UUID, enabled bool) error {
	var users []models.User
	if err := tx.Select("id", "public_id", "user_name").Where("id IN ?", []uuid.UUID{actor, target}).Find(&users).Error; err != nil {
		return err
	}
	if len(users) != 2 {
		return constants.ErrFollowNotificationParticipantsMissing
	}
	byID := map[uuid.UUID]models.User{users[0].ID: users[0], users[1].ID: users[1]}
	kind, title, action := notifications.NotificationTypeFollow, "New follower", " followed "
	if !enabled {
		kind, title, action = notifications.NotificationTypeUnFollow, "Follow removed", " unfollowed "
	}
	body := "@" + byID[actor].UserName + action + "@" + byID[target].UserName + "."
	for _, pair := range [][2]uuid.UUID{{actor, target}, {target, actor}} {
		payload := notifications.NotificationPayload{Data: map[string]string{"user_id": strconv.FormatInt(byID[pair[0]].PublicID, 10)}}
		if _, err := persistQueuedNotification(tx, pair[0], pair[1], kind, title, body, payload); err != nil {
			return err
		}
	}
	return nil
}

// ProcessNextNotification leases a job without holding a DB transaction during
// network calls. A crashed worker's lease expires and another worker retries.
func (r *NotificationRepository) ProcessNextNotification(ctx context.Context, now time.Time) (bool, error) {
	var job notifications.Delivery
	token := uuid.New()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("completed_at IS NULL AND available_at <= ? AND (lease_until IS NULL OR lease_until <= ?)", now, now).
			Order("available_at ASC, id ASC").Take(&job)
		if result.Error != nil {
			return result.Error
		}
		until := now.Add(30 * time.Second)
		return tx.Model(&job).Updates(map[string]any{"lease_until": until, "lease_token": token, "attempts": gorm.Expr("attempts + 1")}).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	job.Attempts++
	remaining, deliveryErr := r.deliverQueuedNotification(ctx, job)
	updates := map[string]any{"lease_until": nil, "lease_token": nil, "last_error": "", "remaining_subscriptions": nil}
	if deliveryErr == nil {
		updates["completed_at"] = now
	} else {
		delay := 5 * time.Second * time.Duration(1<<min(job.Attempts-1, 9))
		updates["available_at"] = now.Add(delay)
		updates["last_error"] = deliveryErr.Error()
		if remaining != nil {
			encoded, err := json.Marshal(remaining)
			if err != nil {
				return true, err
			}
			updates["remaining_subscriptions"] = encoded
		}
	}
	// Shutdown/cancellation must not erase the durable job. An expired lease
	// remains retryable even if this acknowledgement itself cannot be saved.
	ackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	defer cancel()
	if err := r.db.WithContext(ackCtx).Model(&notifications.Delivery{}).Where("id = ? AND lease_token = ?", job.ID, token).Updates(updates).Error; err != nil {
		return true, err
	}
	return true, deliveryErr
}

func (r *NotificationRepository) deliverQueuedNotification(ctx context.Context, job notifications.Delivery) ([]models.Subscription, error) {
	var item notifications.Notification
	if err := r.db.WithContext(ctx).Where("id = ?", job.ID).Take(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	var receiver models.User
	if err := r.db.WithContext(ctx).Where("id = ? AND user_role NOT IN ?", item.UserID, []constants.UserRole{constants.UserRoleBanned, constants.UserRoleDeleted, constants.UserRolePending}).Take(&receiver).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if item.DeletedAt != nil {
		return nil, nil
	}
	current, err := r.notificationStillCurrent(ctx, item)
	if err != nil {
		return nil, err
	}
	if !current {
		return nil, nil
	}
	if item.SenderID != nil {
		blocked, err := (&PrivatePhotoRepository{db: r.db}).ArePrivatePhotoUsersBlocked(ctx, *item.SenderID, item.UserID)
		if err != nil {
			return nil, err
		}
		if blocked {
			return nil, nil
		}
	}
	subscriptions, err := decodePushSubscriptions(receiver.Subscriptions)
	if err != nil {
		return nil, err
	}
	if job.RemainingSubscriptions != nil {
		var previous []models.Subscription
		if err := json.Unmarshal(job.RemainingSubscriptions, &previous); err != nil {
			return nil, err
		}
		active := make(map[string]models.Subscription, len(subscriptions))
		for _, sub := range subscriptions {
			active[sub.Endpoint] = sub
		}
		subscriptions = nil
		for _, sub := range previous {
			if current, ok := active[sub.Endpoint]; ok {
				subscriptions = append(subscriptions, current)
			}
		}
	}
	if len(subscriptions) == 0 {
		return nil, nil
	}
	deliveryCtx, cancel := context.WithTimeout(ctx, notificationPushBatchTimeout)
	defer cancel()
	keys, err := helpers.CreateVapidKeys(r.db.WithContext(deliveryCtx))
	if err != nil {
		return subscriptions, err
	}
	factory := r.newWebPushSender
	if factory == nil {
		factory = defaultWebPushSenderFactory
	}
	sender, err := factory(push.NewOptions().ApplyKeys(keys.PublicKey, keys.PrivateKey).SetDeliveryTimeout(push.DefaultDeliveryTimeout))
	if err != nil {
		return subscriptions, err
	}
	payload, err := json.Marshal(item.Payload)
	if err != nil {
		return subscriptions, err
	}
	failed := make([]models.Subscription, 0)
	for _, sub := range subscriptions {
		if err := sender.SendContext(deliveryCtx, &push.Push{Endpoint: sub.Endpoint, Auth: sub.Keys.Auth, P256DH: sub.Keys.P256dh, Plaintext: payload}); err != nil {
			if errors.Is(err, push.ErrPushGone) || errors.Is(err, push.ErrPushNotFound) {
				if pruneErr := r.removeExpiredSubscription(deliveryCtx, item.UserID, sub); pruneErr == nil {
					continue
				}
			}
			failed = append(failed, sub)
		}
	}
	if len(failed) > 0 {
		return failed, constants.ErrPushDeliveryFailed.Format(len(failed))
	}
	return nil, nil
}

func (r *NotificationRepository) notificationStillCurrent(ctx context.Context, item notifications.Notification) (bool, error) {
	if item.SenderID == nil {
		return true, nil
	}
	if item.Type == notifications.NotificationTypeNewMatch {
		return userMatchDetailExists(r.db.WithContext(ctx), *item.SenderID, item.UserID, models.EngagementKindMatched)
	}
	if item.Type == notifications.NotificationTypeFriendRequest || item.Type == notifications.NotificationTypeFriendRequestAccepted {
		data, ok := item.Payload.Data.(map[string]any)
		if !ok {
			return false, nil
		}
		id, err := strconv.ParseInt(fmt.Sprint(data["request_id"]), 10, 64)
		if err != nil {
			return false, nil
		}
		status := domainuser.FriendshipPending
		if item.Type == notifications.NotificationTypeFriendRequestAccepted {
			status = domainuser.FriendshipAccepted
		}
		var count int64
		err = r.db.WithContext(ctx).Model(&models.Friendship{}).Where("public_id = ? AND status = ?", id, status).Count(&count).Error
		return count > 0, err
	}
	if item.Type == notifications.NotificationTypePrivatePhotoAccessRequest || item.Type == notifications.NotificationTypePrivatePhotoAccessResponse {
		data, ok := item.Payload.Data.(map[string]any)
		if !ok {
			return false, nil
		}
		id, err := strconv.ParseInt(fmt.Sprint(data["request_id"]), 10, 64)
		if err != nil {
			return false, nil
		}
		var count int64
		status := domainmedia.PrivatePhotoAccessPending
		if item.Type == notifications.NotificationTypePrivatePhotoAccessResponse {
			status, err = domainmedia.ParsePrivatePhotoAccessStatus(fmt.Sprint(data["access_status"]))
			if err != nil {
				return false, nil
			}
		}
		err = r.db.WithContext(ctx).Model(&models.PrivatePhotoAccessRequest{}).Where("public_id = ? AND status = ?", id, status).Count(&count).Error
		return count > 0, err
	}
	return true, nil
}

func (r *NotificationRepository) removeExpiredSubscription(ctx context.Context, userID uuid.UUID, expired models.Subscription) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id", "subscriptions").Where("id = ?", userID).Take(&user).Error; err != nil {
			return err
		}
		subs, err := decodePushSubscriptions(user.Subscriptions)
		if err != nil {
			return err
		}
		active := make([]models.Subscription, 0, len(subs))
		for _, sub := range subs {
			if sub.Endpoint == expired.Endpoint && sub.Keys.Auth == expired.Keys.Auth && sub.Keys.P256dh == expired.Keys.P256dh {
				continue
			}
			active = append(active, sub)
		}
		raw, err := json.Marshal(active)
		if err != nil {
			return err
		}
		return tx.Model(&user).UpdateColumn("subscriptions", datatypes.JSON(raw)).Error
	})
}
