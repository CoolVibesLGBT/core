package types

import (
	"bytes"
	"testing"
)

func TestOpaqueIDRoundTripAndTypeBoundary(t *testing.T) {
	raw := [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	token := EncodeOpaqueID("pc", raw)
	if token == "" || bytes.Contains([]byte(token), []byte("00010203-")) {
		t.Fatalf("expected opaque token, got %q", token)
	}

	decoded, err := DecodeOpaqueID("pc", token)
	if err != nil {
		t.Fatalf("DecodeOpaqueID() error = %v", err)
	}
	if decoded != raw {
		t.Fatalf("round trip = %v, want %v", decoded, raw)
	}
	if _, err := DecodeOpaqueID("event", token); err == nil {
		t.Fatal("expected token prefix mismatch to fail")
	}
	if _, err := DecodeOpaqueID("pc", "pc_not-base64!"); err == nil {
		t.Fatal("expected malformed token to fail")
	}
}
