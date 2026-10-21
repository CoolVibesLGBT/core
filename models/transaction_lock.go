package models

// TransactionLock provides a persistent row for resources that do not yet
// have an aggregate row. Never delete lock rows while the application runs:
// another transaction may already be waiting on that exact row.
type TransactionLock struct {
	Key string `gorm:"type:text;primaryKey"`
}
