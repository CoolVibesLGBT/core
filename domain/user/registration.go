package user

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidEmail          = errors.New("invalid email")
	ErrDisplayNameRequired   = errors.New("display name is required")
	ErrUsernameRequired      = errors.New("username is required")
	ErrUsernameAlreadyExists = errors.New("username already exists")
	ErrEmailAlreadyExists    = errors.New("email already exists")
	emailPattern             = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

type RegistrationInput struct {
	Name     string // Profile display name.
	Nickname string // Unique username used for sign-in.
	Password string
	Domain   string
	Email    string
}

type Registration struct {
	Name     string // Profile display name.
	Nickname string // Unique, normalized username used for sign-in.
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
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Registration{}, ErrDisplayNameRequired
	}
	nickname := strings.ToLower(strings.TrimSpace(input.Nickname))
	if nickname == "" {
		return Registration{}, ErrUsernameRequired
	}

	return Registration{
		Name:     name,
		Nickname: nickname,
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
