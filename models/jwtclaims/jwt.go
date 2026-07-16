package jwtclaims

import (
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

type JWTClaims struct {
	Name string `json:"name"`
	jwt.StandardClaims
}

type UserJWTClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	PublicID int64     `json:"public_id"`
	jwt.StandardClaims
}
