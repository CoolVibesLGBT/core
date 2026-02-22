package models

import (
	"time"

	"github.com/google/uuid"
)

type Wallet struct {
	ID uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`

	UserID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"-"`

	HDAccountID uint32 `gorm:"not null" json:"hd_account_id"`
	HDAddressId uint32 `gorm:"not null" json:"hd_address_id"`

	BitcoinAddress   string `gorm:"size:128;uniqueIndex;not null" json:"bitcoin"`
	EthereumAddress  string `gorm:"size:128;uniqueIndex;not null" json:"ethereum"`
	AvalancheAddress string `gorm:"size:128;uniqueIndex;not null" json:"avalanche"`
	TronAddress      string `gorm:"size:128;uniqueIndex;not null" json:"tron"`
	SolanaAddress    string `gorm:"size:128;uniqueIndex;not null" json:"solana"`
	ChilizAddress    string `gorm:"size:128;uniqueIndex;not null" json:"chiliz"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
