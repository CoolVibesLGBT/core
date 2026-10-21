package db

import (
	"core/models/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

func dropIndexIfExists(db *gorm.DB, model any, name string) error {
	if !db.Migrator().HasIndex(model, name) {
		return nil
	}
	return db.Migrator().DropIndex(model, name)
}

// The caller holds the migration table lock. Stream the winning owner rows
// through GORM, then soft-delete duplicates after closing the cursor.
func deduplicateLocationOwners(tx *gorm.DB) error {
	rows, err := tx.Model(&utils.Location{}).Select("id", "contentable_type", "contentable_id").
		Order("contentable_type ASC, contentable_id ASC, updated_at DESC NULLS LAST, created_at DESC NULLS LAST, id DESC").Rows()
	if err != nil {
		return err
	}
	var duplicates []uuid.UUID
	var previousType string
	var previousID uuid.UUID
	first := true
	for rows.Next() {
		var location utils.Location
		if err := tx.ScanRows(rows, &location); err != nil {
			_ = rows.Close()
			return err
		}
		if !first && location.ContentableType == previousType && location.ContentableID == previousID {
			duplicates = append(duplicates, location.ID)
		}
		previousType, previousID, first = location.ContentableType, location.ContentableID, false
	}
	scanErr := rows.Err()
	closeErr := rows.Close()
	if scanErr != nil {
		return scanErr
	}
	if closeErr != nil {
		return closeErr
	}
	now := time.Now().UTC()
	for start := 0; start < len(duplicates); start += 200 {
		end := min(start+200, len(duplicates))
		if err := tx.Model(&utils.Location{}).Where("id IN ?", duplicates[start:end]).Updates(map[string]any{"deleted_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
	}
	return nil
}
