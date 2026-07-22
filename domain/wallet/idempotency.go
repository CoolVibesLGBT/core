package wallet

import (
	"errors"
	"strings"
	"unicode"
)

const (
	MinIdempotencyKeyLength = 8
	MaxIdempotencyKeyLength = 128
)

var (
	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")
	ErrInvalidIdempotencyKey  = errors.New("invalid idempotency key")
	ErrIdempotencyConflict    = errors.New("idempotency key was already used for a different operation")
)

type IdempotencyKey string

func NewIdempotencyKey(raw string) (IdempotencyKey, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", ErrIdempotencyKeyRequired
	}
	if len(value) < MinIdempotencyKeyLength || len(value) > MaxIdempotencyKeyLength {
		return "", ErrInvalidIdempotencyKey
	}
	for _, char := range value {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			continue
		}
		switch char {
		case '-', '_', '.', ':':
			continue
		default:
			return "", ErrInvalidIdempotencyKey
		}
	}
	return IdempotencyKey(value), nil
}

func (k IdempotencyKey) String() string {
	return string(k)
}
