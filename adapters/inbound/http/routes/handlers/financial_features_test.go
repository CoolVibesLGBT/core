package handlers

import (
	"bytes"
	"context"
	"core/application/ports"
	"core/application/types"
	"core/application/usecases"
	"core/constants"
	domainuser "core/domain/user"
	domainwallet "core/domain/wallet"
	"core/models"
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type financialGiftFake struct {
	ports.BroadcastGiftRepository
	actor, id uuid.UUID
	amount    decimal.Decimal
	key       string
	calls     int
	err       error
}

func (r *financialGiftFake) SendBroadcastGift(_ context.Context, actor, id uuid.UUID, amount decimal.Decimal, key domainwallet.IdempotencyKey) (ports.BroadcastGiftReceipt, error) {
	r.calls++
	r.actor, r.id, r.amount, r.key = actor, id, amount, key.String()
	return ports.BroadcastGiftReceipt{ID: uuid.New(), BroadcastID: id, RecipientPublicID: types.SnowflakeID(9007199254740993), Balance: decimal.NewFromInt(8)}, r.err
}
func (r *financialGiftFake) StartBroadcastSession(_ context.Context, actor uuid.UUID) (ports.BroadcastSessionView, error) {
	r.calls++
	r.actor = actor
	return ports.BroadcastSessionView{ID: uuid.New(), OwnerPublicID: 10}, r.err
}

type financialReferralFake struct {
	ports.ReferralRewardRepository
	actor, id uuid.UUID
	review    domainuser.ReferralReview
	calls     int
}

func (r *financialReferralFake) ReviewReferralReward(_ context.Context, actor, id uuid.UUID, review domainuser.ReferralReview) (ports.ReferralRewardView, error) {
	r.calls++
	r.actor, r.id, r.review = actor, id, review
	return ports.ReferralRewardView{ID: id, Status: review.Status()}, nil
}
func giftHTTPApp(repo *financialGiftFake, actor *models.User) *fiber.App {
	app := fiber.New()
	service := usecases.NewBroadcastGiftService(repo)
	app.Post("/", func(c fiber.Ctx) error {
		if actor != nil {
			c.Locals("authenticatedUser", actor)
		}
		return HandleBroadcastGift(service)(c)
	})
	return app
}
func TestGiftHTTPUsesSessionActorAndPreservesPublicIDPrecision(t *testing.T) {
	actor := &models.User{ID: uuid.New(), PublicID: 10}
	id := uuid.New()
	repo := &financialGiftFake{}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"broadcast_id":"`+id.String()+`","amount":"2.25","payer_id":"spoofed","recipient_id":"999","idempotency_key":"gift-json-key"}`))
	request.Header.Set("Content-Type", "application/json")
	response, err := giftHTTPApp(repo, actor).Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close response body: %v", err)
		}
	}()
	if response.StatusCode != http.StatusOK || repo.actor != actor.ID || repo.id != id || repo.key != "gift-json-key" || !repo.amount.Equal(decimal.RequireFromString("2.25")) {
		t.Fatalf("gift status %d, capture %+v", response.StatusCode, repo)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte(`"recipient_id":"9007199254740993"`)) {
		t.Fatalf("public ID lost precision: %s", body)
	}
}
func TestGiftHTTPRejectsInvalidInputsBeforeRepository(t *testing.T) {
	for _, body := range []string{
		`{"broadcast_id":"not-a-uuid","amount":"2.25","idempotency_key":"valid-key-1"}`,
		`{"broadcast_id":"` + uuid.NewString() + `","amount":"1e100","idempotency_key":"valid-key-1"}`,
		`{"broadcast_id":"` + uuid.NewString() + `","amount":"0.001","idempotency_key":"valid-key-1"}`,
		`{"broadcast_id":"` + uuid.NewString() + `","amount":"2.25"}`,
	} {
		repo := &financialGiftFake{}
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response, err := giftHTTPApp(repo, &models.User{ID: uuid.New()}).Test(request)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusBadRequest || repo.calls != 0 {
			t.Fatalf("invalid gift %s: status %d, calls %d", body, response.StatusCode, repo.calls)
		}
	}
}
func TestGiftHTTPMapsFinancialErrorsWithoutLeakingDatabase(t *testing.T) {
	for _, test := range []struct {
		err    error
		status int
	}{
		{domainwallet.ErrTransferBlocked, 403}, {domainwallet.ErrTransferUnavailable, 403}, {ports.ErrBroadcastSessionUnavailable, 404}, {domainwallet.ErrInsufficientFunds, 400}, {domainwallet.ErrSelfTransfer, 400}, {domainwallet.ErrIdempotencyConflict, 409}, {errors.New("private SQL detail"), 500},
	} {
		repo := &financialGiftFake{err: test.err}
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"broadcast_id":"`+uuid.NewString()+`","amount":"2.25","idempotency_key":"gift-error-key"}`))
		request.Header.Set("Content-Type", "application/json")
		response, err := giftHTTPApp(repo, &models.User{ID: uuid.New()}).Test(request)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != test.status || bytes.Contains(body, []byte("private SQL detail")) {
			t.Fatalf("financial error %v: %d %s", test.err, response.StatusCode, body)
		}
	}
}
func TestReferralHTTPRequiresAdminAndUsesServerReviewer(t *testing.T) {
	for _, role := range []constants.UserRole{constants.UserRoleUser, constants.UserRoleModerator, constants.UserRoleAdmin} {
		actor := &models.User{ID: uuid.New(), UserRole: role}
		id := uuid.New()
		repo := &financialReferralFake{}
		service := usecases.NewReferralRewardService(repo)
		app := fiber.New()
		app.Post("/", func(c fiber.Ctx) error {
			c.Locals("authenticatedUser", actor)
			return HandleReferralRewardReview(service)(c)
		})
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"reward_id":"`+id.String()+`","decision":"approve","note":"Manually verified referral evidence","verification_reference":"provider:person-123","reviewed_by_id":"spoofed"}`))
		request.Header.Set("Content-Type", "application/json")
		response, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if role != constants.UserRoleAdmin {
			if response.StatusCode != http.StatusForbidden || repo.calls != 0 {
				t.Fatalf("role %s: status %d calls %d", role, response.StatusCode, repo.calls)
			}
		} else if response.StatusCode != http.StatusOK || repo.actor != actor.ID || repo.id != id || repo.review.IdentityHash() == nil {
			t.Fatalf("admin review status %d, capture %+v", response.StatusCode, repo)
		}
	}
}
func TestReferralHTTPRejectsApprovalWithoutVerificationReference(t *testing.T) {
	repo := &financialReferralFake{}
	service := usecases.NewReferralRewardService(repo)
	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		c.Locals("authenticatedUser", &models.User{ID: uuid.New(), UserRole: constants.UserRoleAdmin})
		return HandleReferralRewardReview(service)(c)
	})
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"reward_id":"`+uuid.NewString()+`","decision":"approve","note":"Manually reviewed the request"}`))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusBadRequest || repo.calls != 0 {
		t.Fatalf("missing evidence status %d, calls %d", response.StatusCode, repo.calls)
	}
}

