package ports

import (
	"context"
	"core/models"
	"core/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type UserAccountRepository interface {
	ExistsByNameOrMail(input string) (bool, error)
	Create(user *models.User) error
	GetByID(userID uuid.UUID) (*models.User, error)
	GetByUserNameOrEmailOrUsername(input string) (*models.User, error)
	UpdateUser(user *models.User) error
	GetUserByPublicIdWithoutRelations(filters types.Filter) (*models.User, error)
	AddReferral(ctx context.Context, referrerID uuid.UUID, referredUserID uuid.UUID, rewardAmount decimal.Decimal) (*decimal.Decimal, error)
}
