package seeders

import (
	"coolvibes/application"
	default_users "coolvibes/seeders/default_users"
	eventkinds "coolvibes/seeders/eventkinds"
	payments "coolvibes/seeders/payments"
	places "coolvibes/seeders/places"
	preferences "coolvibes/seeders/preferences"
	reportkinds "coolvibes/seeders/reportkinds"
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
