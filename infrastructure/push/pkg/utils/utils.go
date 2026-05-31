package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/hkdf"
)

func ParseBase64Key(keyString string) ([]byte, error) {
	padded := strings.Contains(keyString, "=")

	var enc *base64.Encoding

	if strings.ContainsAny(keyString, "-_") {
		if padded {
			enc = base64.URLEncoding
		} else {
			enc = base64.RawURLEncoding
		}
	} else {
		if padded {
			enc = base64.StdEncoding
		} else {
			enc = base64.RawStdEncoding
		}
	}

	data, err := enc.DecodeString(keyString)
	if err != nil {
		return nil, fmt.Errorf("failed to decode: %w", err)
	}

	return data, nil
}

func HkdfExtractAndExpand(length int, secret, salt, info []byte) ([]byte, error) {
	buf := make([]byte, length)

	reader := hkdf.New(sha256.New, secret, salt, info)

	if _, err := io.ReadFull(reader, buf); err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}

	return buf, nil
}
