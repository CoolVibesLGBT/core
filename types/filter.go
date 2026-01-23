package types

import (
	"context"
	"coolvibes/models"
)

type Filter struct {
	AuthUser  *models.User
	Context   context.Context
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
