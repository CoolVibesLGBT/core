package seeders

import (
	"core/application"
	default_users "core/seeders/default_users"
	eventkinds "core/seeders/eventkinds"
	payments "core/seeders/payments"
	places "core/seeders/places"
	preferences "core/seeders/preferences"
	reportkinds "core/seeders/reportkinds"
)

func Seed(app *application.App) error {
	err := preferences.SeedPreferences(app.DB)
	if err != nil {
		return err
	}

	err = default_users.SeedDefaultSystemUsers(app)
	if err != nil {
		return err
	}

	err = eventkinds.SeedEventKinds(app.DB)
	if err != nil {
		return err
	}

	err = reportkinds.SeedReportKinds(app.DB)
	if err != nil {
		return err
	}

	err = reportkinds.SeedReportKinds(app.DB)
	if err != nil {
		return err
	}

	err = payments.SeedPackagesAndPaymentMethods(app.DB)
	if err != nil {
		return err
	}

	err = places.SeedPlaces(app)
	if err != nil {
		return err
	}

	return nil
}
