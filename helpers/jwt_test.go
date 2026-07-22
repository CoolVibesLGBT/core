package helpers

import (
	"errors"
	"strings"
	"testing"
	"time"

	"core/constants"
	"core/models/jwtclaims"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

const testUserJWTSecret = "test-only-user-jwt-secret-with-more-than-32-bytes"

func TestValidateUserJWTConfigurationFailsClosed(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantErr error
	}{
		{name: "missing", secret: "", wantErr: ErrUserJWTSecretRequired},
		{name: "whitespace", secret: "   ", wantErr: ErrUserJWTSecretRequired},
		{name: "too short", secret: "short-secret", wantErr: ErrUserJWTSecretTooShort},
		{name: "valid", secret: testUserJWTSecret},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(userJWTSecretEnvironmentKey, tt.secret)
			err := ValidateUserJWTConfiguration()
			if tt.wantErr == nil && err != nil {
				t.Fatalf("ValidateUserJWTConfiguration() error = %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateUserJWTConfiguration() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateAndDecodeUserJWTValidatesIdentityClaims(t *testing.T) {
	t.Setenv(userJWTSecretEnvironmentKey, testUserJWTSecret)
	userID := uuid.New()

	tokenWithScheme, err := GenerateUserJWT(userID, 42)
	if err != nil {
		t.Fatalf("GenerateUserJWT() error = %v", err)
	}
	tokenString, found := strings.CutPrefix(tokenWithScheme, "Bearer ")
	if !found {
		t.Fatalf("GenerateUserJWT() = %q, want Bearer scheme", tokenWithScheme)
	}

	claims, err := DecodeUserJWT(tokenString)
	if err != nil {
		t.Fatalf("DecodeUserJWT() error = %v", err)
	}
	if claims.UserID != userID || claims.PublicID != 42 {
		t.Fatalf("DecodeUserJWT() identity = %s/%d, want %s/42", claims.UserID, claims.PublicID, userID)
	}
	if claims.Issuer != constants.APPLICATION_NAME || claims.Subject != userJWTSubject {
		t.Fatalf("DecodeUserJWT() issuer/subject = %q/%q", claims.Issuer, claims.Subject)
	}
	if claims.IssuedAt == 0 || claims.NotBefore == 0 || claims.ExpiresAt == 0 {
		t.Fatalf("DecodeUserJWT() missing temporal claims: %#v", claims.StandardClaims)
	}
	if claims.IssuedAt-claims.NotBefore < int64(userJWTClockSkew/time.Second) {
		t.Fatalf("generated token nbf is not backdated for clock skew: %#v", claims.StandardClaims)
	}
}

func TestGenerateUserJWTRejectsMissingIdentity(t *testing.T) {
	t.Setenv(userJWTSecretEnvironmentKey, testUserJWTSecret)

	if _, err := GenerateUserJWT(uuid.Nil, 42); !errors.Is(err, ErrInvalidUserJWTClaims) {
		t.Fatalf("GenerateUserJWT(nil user) error = %v, want ErrInvalidUserJWTClaims", err)
	}
	if _, err := GenerateUserJWT(uuid.New(), 0); !errors.Is(err, ErrInvalidUserJWTClaims) {
		t.Fatalf("GenerateUserJWT(zero public ID) error = %v, want ErrInvalidUserJWTClaims", err)
	}
}

func TestDecodeUserJWTRejectsNonHS256Algorithm(t *testing.T) {
	t.Setenv(userJWTSecretEnvironmentKey, testUserJWTSecret)
	claims := validTestUserJWTClaims()
	tokenString := signTestUserJWT(t, jwt.SigningMethodHS384, claims)

	if _, err := DecodeUserJWT(tokenString); err == nil || !strings.Contains(strings.ToLower(err.Error()), "signing method") {
		t.Fatalf("DecodeUserJWT(HS384) error = %v, want signing method rejection", err)
	}
}

func TestDecodeUserJWTRejectsInvalidRequiredClaims(t *testing.T) {
	t.Setenv(userJWTSecretEnvironmentKey, testUserJWTSecret)

	tests := []struct {
		name   string
		mutate func(*jwtclaims.UserJWTClaims)
	}{
		{name: "missing user ID", mutate: func(c *jwtclaims.UserJWTClaims) { c.UserID = uuid.Nil }},
		{name: "missing public ID", mutate: func(c *jwtclaims.UserJWTClaims) { c.PublicID = 0 }},
		{name: "wrong issuer", mutate: func(c *jwtclaims.UserJWTClaims) { c.Issuer = "another-app" }},
		{name: "wrong subject", mutate: func(c *jwtclaims.UserJWTClaims) { c.Subject = "OTHER" }},
		{name: "missing issued at", mutate: func(c *jwtclaims.UserJWTClaims) { c.IssuedAt = 0 }},
		{name: "missing not before", mutate: func(c *jwtclaims.UserJWTClaims) { c.NotBefore = 0 }},
		{name: "missing expiry", mutate: func(c *jwtclaims.UserJWTClaims) { c.ExpiresAt = 0 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := validTestUserJWTClaims()
			tt.mutate(claims)
			tokenString := signTestUserJWT(t, jwt.SigningMethodHS256, claims)

			if _, err := DecodeUserJWT(tokenString); !errors.Is(err, ErrInvalidUserJWTClaims) {
				t.Fatalf("DecodeUserJWT() error = %v, want ErrInvalidUserJWTClaims", err)
			}
		})
	}
}

func TestDecodeUserJWTToleratesOnlyBoundedClockSkew(t *testing.T) {
	t.Setenv(userJWTSecretEnvironmentKey, testUserJWTSecret)

	withinSkew := validTestUserJWTClaims()
	withinSkew.IssuedAt = time.Now().UTC().Add(userJWTClockSkew / 2).Unix()
	withinSkew.NotBefore = withinSkew.IssuedAt
	if _, err := DecodeUserJWT(signTestUserJWT(t, jwt.SigningMethodHS256, withinSkew)); err != nil {
		t.Fatalf("DecodeUserJWT(within clock skew) error = %v", err)
	}

	beyondSkew := validTestUserJWTClaims()
	beyondSkew.IssuedAt = time.Now().UTC().Add(userJWTClockSkew + 5*time.Second).Unix()
	beyondSkew.NotBefore = beyondSkew.IssuedAt
	if _, err := DecodeUserJWT(signTestUserJWT(t, jwt.SigningMethodHS256, beyondSkew)); !errors.Is(err, ErrInvalidUserJWTClaims) {
		t.Fatalf("DecodeUserJWT(beyond clock skew) error = %v; want ErrInvalidUserJWTClaims", err)
	}
}

func TestTokenHelpersRejectMissingSecret(t *testing.T) {
	t.Setenv(userJWTSecretEnvironmentKey, "")

	if _, err := GenerateUserJWT(uuid.New(), 42); !errors.Is(err, ErrUserJWTSecretRequired) {
		t.Fatalf("GenerateUserJWT() error = %v, want ErrUserJWTSecretRequired", err)
	}
	if _, err := DecodeUserJWT("header.payload.signature"); !errors.Is(err, ErrUserJWTSecretRequired) {
		t.Fatalf("DecodeUserJWT() error = %v, want ErrUserJWTSecretRequired", err)
	}
}

func validTestUserJWTClaims() *jwtclaims.UserJWTClaims {
	now := time.Now().UTC()
	return &jwtclaims.UserJWTClaims{
		UserID:   uuid.New(),
		PublicID: 42,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: now.Add(time.Hour).Unix(),
			IssuedAt:  now.Add(-time.Minute).Unix(),
			NotBefore: now.Add(-time.Minute).Unix(),
			Issuer:    constants.APPLICATION_NAME,
			Subject:   userJWTSubject,
		},
	}
}

func signTestUserJWT(t *testing.T, method jwt.SigningMethod, claims *jwtclaims.UserJWTClaims) string {
	t.Helper()
	tokenString, err := jwt.NewWithClaims(method, claims).SignedString([]byte(testUserJWTSecret))
	if err != nil {
		t.Fatalf("sign test JWT: %v", err)
	}
	return tokenString
}