type liveListFinancialFake struct {
	ports.UserRepository
	users []*models.User
}

func (r *liveListFinancialFake) FetchLiveUsers(types.Filter) ([]*models.User, error) {
	return r.users, nil
}
func TestLiveListDoesNotExposePublisherBalancesOrCredentials(t *testing.T) {
	human := &models.User{ID: uuid.New(), PublicID: 10, UserName: "human", Balance: decimal.NewFromInt(500), IsLive: true, UserRole: constants.UserRoleAdmin, BroadcastInfo: []byte(`{"token":"human-secret"}`)}
	external := &models.User{ID: uuid.New(), PublicID: 20, UserName: "external", Balance: decimal.NewFromInt(800), IsLive: true, IsBot: true, BroadcastInfo: []byte(`{"provider":"hornet","objectId":"external-stream","viewerToken":"public-viewer-playback","broadcasterToken":"publisher-secret","token":"provider-secret"}`)}
	repo := &liveListFinancialFake{users: []*models.User{human, external}}
	service := usecases.NewUserService(repo, nil, nil, nil, nil)
	app := fiber.New()
	app.Post("/", HandleFetchBroadcasts(service))
	response, err := app.Test(httptest.NewRequest(http.MethodPost, "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close response body: %v", err)
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("live list %d: %s", response.StatusCode, body)
	}
	for _, secret := range []string{`"balance"`, `"user_role"`, `"preferences_flags"`, human.ID.String(), external.ID.String(), "human-secret", "publisher-secret", "provider-secret"} {
		if bytes.Contains(body, []byte(secret)) {
			t.Fatalf("live list exposed %s", secret)
		}
	}
	if !bytes.Contains(body, []byte("public-viewer-playback")) || !bytes.Contains(body, []byte(`"can_receive_gifts":false`)) || !bytes.Contains(body, []byte(`"can_receive_gifts":true`)) {
		t.Fatalf("live compatibility fields missing: %s", body)
	}
}
