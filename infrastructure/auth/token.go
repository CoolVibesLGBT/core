package auth

import (
	"core/helpers"

	"github.com/google/uuid"
)

type TokenIssuer struct{}

func (TokenIssuer) GenerateUserToken(userID uuid.UUID, publicID int64) (string, error) {
	return helpers.GenerateUserJWT(userID, publicID)
}
