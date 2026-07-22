package wallet

import (
	"errors"

	"github.com/shopspring/decimal"
)

const (
	MoneyPrecision   = 38
	MoneyScale       = 18
	MoneyIntegerSize = MoneyPrecision - MoneyScale
)

var ErrAmountOutOfRange = errors.New("amount exceeds supported precision or scale")

func MinimumTipAmount() decimal.Decimal {
	return decimal.New(1, -2)
}

// ValidateMoneyRepresentation checks PostgreSQL NUMERIC(38,18) compatibility
// without rescaling. Checking the exponent first is important: decimal.Cmp on
// a hostile exponent can otherwise attempt an enormous power-of-ten allocation.
func ValidateMoneyRepresentation(value decimal.Decimal) error {
	exponent := int(value.Exponent())
	if exponent < -MoneyScale || exponent >= MoneyIntegerSize {
		return ErrAmountOutOfRange
	}

	coefficient := value.Coefficient()
	digits := len(coefficient.Abs(coefficient).String())
	if digits > MoneyPrecision {
		return ErrAmountOutOfRange
	}

	integerDigits := digits + exponent
	if integerDigits < 0 {
		integerDigits = 0
	}
	if integerDigits > MoneyIntegerSize {
		return ErrAmountOutOfRange
	}
	return nil
}

func ValidateTipAmount(amount decimal.Decimal) error {
	if err := ValidateMoneyRepresentation(amount); err != nil {
		return err
	}
	if amount.Coefficient().Sign() <= 0 {
		return ErrInvalidAmount
	}
	if amount.Cmp(MinimumTipAmount()) < 0 {
		return ErrAmountBelowMinimum
	}
	return nil
}
