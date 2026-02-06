package types

import (
	"context"
	"coolvibes/models"
	"coolvibes/models/post"

	"github.com/google/uuid"
)

type Filter struct {
	AuthUser *models.User
	Context  context.Context
	PostKind post.PostKind

	UserUUID  uuid.UUID
	UserID    int64
	Search    *string
	Category  *string
	Name      *string
	City      *string
	Country   *string
	Latitude  *float64
	Longitude *float64
	Cursor    *int64
	Limit     int
}
