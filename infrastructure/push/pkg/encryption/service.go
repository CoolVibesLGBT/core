package encryption

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

const maxPlaintextLen = 3993

type Service struct {
	publicKey  []byte
	privateKey *ecdh.PrivateKey

	mu *sync.RWMutex
}

func NewService() (*Service, error) {
	privateKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	publicKey := privateKey.PublicKey().Bytes()

	return &Service{
		publicKey:  publicKey,
		privateKey: privateKey,
		mu:         new(sync.RWMutex),
	}, nil
}

// Encrypt return *bytes.Buffer with ciphertext.
func (s *Service) Encrypt(auth, p256dh string, plaintext []byte) (*bytes.Buffer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(plaintext) > maxPlaintextLen {
		return nil, fmt.Errorf("plaintext too long (%d > %d)", len(plaintext), maxPlaintextLen)
	}

	authSecret, uaPublicKey, err := s.prepareInputData(auth, p256dh)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare input data: %w", err)
	}

	sharedSecret, err := s.ecdhExchange(uaPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange shared secret: %w", err)
	}

	ikm, err := s.prepareIKM(sharedSecret, authSecret, uaPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare ikm: %w", err)
	}

	salt, err := s.prepareSalt()
	if err != nil {
		return nil, fmt.Errorf("failed to prepare salt: %w", err)
	}

	nonce, gcm, err := s.prepareNonceAndGCM(salt, ikm)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare nonce and GCM: %w", err)
	}

	recordSize := len(plaintext) + 86 + 16 + 1
	buf := bytes.NewBuffer(make([]byte, 0, recordSize))
	s.messageHeader(buf, salt, uint32(recordSize))
	s.messageBody(buf, gcm, nonce, plaintext)

	return buf, nil
}

// Rotate enables key rotation according to interval.
func (s *Service) Rotate(interval time.Duration) {
	go func(s *Service) {
		ticker := time.NewTicker(interval)

		for range ticker.C {
			s.mu.Lock()
			privateKey, _ := ecdh.P256().GenerateKey(rand.Reader)
			s.privateKey = privateKey
			s.publicKey = privateKey.PublicKey().Bytes()
			s.mu.Unlock()
		}
	}(s)
}
