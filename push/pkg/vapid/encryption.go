package vapid

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/x509"
	"fmt"

	"core/push/pkg/utils"
)

func checkPublicKey(key string) error {
	publicKey, err := utils.ParseBase64Key(key)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	_, err = ecdh.P256().NewPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to create public key: %w", err)
	}

	return nil
}

func preparePrivateKey(key string) (*ecdsa.PrivateKey, error) {
	privateKeyDecoded, err := utils.ParseBase64Key(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	privateECDH, err := ecdh.P256().NewPrivateKey(privateKeyDecoded)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key: %w", err)
	}

	privatePKCS8, err := x509.MarshalPKCS8PrivateKey(privateECDH)
	if err != nil {
		return nil, fmt.Errorf("failed to convert private key: %w", err)
	}

	privateECDSA, err := x509.ParsePKCS8PrivateKey(privatePKCS8)
	if err != nil {
		return nil, fmt.Errorf("failed to convert private key: %w", err)
	}

	return privateECDSA.(*ecdsa.PrivateKey), nil
}
