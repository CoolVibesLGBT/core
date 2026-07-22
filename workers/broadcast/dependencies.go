package broadcast

import (
	"context"
	"core/application/ports"
	"core/models"
	mediamodel "core/models/media"
	"errors"

	"github.com/google/uuid"
)

// Repository is the broadcast-specific persistence port used by the worker.
type Repository interface {
	ResetBotBroadcastPresence(context.Context) error
	FindBroadcastUser(context.Context, []string) (*models.User, bool, error)
	UpdateBroadcastState(context.Context, uuid.UUID, []byte) error
}

// UserService contains only the application operations used while importing a
// broadcast user.
type UserService interface {
	CreateBotUser(context.Context, *models.User) (*models.User, error)
	UpdateAvatarFromURL(context.Context, string, *models.User) (*mediamodel.Media, error)
}

// Dependencies are assembled by the infrastructure composition root.
type Dependencies struct {
	Repository Repository
	Users      UserService
	Gateway    ports.BroadcastGateway
}

func (d Dependencies) validateFetcher() error {
	if err := d.validate(); err != nil {
		return err
	}
	if d.Gateway == nil {
		return ports.ErrBroadcastIntegrationDisabled
	}
	return nil
}

func (d Dependencies) validate() error {
	if d.Repository == nil {
		return errors.New("broadcast worker repository is not configured")
	}
	if d.Users == nil {
		return errors.New("broadcast worker user service is not configured")
	}
	return nil
}
