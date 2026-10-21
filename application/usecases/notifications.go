package usecases

import (
	"context"
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

func (service *NotificationsService) FetchNotifications(ctx context.Context, userID uuid.UUID, limit int) ([]notifications.Notification, error) {
	return service.notificationRepo.FetchAndMarkShownNotifications(ctx, userID, limit)
}
