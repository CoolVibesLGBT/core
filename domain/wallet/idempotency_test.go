package wallet

import (
	"errors"
	"strings"
	"testing"
)

func TestNewIdempotencyKey(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    IdempotencyKey
		wantErr error
	}{
		{name: "uuid", input: " 019c-key_1234.retry ", want: "019c-key_1234.retry"},
		{name: "missing", input: "   ", wantErr: ErrIdempotencyKeyRequired},
		{name: "short", input: "short", wantErr: ErrInvalidIdempotencyKey},
		{name: "long", input: strings.Repeat("a", MaxIdempotencyKeyLength+1), wantErr: ErrInvalidIdempotencyKey},
		{name: "unsafe", input: "key with spaces", wantErr: ErrInvalidIdempotencyKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewIdempotencyKey(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewIdempotencyKey() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("NewIdempotencyKey() = %q, want %q", got, tt.want)
			}
		})
	}
}
