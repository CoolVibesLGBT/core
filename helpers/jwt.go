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

const (
	userJWTSecretEnvironmentKey = "USER_JWT_SECRET"
	userJWTMinimumSecretBytes   = 32
	userJWTSubject              = "AUTH"
	userJWTClockSkew            = 30 * time.Second
)

var (
	ErrUserJWTSecretRequired = errors.New("USER_JWT_SECRET is required")
	ErrUserJWTSecretTooShort = fmt.Errorf("USER_JWT_SECRET must be at least %d bytes", userJWTMinimumSecretBytes)
	ErrInvalidUserJWTClaims  = errors.New("invalid user JWT claims")
	userJWTFormat            = regexp.MustCompile(`^[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+$`)
)

// ValidateUserJWTConfiguration is called during application startup and by
// the token helpers themselves. Keeping the runtime checks makes token
// handling fail closed even when a helper is used outside the HTTP bootstrap.
func ValidateUserJWTConfiguration() error {
	_, err := userJWTSecretFromEnvironment()
	return err
}

func userJWTSecretFromEnvironment() ([]byte, error) {
	secret := strings.TrimSpace(os.Getenv(userJWTSecretEnvironmentKey))
	if secret == "" {
		return nil, ErrUserJWTSecretRequired
	}
	if len([]byte(secret)) < userJWTMinimumSecretBytes {
		return nil, ErrUserJWTSecretTooShort
	}

	switch strings.ToLower(secret) {
	case "changeme", "change-me", "replace-me", "replace-with-a-secure-secret", "your-secret-here":
		return nil, ErrUserJWTSecretRequired
	}
	return []byte(secret), nil
}

func GenerateUserJWT(userID uuid.UUID, publicID int64) (string, error) {
	jwtSecret, err := userJWTSecretFromEnvironment()
	if err != nil {
		return "", err
	}
	if userID == uuid.Nil || publicID <= 0 {
		return "", ErrInvalidUserJWTClaims
	}
	now := time.Now().UTC()

	claims := &jwtclaims.UserJWTClaims{
		UserID:   userID,
		PublicID: publicID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: now.AddDate(1, 0, 30).Unix(),
			IssuedAt:  now.Unix(),
			NotBefore: now.Add(-userJWTClockSkew).Unix(),
			Issuer:    constants.APPLICATION_NAME,
			Subject:   userJWTSubject,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}
	return "Bearer " + tokenString, nil
}

func IsValidJWTFormat(token string) bool {
	return userJWTFormat.MatchString(token)
}

func DecodeUserJWT(tokenString string) (*jwtclaims.UserJWTClaims, error) {
	jwtSecret, err := userJWTSecretFromEnvironment()
	if err != nil {
		return nil, err
	}
	if len(tokenString) > 1024 {
		return nil, errors.New("token too long")
	}

	if !IsValidJWTFormat(tokenString) {
		return nil, errors.New("invalid JWT format")
	}

	if strings.Count(tokenString, ".") != 2 {
		return nil, errors.New("invalid token format")
	}

	parser := &jwt.Parser{
		ValidMethods:         []string{jwt.SigningMethodHS256.Alg()},
		SkipClaimsValidation: true,
	}
	token, err := parser.ParseWithClaims(tokenString, &jwtclaims.UserJWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid jwt token")
	}
	claims, ok := token.Claims.(*jwtclaims.UserJWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token or claims")
	}

	now := time.Now().UTC().Unix()
	clockSkewSeconds := int64(userJWTClockSkew / time.Second)
	if claims.UserID == uuid.Nil ||
		claims.PublicID <= 0 ||
		claims.Issuer != constants.APPLICATION_NAME ||
		claims.Subject != userJWTSubject ||
		claims.IssuedAt <= 0 ||
		claims.NotBefore <= 0 ||
		claims.ExpiresAt <= 0 ||
		claims.IssuedAt > now+clockSkewSeconds ||
		claims.NotBefore > now+clockSkewSeconds ||
		claims.ExpiresAt <= now-clockSkewSeconds ||
		claims.ExpiresAt <= claims.IssuedAt ||
		claims.NotBefore > claims.ExpiresAt {
		return nil, ErrInvalidUserJWTClaims
	}
	return claims, nil
}
