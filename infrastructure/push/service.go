// Package pushbell provides functions for sending web push notifications with
// support for the VAPID (Voluntary Application Server Identification)
// specification.

package push

import (
	"fmt"

	"core/infrastructure/push/pkg/encryption"
	"core/infrastructure/push/pkg/httpclient"
	"core/infrastructure/push/pkg/vapid"
)

// Service contains all dependencies needed for sending web push notifications.
type Service struct {
	Encryption               *encryption.Service
	Vapid                    *vapid.Service
	Client                   httpclient.Client
	StatusCodeValidationFunc StatusCodeValidationFunc
}

// NewService creates new service with given application server keys and subject.
func NewService(options *Options) (*Service, error) {
	vapidService, err := vapid.NewService(
		options.ApplicationServerPublicKey,
		options.ApplicationServerPrivateKey,
		options.ApplicationServerSubject,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create vapid service: %w", err)
	}

	encryptionService, err := encryption.NewService()
	if err != nil {
		return nil, fmt.Errorf("failed to create encryption service: %w", err)
	}

	if options.KeyRotationInterval != 0 {
		encryptionService.Rotate(options.KeyRotationInterval)
	}

	client := options.HttpClient
	if client == nil {
		client = httpclient.FastHttp(nil)
	}

	return &Service{
		Encryption:               encryptionService,
		Vapid:                    vapidService,
		Client:                   client,
		StatusCodeValidationFunc: options.StatusCodeValidationFunc,
	}, nil
}

// Send sends a WebPush notification with parameters to the specified endpoint.
func (s *Service) Send(push *Push) error {
	// Cipher text.
	body, err := s.Encryption.Encrypt(push.Auth, push.P256DH, push.Plaintext)
	if err != nil {
		return fmt.Errorf("failed to encrypt push body: %w", err)
	}

	// Get auth header.
	authHeader, err := s.Vapid.Header(push.Endpoint)
	if err != nil {
		return fmt.Errorf("failed to generate vapid auth header: %w", err)
	}

	// Prepare headers for client.
	headers := &httpclient.Headers{
		Authorization: authHeader,
		Urgency:       string(push.Urgency),
		TTL:           push.TTL,
	}

	// Request delivery.
	statusCode, err := s.Client.RequestDelivery(push.Endpoint, headers, body)
	if err != nil {
		return fmt.Errorf("failed to send push request: %w", err)
	}

	// Check status code if enabled.
	if s.StatusCodeValidationFunc != nil {
		return s.StatusCodeValidationFunc(statusCode)
	}

	return nil
}
