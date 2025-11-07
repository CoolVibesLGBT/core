package repositories

import (
	"coolvibes/helpers"

	"gorm.io/gorm"
)

type NotificationRepository struct {
	db            *gorm.DB
	snowFlakeNode *helpers.Node
}

func (r *NotificationRepository) DB() *gorm.DB {
	return r.db
}

func (r *NotificationRepository) Node() *helpers.Node {
	return r.snowFlakeNode
}

func NewNotificationRepository(db *gorm.DB, snowFlakeNode *helpers.Node) *NotificationRepository {
	return &NotificationRepository{db: db, snowFlakeNode: snowFlakeNode}
}
