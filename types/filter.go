package types

import (
	"context"
	"coolvibes/models"

	"github.com/google/uuid"
)

type Filter struct {
	AuthUser  *models.User
	Context   context.Context
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
