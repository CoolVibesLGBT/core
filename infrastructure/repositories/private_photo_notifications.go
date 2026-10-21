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

func enqueuePrivatePhotoNotification(tx *gorm.DB, request models.PrivatePhotoAccessRequest, response bool) error {
	var users []models.User
	if err := tx.Select("id", "public_id").Where("id IN ?", []uuid.UUID{request.OwnerID, request.ViewerID}).Find(&users).Error; err != nil {
		return err
	}
	if len(users) != 2 {
		return constants.ErrPrivatePhotoNotificationParticipantsMissing
	}
	ids := map[uuid.UUID]int64{users[0].ID: users[0].PublicID, users[1].ID: users[1].PublicID}
	sender, receiver, kind, title, body := request.ViewerID, request.OwnerID, notifications.NotificationTypePrivatePhotoAccessRequest, "Private photo request", "Someone requested access to your private photos."
	if response {
		sender, receiver, kind, title, body = request.OwnerID, request.ViewerID, notifications.NotificationTypePrivatePhotoAccessResponse, "Private photo access updated", "Your private photo access is "+string(request.Status)+"."
		if err := (&NotificationRepository{db: tx}).resolvePrivatePhotoRequestNotification(tx.Statement.Context, request.OwnerID, privatePhotoAccessRecord(request, ids[request.OwnerID], ports.PrivatePhotoUser{ID: request.ViewerID, PublicID: ids[request.ViewerID]})); err != nil {
			return err
		}
	}
	payload := notifications.NotificationPayload{Data: map[string]any{"request_id": strconv.FormatInt(request.PublicID, 10), "owner_id": strconv.FormatInt(ids[request.OwnerID], 10), "viewer_id": strconv.FormatInt(ids[request.ViewerID], 10), "status": request.Status, "access_status": request.Status}}
	_, err := persistQueuedNotification(tx, sender, receiver, kind, title, body, payload)
	return err
}
