package payment

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type PaymentKind string

const (
	PaymentKind_IBAN      PaymentKind = "iban"
	PaymentKind_CRYPTO    PaymentKind = "crypto"
	PaymentKind_GOOGLEPAY PaymentKind = "google_pay"
)

/*

IBAN extends PaymentKind
CRYPTO extends PaymentKind
GOOGLE_PAY extends PaymentKind

*/

type PaymentMethod struct {
	ID                 uuid.UUID   `gorm:"type:uuid;primaryKey"  json:"id"`
	DefaultPaymentKind PaymentKind `gorm:"type:varchar(20)" json:"kind"`

	IBANDetails        datatypes.JSON `gorm:"type:jsonb" json:"iban_details"`
	IsIBANEnabled      bool           `gorm:"default:false" json:"is_iban_enabled"`
	CryptoDetails      datatypes.JSON `gorm:"type:jsonb" json:"crypto_details"`
	IsCryptoEnabled    bool           `gorm:"default:false" json:"is_crypto_enabled"`
	GooglePayDetails   datatypes.JSON `gorm:"type:jsonb" json:"google_pay_details"`
	IsGooglePayEnabled bool           `gorm:"default:false" json:"is_google_pay_enabled"`
	Packages           datatypes.JSON `gorm:"type:jsonb" json:"packages"`
	SecretKeys         datatypes.JSON `gorm:"type:jsonb" json:"secrets"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (PaymentMethod) TableName() string {
	return "payment_methods"
}
