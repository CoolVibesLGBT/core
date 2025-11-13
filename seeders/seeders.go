package seeders

import (
	preferences "coolvibes/seeders/preferences"

	"gorm.io/gorm"
)

func Seed(db *gorm.DB) error {

	err := preferences.SeedPreferences(db)
	return err

}
