package repositories

import (
	"context"
	"core/infrastructure/push"
	"core/models"
	"core/models/notifications"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

func TestNotificationOutboxRetriesOnlyFailedEndpointsIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	sender, receiver := privatePhotoTestUsers(t, db)
	subs := pushSubscriptions(2)
	for i := range subs {
		subs[i].Keys.Auth = "auth"
		subs[i].Keys.P256dh = "key"
	}
	raw, _ := json.Marshal(subs)
	if err := db.Model(&receiver).Update("subscriptions", datatypes.JSON(raw)).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewNotificationRepository(db, nil)
	calls := map[string]int{}
	tags := []string{}
	repo.newWebPushSender = func(*push.Options) (webPushSender, error) {
		return &trackingWebPushSender{send: func(_ context.Context, message *push.Push) error {
			calls[message.Endpoint]++
			var payload notifications.NotificationPayload
			if err := json.Unmarshal(message.Plaintext, &payload); err != nil {
				return err
			}
			tags = append(tags, payload.Tag)
			if message.Endpoint == subs[1].Endpoint && calls[message.Endpoint] == 1 {
				return errors.New("temporary provider failure")
			}
			return nil
		}}, nil
	}
	item, err := repo.CreateNotification(context.Background(), sender.ID, receiver.ID, "test", "Title", "Body", notifications.NotificationPayload{})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Add(time.Second)
	worked, err := repo.ProcessNextNotification(context.Background(), now)
	if !worked || err == nil {
		t.Fatalf("failed delivery not retained: %v %v", worked, err)
	}
	var pending notifications.Delivery
	if err := db.First(&pending, "id = ?", item.ID).Error; err != nil || pending.CompletedAt != nil || pending.Attempts != 1 {
		t.Fatalf("retry record=%+v %v", pending, err)
	}
	if worked, err := repo.ProcessNextNotification(context.Background(), now.Add(time.Second)); worked || err != nil {
		t.Fatalf("retry ignored backoff: %v %v", worked, err)
	}
	if worked, err := repo.ProcessNextNotification(context.Background(), now.Add(10*time.Second)); !worked || err != nil {
		t.Fatalf("retry failed: %v %v", worked, err)
	}
	if calls[subs[0].Endpoint] != 1 || calls[subs[1].Endpoint] != 2 {
		t.Fatalf("successful endpoint retried: %v", calls)
	}
	for _, tag := range tags {
		if tag != item.ID.String() {
			t.Fatalf("unstable push tag=%s", tag)
		}
	}
	if err := db.First(&pending, "id = ?", item.ID).Error; err != nil || pending.CompletedAt == nil || pending.Attempts != 2 {
		t.Fatalf("delivery not completed=%+v %v", pending, err)
	}
	var count int64
	if err := db.Model(&notifications.Notification{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("notification duplicated=%d %v", count, err)
	}
}

func TestNotificationOutboxConcurrentWorkersLeaseOneJobIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	sender, receiver := privatePhotoTestUsers(t, db)
	repo := NewNotificationRepository(db, nil)
	if err := repo.SendNotificationToUser(context.Background(), sender, receiver, "test", "Title", "Body", notifications.NotificationPayload{}); err != nil {
		t.Fatal(err)
	}
	results := make(chan bool, 8)
	errs := make(chan error, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Go(func() {
			worked, err := repo.ProcessNextNotification(context.Background(), time.Now().Add(time.Second))
			results <- worked
			errs <- err
		})
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	worked := 0
	for result := range results {
		if result {
			worked++
		}
	}
	if worked != 1 {
		t.Fatalf("workers consumed same job %d times", worked)
	}
}

func TestFollowRollsBackWhenNotificationCannotBeQueuedIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	actor, target := privatePhotoTestUsers(t, db)
	if err := db.Exec("DROP TABLE notification_deliveries").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewEngagementRepository(db).ApplyReciprocalUserInteraction(context.Background(), actor.ID, target.ID, socialIntent(t, "follow", true)); err == nil {
		t.Fatal("follow committed without durable notification")
	}
	var count int64
	if err := db.Model(&models.EngagementDetail{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("partial graph commit=%d %v", count, err)
	}
	if err := db.Model(&notifications.Notification{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("partial notification commit=%d %v", count, err)
	}
}

func TestNotificationOutboxRecoversExpiredWorkerLeaseIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	sender, receiver := privatePhotoTestUsers(t, db)
	repo := NewNotificationRepository(db, nil)
	item, err := repo.CreateNotification(context.Background(), sender.ID, receiver.ID, "test", "Title", "Body", notifications.NotificationPayload{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&notifications.Delivery{}).Where("id = ?", item.ID).Updates(map[string]any{"lease_token": uuid.New(), "lease_until": time.Now().Add(-time.Minute), "attempts": 1}).Error; err != nil {
		t.Fatal(err)
	}
	if worked, err := repo.ProcessNextNotification(context.Background(), time.Now().Add(time.Second)); err != nil || !worked {
		t.Fatalf("lease not recovered: %v %v", worked, err)
	}
	var row notifications.Delivery
	if err := db.First(&row, "id = ?", item.ID).Error; err != nil || row.CompletedAt == nil || row.Attempts != 2 {
		t.Fatalf("lease result=%+v %v", row, err)
	}
}

func TestNotificationOutboxPrunesExpiredSubscriptionIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	sender, receiver := privatePhotoTestUsers(t, db)
	repo := NewNotificationRepository(db, nil)
	subs := pushSubscriptions(1)
	subs[0].Keys.Auth = "auth"
	subs[0].Keys.P256dh = "key"
	raw, _ := json.Marshal(subs)
	if err := db.Model(&receiver).Update("subscriptions", datatypes.JSON(raw)).Error; err != nil {
		t.Fatal(err)
	}
	repo.newWebPushSender = func(*push.Options) (webPushSender, error) {
		return &trackingWebPushSender{send: func(context.Context, *push.Push) error { return push.ErrPushGone }}, nil
	}
	if _, err := repo.CreateNotification(context.Background(), sender.ID, receiver.ID, "test", "Title", "Body", notifications.NotificationPayload{}); err != nil {
		t.Fatal(err)
	}
	if worked, err := repo.ProcessNextNotification(context.Background(), time.Now().Add(time.Second)); err != nil || !worked {
		t.Fatalf("expired subscription not handled: %v %v", worked, err)
	}
	if err := db.First(&receiver, "id = ?", receiver.ID).Error; err != nil {
		t.Fatal(err)
	}
	active, err := decodePushSubscriptions(receiver.Subscriptions)
	if err != nil || len(active) != 0 {
		t.Fatalf("expired subscription retained=%+v %v", active, err)
	}
}
