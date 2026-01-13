package helpers

import (
	"crypto/rand"
	"encoding/base64"
)

// length ≈ çıktı uzunluğu (base64 olduğu için biraz uzar)
func GenerateRandomPassword(length int) (string, error) {
	b := make([]byte, length)

	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	// URL-safe, copy-paste friendly
	return base64.RawURLEncoding.EncodeToString(b), nil
}
