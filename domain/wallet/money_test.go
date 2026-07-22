package wallet

import (
	"errors"
	"math"
	"testing"

	"github.com/shopspring/decimal"
)

func TestValidateTipAmountRejectsUnsafeExponentWithoutRescaling(t *testing.T) {
	unsafe := decimal.New(1, math.MaxInt32)
	if err := ValidateTipAmount(unsafe); !errors.Is(err, ErrAmountOutOfRange) {
		t.Fatalf("ValidateTipAmount(huge exponent) error = %v", err)
	}
}

func TestValidateTipAmountMatchesNumericPrecision(t *testing.T) {
	tests := []struct {
		value   string
		wantErr error
	}{
		{value: "0.01"},
		{value: "99999999999999999999.999999999999999999"},
		{value: "100000000000000000000", wantErr: ErrAmountOutOfRange},
		{value: "0.0000000000000000001", wantErr: ErrAmountOutOfRange},
		{value: "0", wantErr: ErrInvalidAmount},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			amount, err := decimal.NewFromString(tt.value)
			if err != nil {
				t.Fatalf("parse fixture: %v", err)
			}
			err = ValidateTipAmount(amount)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateTipAmount(%s) error = %v, want %v", tt.value, err, tt.wantErr)
			}
		})
	}
}
