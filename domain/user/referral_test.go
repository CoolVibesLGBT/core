package user

import (
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestNewReferralValidatesRewardInvariant(t *testing.T) {
	referrerID := uuid.New()
	referredID := uuid.New()

	tests := []struct {
		name       string
		referrerID uuid.UUID
		referredID uuid.UUID
		reward     decimal.Decimal
		wantErr    error
	}{
		{name: "missing referrer", referredID: referredID, reward: decimal.NewFromInt(1), wantErr: ErrReferralPartiesRequired},
		{name: "missing referred user", referrerID: referrerID, reward: decimal.NewFromInt(1), wantErr: ErrReferralPartiesRequired},
		{name: "self referral", referrerID: referrerID, referredID: referrerID, reward: decimal.NewFromInt(1), wantErr: ErrSelfReferral},
		{name: "zero reward", referrerID: referrerID, referredID: referredID, reward: decimal.Zero, wantErr: ErrInvalidReferralReward},
		{name: "negative reward", referrerID: referrerID, referredID: referredID, reward: decimal.NewFromInt(-1), wantErr: ErrInvalidReferralReward},
		{name: "unsafe exponent", referrerID: referrerID, referredID: referredID, reward: decimal.New(1, math.MaxInt32), wantErr: ErrInvalidReferralReward},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewReferral(tt.referrerID, tt.referredID, tt.reward)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewReferral() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestReferralHasStableIdentityAndCreditsBalance(t *testing.T) {
	referrerID := uuid.New()
	referredID := uuid.New()
	reward := decimal.RequireFromString("2.25")
	referral, err := NewReferral(referrerID, referredID, reward)
	if err != nil {
		t.Fatalf("NewReferral() error = %v", err)
	}

	again, err := NewReferral(referrerID, referredID, reward)
	if err != nil {
		t.Fatalf("second NewReferral() error = %v", err)
	}
	if referral.DedupeKey() != again.DedupeKey() {
		t.Fatalf("dedupe key is not deterministic: %q != %q", referral.DedupeKey(), again.DedupeKey())
	}
	if referral.DedupeKey() == "" {
		t.Fatal("dedupe key is empty")
	}
	got, err := referral.Credit(decimal.NewFromInt(4))
	if err != nil || !got.Equal(decimal.RequireFromString("6.25")) {
		t.Fatalf("Credit() = %s, want 6.25", got)
	}
	if _, err := referral.Credit(decimal.RequireFromString("99999999999999999999")); err == nil {
		t.Fatal("Credit() allowed a balance beyond NUMERIC(38,18)")
	}
}

func TestReferralIdentityBelongsToReferredUser(t *testing.T) {
	referred := uuid.New()
	first, err := NewReferral(uuid.New(), referred, decimal.NewFromInt(50))
	if err != nil {
		t.Fatal(err)
	}
	other, err := NewReferral(uuid.New(), referred, decimal.NewFromInt(100))
	if err != nil {
		t.Fatal(err)
	}
	if first.DedupeKey() != other.DedupeKey() {
		t.Fatal("changing referrer must not create a new reward identity")
	}
}
