package repositories

import (
	"core/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// lockTransactionResource must run in the caller's transaction. Creating the
// key and locking its row serializes all processes, including first writers,
// through GORM instead of PostgreSQL advisory-lock statements.
func lockTransactionResource(tx *gorm.DB, key string) *gorm.DB {
	row := models.TransactionLock{Key: key}
	result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key"}}, DoNothing: true}).Create(&row)
	if result.Error != nil {
		return result
	}
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(&models.TransactionLock{Key: key}).Take(&row)
}
