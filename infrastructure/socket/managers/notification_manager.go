// socket/notification_store.go

package managers

import (
	"core/application/ports"
)

type NotificationManager struct {
	notificationRepo ports.NotificationReadMarker
}

func NewNotificationManager(notificationRepo ports.NotificationReadMarker) *NotificationManager {
	return &NotificationManager{notificationRepo: notificationRepo}
}

func (m *NotificationManager) MarkNotificationAsRead(notificationID string) error {
	return m.notificationRepo.MarkNotificationAsRead(notificationID)
}
