package ports

import (
	"context"
	domainevents "core/domain/events"
	domainuser "core/domain/user"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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

// SessionUser is the narrow read model required by authenticated request hot
// paths. Sensitive credentials and large profile collections intentionally do
// not cross the session repository boundary.
type SessionUser struct {
	ID               uuid.UUID
	PublicID         int64
	Domain           domainuser.DomainKind
	UserName         string
	DisplayName      string
	DefaultLanguage  string
	PreferencesFlags string
	Role             string
	IsBot            bool
	Balance          decimal.Decimal
	HasLocation      bool
}

type SessionRepository interface {
	GetSessionUserByPublicID(ctx context.Context, publicID int64) (*SessionUser, error)
	UpdateLocation(ctx context.Context, userID uuid.UUID, ip string) error
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
