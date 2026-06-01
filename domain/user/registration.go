package user

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidEmail = errors.New("invalid email")
	emailPattern    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

type RegistrationInput struct {
	Name     string
	Nickname string
	Password string
	Domain   string
	Email    string
}

type Registration struct {
	Name     string
	Nickname string
	Password string
	Domain   DomainKind
	Email    string
}

type Credentials struct {
	UserName string
	Password string
}

func NewRegistration(input RegistrationInput) (Registration, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email != "" && !IsValidEmail(email) {
		return Registration{}, ErrInvalidEmail
	}

	domain, err := ParseDomainKind(input.Domain)
	if err != nil {
		return Registration{}, err
	}

	return Registration{
		Name:     strings.TrimSpace(input.Name),
		Nickname: strings.ToLower(strings.TrimSpace(input.Nickname)),
		Password: input.Password,
		Domain:   domain,
		Email:    email,
	}, nil
}

func NewCredentials(userName, password string) Credentials {
	return Credentials{
		UserName: strings.ToLower(strings.TrimSpace(userName)),
		Password: password,
	}
}

func IsValidEmail(email string) bool {
	return emailPattern.MatchString(email)
}
