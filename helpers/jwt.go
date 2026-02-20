package helpers

import (
	"core/constants"
	"core/models/jwtclaims"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

var zeroNamespace = uuid.Nil
var NameSpace = uuid.NewSHA1(zeroNamespace, []byte(constants.APPLICATION_NAME))

func GenerateUserJWT(user_id uuid.UUID, publicId int64) (string, error) {
	var jwtSecret = []byte(os.Getenv("USER_JWT_SECRET"))

	claims := &jwtclaims.UserJWTClaims{
		UserID:   user_id,  // uuid.UUID
		PublicID: publicId, // int64
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().AddDate(1, 0, 30).Unix(),
			NotBefore: time.Now().Unix(),
			Issuer:    constants.APPLICATION_NAME,
			Subject:   "AUTH",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, error := token.SignedString(jwtSecret)
	result := fmt.Sprintf("Bearer %s", tokenString)
	return result, error
}

func IsValidJWTFormat(token string) bool {
	var jwtRegex = regexp.MustCompile(`^[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+$`)
	return jwtRegex.MatchString(token)
}

func DecodeUserJWT(tokenString string) (*jwtclaims.UserJWTClaims, error) {
	if len(tokenString) > 1024 {
		return nil, errors.New("token too long")
	}

	if !IsValidJWTFormat(tokenString) {
		return nil, errors.New("invalid JWT format")
	}

	if strings.Count(tokenString, ".") != 2 {
		return nil, errors.New("invalid token format")
	}

	token, err := jwt.ParseWithClaims(tokenString, &jwtclaims.UserJWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(os.Getenv("USER_JWT_SECRET")), nil
	})
	if err != nil {
		fmt.Println("DecodeUserJWT:Error:1", err)
		return nil, err
	}
	if !token.Valid {
		fmt.Println("DecodeUserJWT:Error:2")
		return nil, errors.New("invalid jwt token")
	}
	claims, ok := token.Claims.(*jwtclaims.UserJWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token or claims")
	}
	if claims.ExpiresAt != 0 {
		expTime := time.Unix(claims.ExpiresAt, 0) // Unix timestamp -> time.Time
		if expTime.Before(time.Now()) {
			return nil, errors.New("token expired")
		}
	}
	return claims, nil
}
