package usecases

import (
	"core/application/ports"
	"core/models/notifications"

	"github.com/google/uuid"
)

type NotificationsService struct {
	notificationRepo ports.NotificationRepository
}

func NewNotificationsService(
	notificationRepo ports.NotificationRepository,
) *NotificationsService {
	return &NotificationsService{notificationRepo: notificationRepo}
}

func (service *NotificationsService) FetchNotifications(userID uuid.UUID, limit int) ([]notifications.Notification, error) {
	return service.notificationRepo.FetchAndMarkShownNotifications(userID, limit)
}
