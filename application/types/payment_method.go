package types

import "encoding/json"

// PublicPaymentMethod is the client-safe payment configuration. Persistence
// identity, audit timestamps and gateway credentials intentionally do not
// exist at this boundary, so they cannot be exposed by accidental JSON tags.
type PublicPaymentMethod struct {
	Kind               string          `json:"kind"`
	IBANDetails        json.RawMessage `json:"iban_details,omitempty"`
	IsIBANEnabled      bool            `json:"is_iban_enabled"`
	CryptoDetails      json.RawMessage `json:"crypto_details,omitempty"`
	IsCryptoEnabled    bool            `json:"is_crypto_enabled"`
	GooglePayDetails   json.RawMessage `json:"google_pay_details,omitempty"`
	IsGooglePayEnabled bool            `json:"is_google_pay_enabled"`
	Packages           json.RawMessage `json:"packages,omitempty"`
}
