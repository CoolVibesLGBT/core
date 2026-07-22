package wallet

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestNewTransferValidatesMoneyInvariants(t *testing.T) {
	fromID := uuid.New()
	toID := uuid.New()
	minimum := decimal.RequireFromString("0.01")

	tests := []struct {
		name      string
		fromID    uuid.UUID
		toID      uuid.UUID
		amount    decimal.Decimal
		available decimal.Decimal
		wantErr   error
	}{
		{name: "missing party", fromID: uuid.Nil, toID: toID, amount: decimal.NewFromInt(1), available: decimal.NewFromInt(2), wantErr: ErrInvalidParty},
		{name: "self transfer", fromID: fromID, toID: fromID, amount: decimal.NewFromInt(1), available: decimal.NewFromInt(2), wantErr: ErrSelfTransfer},
		{name: "zero", fromID: fromID, toID: toID, amount: decimal.Zero, available: decimal.NewFromInt(2), wantErr: ErrInvalidAmount},
		{name: "below minimum", fromID: fromID, toID: toID, amount: decimal.RequireFromString("0.001"), available: decimal.NewFromInt(2), wantErr: ErrAmountBelowMinimum},
		{name: "insufficient", fromID: fromID, toID: toID, amount: decimal.NewFromInt(3), available: decimal.NewFromInt(2), wantErr: ErrInsufficientFunds},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTransfer(tt.fromID, tt.toID, tt.amount, minimum, tt.available)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewTransfer() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransferApplyPreservesTotalBalance(t *testing.T) {
	transfer, err := NewTransfer(
		uuid.New(),
		uuid.New(),
		decimal.RequireFromString("2.25"),
		decimal.RequireFromString("0.01"),
		decimal.NewFromInt(10),
	)
	if err != nil {
		t.Fatalf("NewTransfer() error = %v", err)
	}

	from, to := transfer.Apply(decimal.NewFromInt(10), decimal.NewFromInt(4))
	if !from.Equal(decimal.RequireFromString("7.75")) || !to.Equal(decimal.RequireFromString("6.25")) {
		t.Fatalf("Apply() balances = %s/%s", from, to)
	}
	if !from.Add(to).Equal(decimal.NewFromInt(14)) {
		t.Fatalf("Apply() changed total balance: %s", from.Add(to))
	}
}
