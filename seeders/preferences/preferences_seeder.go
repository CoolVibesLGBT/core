package preferences

import (
	lgbt "core/seeders/preferences/lgbt"

	"gorm.io/gorm"
)

func SeedPreferences(db *gorm.DB) error {

	err := lgbt.SeedLGBTPreferences(db)
	return err

}
