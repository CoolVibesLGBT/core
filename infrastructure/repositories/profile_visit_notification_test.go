package repositories

import (
	"context"
	"core/models"
	"core/models/notifications"
	"strconv"
	"testing"
)

func TestProfileVisitNotificationOnceIntegration(t *testing.T) {
	for _, language := range []string{"tr-TR", "en"} {
		t.Run(language, func(t *testing.T) {
			db := socialGraphIntegrationDB(t)
			viewer, target := privatePhotoTestUsers(t, db)
			if err := db.Model(&target).Update("default_language", language).Error; err != nil {
				t.Fatal(err)
			}
			repo := NewEngagementRepository(db)
			for i, self := range []bool{true, false, false} {
				targetID := target.ID
				if self {
					targetID = viewer.ID
				}
				counted, err := repo.RecordViewOnce(context.Background(), viewer.ID, targetID, models.EngagementKindViewReceived, targetID, models.EngagementContentableTypeUser)
				if err != nil || counted != (i == 1) {
					t.Fatalf("view %d: counted=%v err=%v", i, counted, err)
				}
			}
			var items []notifications.Notification
			if err := db.Find(&items).Error; err != nil {
				t.Fatal(err)
			}
			if len(items) != 1 {
				t.Fatalf("notifications=%d, want one", len(items))
			}
			item := items[0]
			if item.Type != notifications.NotificationTypeProfileVisit || item.UserID != target.ID || item.SenderID == nil || *item.SenderID != viewer.ID {
				t.Fatalf("wrong notification recipient/type: %+v", item)
			}
			body := "@" + viewer.UserName + " viewed your profile."
			if language == "tr-TR" {
				body = "@" + viewer.UserName + " profilinize baktı."
			}
			if item.Message != body || item.Payload.Body != body || item.Payload.URL != "/"+viewer.UserName {
				t.Fatalf("wrong notification content: %+v", item)
			}
			data, ok := item.Payload.Data.(map[string]any)
			if !ok || data["user_id"] != strconv.FormatInt(viewer.PublicID, 10) {
				t.Fatalf("wrong visitor payload: %#v", item.Payload.Data)
			}
			var deliveries []notifications.Delivery
			if err := db.Find(&deliveries).Error; err != nil || len(deliveries) != 1 || deliveries[0].ID != item.ID {
				t.Fatalf("delivery not queued: %+v %v", deliveries, err)
			}
		})
	}
}

func TestProfileVisitNotificationFailureRollsBackIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	viewer, target := privatePhotoTestUsers(t, db)
	if err := db.Exec("DROP TABLE notification_deliveries").Error; err != nil {
		t.Fatal(err)
	}
	_, err := NewEngagementRepository(db).RecordViewOnce(context.Background(), viewer.ID, target.ID, models.EngagementKindViewReceived, target.ID, models.EngagementContentableTypeUser)
	if err == nil {
		t.Fatal("expected queue failure")
	}
	for _, model := range []any{&notifications.Notification{}, &models.EngagementDetail{}} {
		var count int64
		if err := db.Model(model).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("write not rolled back: %T count=%d err=%v", model, count, err)
		}
	}
}
