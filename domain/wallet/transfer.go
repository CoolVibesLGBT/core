package wallet

import (
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrInvalidParty       = errors.New("transfer parties are required")
	ErrSelfTransfer       = errors.New("cannot transfer to self")
	ErrInvalidAmount      = errors.New("transfer amount must be positive")
	ErrAmountBelowMinimum = errors.New("transfer amount is below minimum")
	ErrInsufficientFunds  = errors.New("insufficient funds")
)

// Transfer is the validated value object used by money-moving use cases. It
// deliberately contains no persistence or transport concerns.
type Transfer struct {
	fromID uuid.UUID
	toID   uuid.UUID
	amount decimal.Decimal
}

func NewTransfer(
	fromID uuid.UUID,
	toID uuid.UUID,
	amount decimal.Decimal,
	minimum decimal.Decimal,
	available decimal.Decimal,
) (Transfer, error) {
	if fromID == uuid.Nil || toID == uuid.Nil {
		return Transfer{}, ErrInvalidParty
	}
	if fromID == toID {
		return Transfer{}, ErrSelfTransfer
	}
	if err := ValidateMoneyRepresentation(amount); err != nil {
		return Transfer{}, err
	}
	if amount.Coefficient().Sign() <= 0 {
		return Transfer{}, ErrInvalidAmount
	}
	if err := ValidateMoneyRepresentation(minimum); err != nil {
		return Transfer{}, err
	}
	if minimum.Coefficient().Sign() > 0 && amount.Cmp(minimum) < 0 {
		return Transfer{}, ErrAmountBelowMinimum
	}
	if err := ValidateMoneyRepresentation(available); err != nil {
		return Transfer{}, err
	}
	if available.Cmp(amount) < 0 {
		return Transfer{}, ErrInsufficientFunds
	}

	return Transfer{fromID: fromID, toID: toID, amount: amount}, nil
}

func (t Transfer) FromID() uuid.UUID       { return t.fromID }
func (t Transfer) ToID() uuid.UUID         { return t.toID }
func (t Transfer) Amount() decimal.Decimal { return t.amount }

func (t Transfer) Apply(fromBalance, toBalance decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	return fromBalance.Sub(t.amount), toBalance.Add(t.amount)
}
