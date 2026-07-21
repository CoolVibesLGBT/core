package ports

import (
	"context"
	domainevents "core/domain/events"

	"github.com/google/uuid"
)

type CaptchaVerifier interface {
	VerifyCaptcha(ctx context.Context, response string) (bool, error)
}

type PasswordHasher interface {
	HashPassword(raw string) (string, error)
	ComparePassword(hashed string, raw string) (bool, error)
}

type TokenIssuer interface {
	GenerateUserToken(userID uuid.UUID, publicID int64) (string, error)
}

type UserTokenDecoder interface {
	DecodeUserPublicID(token string) (int64, error)
}

type PublicIDGenerator interface {
	GeneratePublicID() int64
}

type RemoteImageFetcher interface {
	FetchImage(ctx context.Context, imageURL string) (UploadedFile, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, events ...domainevents.Event) error
}

type EventPublisherFunc func(ctx context.Context, events ...domainevents.Event) error

func (f EventPublisherFunc) Publish(ctx context.Context, events ...domainevents.Event) error {
	return f(ctx, events...)
}

func NoopEventPublisher() EventPublisher {
	return EventPublisherFunc(func(context.Context, ...domainevents.Event) error {
		return nil
	})
}
