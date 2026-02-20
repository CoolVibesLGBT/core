package types

import (
	"context"
	"core/models"
	"core/models/post"

	"github.com/google/uuid"
)

type Filter struct {
	AuthUser *models.User
	Context  context.Context
	Domain   *string

	PostUUID uuid.UUID
	PostID   int64 // snowflakeid

	PostKind  post.PostKind
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
