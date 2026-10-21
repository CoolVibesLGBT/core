package news

import (
	"context"
	"core/application/types"
	domainmedia "core/domain/media"
	"core/models"
	"errors"
)

// UserService is the minimum application boundary needed by the news worker.
type UserService interface {
	FetchUserProfileByUsername(string) (*models.User, error)
}

// NewsService is the minimum application boundary needed to publish an item.
type NewsService interface {
	IsNewsExists(types.Filter) (bool, error)
	CreateNews(context.Context, map[string][]string, []*domainmedia.UploadedFile, *models.User) (*types.PublicPost, error)
}

// Notifier is an optional outbound notification boundary.
type Notifier interface {
	SendNews(post *types.PublicPost, localImages []string) error
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
