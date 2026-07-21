package auth

import "core/helpers"

type TokenDecoder struct{}

func (TokenDecoder) DecodeUserPublicID(token string) (int64, error) {
	claims, err := helpers.DecodeUserJWT(token)
	if err != nil {
		return 0, err
	}
	return claims.PublicID, nil
}
