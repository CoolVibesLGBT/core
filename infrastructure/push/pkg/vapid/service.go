package vapid

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const headerTemplate = `vapid t=%s, k=%s`

var errSubjectNotValid = errors.New("subject VAPID should be either a \"mailto:\" (email) or a \"https:\" URI")

type Service struct {
	publicKey  string
	privateKey *ecdsa.PrivateKey
	subject    string
}

func NewService(publicKey, privateKey, subject string) (*Service, error) {
	if !regexp.MustCompile(`^(https:|mailto:)`).MatchString(subject) {
		return nil, errSubjectNotValid
	}

	if err := checkPublicKey(publicKey); err != nil {
		return nil, fmt.Errorf("failed to validate public key: %w", err)
	}

	privateECDSA, err := preparePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare private key: %w", err)
	}

	jwt.MarshalSingleStringAsArray = false

	return &Service{
		publicKey:  publicKey,
		privateKey: privateECDSA,
		subject:    subject,
	}, nil
}

func (s *Service) Header(endpoint string) (string, error) {
	uri, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to parse endpoint: %w", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.RegisteredClaims{
		Subject:   s.subject,
		Audience:  jwt.ClaimStrings{fmt.Sprintf("%s://%s", uri.Scheme, uri.Host)},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 12)),
	})

	tokenSigned, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return fmt.Sprintf(headerTemplate, tokenSigned, s.publicKey), nil
}
