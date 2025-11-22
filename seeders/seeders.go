package seeders

import (
	eventkinds "coolvibes/seeders/eventkinds"
	preferences "coolvibes/seeders/preferences"
	reportkinds "coolvibes/seeders/reportkinds"

	"gorm.io/gorm"
)

func Seed(db *gorm.DB) error {
	err := preferences.SeedPreferences(db)
	if err != nil {
		return err
	}

	err = eventkinds.SeedEventKinds(db)
	if err != nil {
		return err
	}

	err = reportkinds.SeedReportKinds(db)
	if err != nil {
		return err
	}

	return nil
}
