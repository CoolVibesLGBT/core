package services

import (
	"coolvibes/models/notifications"
	"coolvibes/repositories"

	"github.com/google/uuid"
)

type NotificationsService struct {
	notificationRepo *repositories.NotificationRepository
}

func NewNotificationsService(
	notificationRepo *repositories.NotificationRepository,
) *NotificationsService {
	return &NotificationsService{notificationRepo: notificationRepo}
}

func (service *NotificationsService) FetchNotifications(userID uuid.UUID, limit int) ([]notifications.Notification, error) {
	return service.notificationRepo.FetchAndMarkShownNotifications(userID, limit)
}
