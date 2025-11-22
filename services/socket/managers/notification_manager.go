// socket/notification_store.go

package managers

import (
	"coolvibes/repositories"

	"gorm.io/gorm"
)

type NotificationManager struct {
	db               *gorm.DB
	notificationRepo *repositories.NotificationRepository
}

func NewNotificationManager(db *gorm.DB, notificationRepo *repositories.NotificationRepository) *NotificationManager {
	return &NotificationManager{db: db, notificationRepo: notificationRepo}
}

func (m *NotificationManager) DB() *gorm.DB {
	return m.db
}

func (m *NotificationManager) NotificationRepository() *repositories.NotificationRepository {
	return m.notificationRepo
}

func (m *NotificationManager) MarkNotificationAsRead(notificationID string) error {
	return m.notificationRepo.MarkNotificationAsRead(notificationID)
}
