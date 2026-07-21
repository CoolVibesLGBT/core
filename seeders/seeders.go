package seeders

import (
	"core/helpers"
	default_users "core/seeders/default_users"
	eventkinds "core/seeders/eventkinds"
	payments "core/seeders/payments"
	places "core/seeders/places"
	preferences "core/seeders/preferences"
	reportkinds "core/seeders/reportkinds"

	"gorm.io/gorm"
)

func Seed(db *gorm.DB, node *helpers.Node) error {
	err := preferences.SeedPreferences(db)
	if err != nil {
		return err
	}

	err = default_users.SeedDefaultSystemUsers(db, node)
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

	err = payments.SeedPackagesAndPaymentMethods(db)
	if err != nil {
		return err
	}

	err = places.SeedPlaces(db, node)
	if err != nil {
		return err
	}

	return nil
}
