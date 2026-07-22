package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"core/application/ports"
	usecases "core/application/usecases"
	domainwallet "core/domain/wallet"
	"core/models"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type tipHandlerPostRepository struct {
	ports.PostRepository
	postID         int64
	amount         decimal.Decimal
	idempotencyKey domainwallet.IdempotencyKey
	err            error
}

func (r *tipHandlerPostRepository) Tip(_ context.Context, postID int64, _ *models.User, amount decimal.Decimal, key domainwallet.IdempotencyKey) (*decimal.Decimal, error) {
	r.postID = postID
	r.amount = amount
	r.idempotencyKey = key
	if r.err != nil {
		return nil, r.err
	}
	balance := decimal.NewFromInt(8)
	return &balance, nil
}

func TestParseTipAmountRejectsScientificNotationAndUnsafeScale(t *testing.T) {
	for _, value := range []string{
		"1e2147483647",
		"1e2",
		"0.0000000000000000001",
		"100000000000000000000",
		"-1",
	} {
		t.Run(value, func(t *testing.T) {
			if _, err := parseTipAmount(value); err == nil {
				t.Fatalf("parseTipAmount(%q) error = nil", value)
			}
		})
	}
}

func TestParseTipAmountAcceptsNumeric38Scale18Boundary(t *testing.T) {
	amount, err := parseTipAmount("99999999999999999999.999999999999999999")
	if err != nil {
		t.Fatalf("parseTipAmount() error = %v", err)
	}
	if amount.String() != "99999999999999999999.999999999999999999" {
		t.Fatalf("parseTipAmount() = %s", amount)
	}
	if _, err := parseTipAmount("0.001"); !errors.Is(err, domainwallet.ErrAmountBelowMinimum) {
		t.Fatalf("below-minimum error = %v", err)
	}
}

func TestHandlePostTipRequiresAndDelegatesIdempotencyKey(t *testing.T) {
	repository := &tipHandlerPostRepository{}
	service := usecases.NewPostService(&handlerUserRepo{}, repository, &handlerMediaRepo{})
	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		c.Locals("authenticatedUser", &models.User{ID: uuid.New(), PublicID: 10})
		return HandlePostTip(service)(c)
	})

	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("post_id=123&amount=2.25"))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationForm)
	request.Header.Set("Idempotency-Key", "tip-handler-request")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("tip response status = %d, want 200", response.StatusCode)
	}
	if repository.postID != 123 || !repository.amount.Equal(decimal.RequireFromString("2.25")) || repository.idempotencyKey.String() != "tip-handler-request" {
		t.Fatalf("tip delegate = post:%d amount:%s key:%q", repository.postID, repository.amount, repository.idempotencyKey)
	}

	missingKey := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("post_id=123&amount=2.25"))
	missingKey.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationForm)
	response, err = app.Test(missingKey)
	if err != nil {
		t.Fatalf("missing-key app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("missing-key status = %d, want 400", response.StatusCode)
	}
}

func TestHandlePostTipMapsIdempotencyConflictToHTTP409(t *testing.T) {
	repository := &tipHandlerPostRepository{err: domainwallet.ErrIdempotencyConflict}
	service := usecases.NewPostService(&handlerUserRepo{}, repository, &handlerMediaRepo{})
	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		c.Locals("authenticatedUser", &models.User{ID: uuid.New(), PublicID: 10})
		return HandlePostTip(service)(c)
	})

	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("post_id=123&amount=2.25"))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationForm)
	request.Header.Set("Idempotency-Key", "tip-conflict-request")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusConflict {
		t.Fatalf("conflict status = %d, want 409", response.StatusCode)
	}
}
