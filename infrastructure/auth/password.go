package auth

import "core/helpers"

type PasswordHasher struct{}

func (PasswordHasher) HashPassword(raw string) (string, error) {
	return helpers.HashPasswordArgon2id(raw)
}

func (PasswordHasher) ComparePassword(hashed string, raw string) (bool, error) {
	return helpers.ComparePasswordArgon2id(hashed, raw)
}
