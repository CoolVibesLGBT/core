package push

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

type Urgency string

const (
	UrgencyVeryLow Urgency = "very-low" // Device State - On power and Wi-Fi
	UrgencyLow     Urgency = "low"      // Device State - On either power or Wi-Fi
	UrgencyNormal  Urgency = "normal"   // Device State - On neither power nor Wi-Fi
	UrgencyHigh    Urgency = "high"     // Device State - Low battery
)

type Push struct {
	Endpoint  string
	Auth      string
	P256DH    string
	Plaintext []byte
	Urgency   Urgency
	TTL       time.Duration
}

func GenerateVAPIDKeys() (string, string, error) {

	privateKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %v", err)
	}
	publicKey := privateKey.PublicKey().Bytes()

	publicB64 := base64.RawURLEncoding.EncodeToString(publicKey)

	privateB64 := base64.RawURLEncoding.EncodeToString(privateKey.Bytes())

	return privateB64, publicB64, nil
}
