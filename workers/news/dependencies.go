package news

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/models"
	"core/models/post"
	"errors"
)

// UserService is the minimum application boundary needed by the news worker.
type UserService interface {
	FetchUserProfileByUsername(string) (*models.User, error)
}

// NewsService is the minimum application boundary needed to publish an item.
type NewsService interface {
	IsNewsExists(types.Filter) (bool, error)
	CreateNews(context.Context, ports.FormData, *models.User) (*post.Post, error)
}

// Notifier is an optional outbound notification boundary.
type Notifier interface {
	SendNews(*post.Post) error
}

// Dependencies are assembled by the infrastructure composition root.
type Dependencies struct {
	Users    UserService
	News     NewsService
	Notifier Notifier
}

func (d Dependencies) validate() error {
	if d.Users == nil {
		return errors.New("news worker user service is not configured")
	}
	if d.News == nil {
		return errors.New("news worker news service is not configured")
	}
	return nil
}
