package repositories

import (
	"core/application/ports"
	"core/constants"
	"core/models"
	"core/models/notifications"
	"strconv"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (r *PrivatePhotoRepository) SocialNotificationsTransactional() bool { return true }

func enqueuePrivatePhotoNotification(tx *gorm.DB, row models.EngagementDetail, details privatePhotoAccessDetails, response bool) error {
	ownerID, viewerID := row.EngagerID, row.EngageeID
	var users []models.User
	if err := tx.Select("id", "public_id").Where("id IN ?", []uuid.UUID{ownerID, viewerID}).Find(&users).Error; err != nil {
		return err
	}
	if len(users) != 2 {
		return constants.ErrPrivatePhotoNotificationParticipantsMissing
	}
	ids := map[uuid.UUID]int64{users[0].ID: users[0].PublicID, users[1].ID: users[1].PublicID}
	sender, receiver, kind, title, body := viewerID, ownerID, notifications.NotificationTypePrivatePhotoAccessRequest, "Private photo request", "Someone requested access to your private photos."
	if response {
		sender, receiver, kind, title, body = ownerID, viewerID, notifications.NotificationTypePrivatePhotoAccessResponse, "Private photo access updated", "Your private photo access is "+string(details.Status)+"."
		record := privatePhotoAccessRecord(row, details, ids[ownerID], ports.PrivatePhotoUser{ID: viewerID, PublicID: ids[viewerID]})
		if err := (&NotificationRepository{db: tx}).resolvePrivatePhotoRequestNotification(tx.Statement.Context, ownerID, record); err != nil {
			return err
		}
	}
	payload := notifications.NotificationPayload{Data: map[string]any{"request_id": strconv.FormatInt(details.PublicID, 10), "owner_id": strconv.FormatInt(ids[ownerID], 10), "viewer_id": strconv.FormatInt(ids[viewerID], 10), "status": details.Status, "access_status": details.Status}}
	_, err := persistQueuedNotification(tx, sender, receiver, kind, title, body, payload)
	return err
}
