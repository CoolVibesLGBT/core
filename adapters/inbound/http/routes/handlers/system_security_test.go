package handlers

import (
	"context"
	"core/application/ports"
	usecases "core/application/usecases"
	"core/models"
	"core/models/payment"
	postpayloads "core/models/post/payloads"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type systemSecurityRepository struct {
	method *payment.PaymentMethod
}

func (r *systemSecurityRepository) GetPreferences(context.Context) (models.PreferencesData, error) {
	return models.PreferencesData{}, nil
}

func (r *systemSecurityRepository) GetEventKinds(context.Context) ([]postpayloads.EventKind, error) {
	return nil, nil
}

func (r *systemSecurityRepository) GetReportKinds(context.Context) ([]models.ReportKind, error) {
	return nil, nil
}

func (r *systemSecurityRepository) GetVapidPublicKey(context.Context) (string, error) {
	return "", nil
}

func (r *systemSecurityRepository) SaveVapidSubscription(context.Context, uuid.UUID, models.Subscription) error {
	return nil
}

func (r *systemSecurityRepository) GetPaymentMethod(context.Context) (*payment.PaymentMethod, error) {
	return r.method, nil
}

var _ ports.SystemRepository = (*systemSecurityRepository)(nil)

func TestPaymentMethodsResponseNeverSerializesGatewaySecrets(t *testing.T) {
	repository := &systemSecurityRepository{method: &payment.PaymentMethod{
		ID:                 uuid.New(),
		DefaultPaymentKind: payment.PaymentKind_CRYPTO,
		CryptoDetails:      datatypes.JSON(`{"network":"test"}`),
		Packages:           datatypes.JSON(`[{"id":"starter"}]`),
		SecretKeys:         datatypes.JSON(`{"private_key":"server-private-key"}`),
	}}
	app := fiber.New()
	app.Get("/", HandleFetchPaymentMethods(usecases.NewSystemService(repository)))

	response, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatalf("payment methods request: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read payment methods response: %v", err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("payment methods status = %d, body=%s", response.StatusCode, body)
	}
	serialized := string(body)
	for _, forbidden := range []string{"server-private-key", `"secrets"`, strings.ToLower(repository.method.ID.String())} {
		if strings.Contains(strings.ToLower(serialized), strings.ToLower(forbidden)) {
			t.Fatalf("payment methods response leaked %q: %s", forbidden, serialized)
		}
	}
	if !strings.Contains(serialized, `"network":"test"`) || !strings.Contains(serialized, `"starter"`) {
		t.Fatalf("payment methods response lost public configuration: %s", serialized)
	}
}
