package repositories

import (
	"core/application/ports"
	"core/models"
	"core/models/payment"
	"errors"
	"testing"
)

func TestPaymentRepositoryNeverReportsUnimplementedProcessingAsSuccess(t *testing.T) {
	repository := &PaymentRepository{}
	if err := repository.ProcessPayment(payment.PaymentKind_GOOGLEPAY, models.User{}); !errors.Is(err, ports.ErrPaymentProcessingNotImplemented) {
		t.Fatalf("ProcessPayment() error = %v, want not implemented", err)
	}
}
