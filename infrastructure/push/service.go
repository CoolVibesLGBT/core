// Package pushbell provides functions for sending web push notifications with
// support for the VAPID (Voluntary Application Server Identification)
// specification.

package push

import (
	"context"
	"fmt"
	"time"

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
	DeliveryTimeout          time.Duration
}

// NewService creates new service with given application server keys and subject.
func NewService(options *Options) (*Service, error) {
	if options == nil {
		options = NewOptions()
	}
	deliveryTimeout := options.DeliveryTimeout
	if deliveryTimeout <= 0 {
		deliveryTimeout = DefaultDeliveryTimeout
	}

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
		DeliveryTimeout:          deliveryTimeout,
	}, nil
}

// Send sends a WebPush notification with parameters to the specified endpoint.
func (s *Service) Send(push *Push) error {
	return s.SendContext(context.Background(), push)
}

// SendContext sends a WebPush notification while honoring both the caller's
// deadline and the service's per-delivery timeout.
func (s *Service) SendContext(ctx context.Context, push *Push) error {
	if ctx == nil {
		ctx = context.Background()
	}
	deliveryTimeout := s.DeliveryTimeout
	if deliveryTimeout <= 0 {
		deliveryTimeout = DefaultDeliveryTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, deliveryTimeout)
	defer cancel()

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
	statusCode, err := s.Client.RequestDelivery(ctx, push.Endpoint, headers, body)
	if err != nil {
		return fmt.Errorf("failed to send push request: %w", err)
	}

	// Check status code if enabled.
	if s.StatusCodeValidationFunc != nil {
		return s.StatusCodeValidationFunc(statusCode)
	}

	return nil
}
