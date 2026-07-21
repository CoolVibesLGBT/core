package main

import (
	"errors"
	"reflect"
	"testing"
)

func TestMaintenanceModesInstallRunsMigrationAndSeed(t *testing.T) {
	runMigrate, runSeed := maintenanceModes(false, false, true)
	if !runMigrate || !runSeed {
		t.Fatalf("install modes = migrate:%v seed:%v, want both true", runMigrate, runSeed)
	}
}

func TestExecuteMaintenanceRunsMigrationBeforeSeed(t *testing.T) {
	var calls []string
	err := executeMaintenance(true, true, maintenanceSteps{
		migrate: func() error {
			calls = append(calls, "migrate")
			return nil
		},
		seed: func() error {
			calls = append(calls, "seed")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("executeMaintenance() error = %v", err)
	}
	if want := []string{"migrate", "seed"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestExecuteMaintenanceStopsOnMigrationError(t *testing.T) {
	migrationErr := errors.New("migration failed")
	seedCalled := false
	err := executeMaintenance(true, true, maintenanceSteps{
		migrate: func() error { return migrationErr },
		seed: func() error {
			seedCalled = true
			return nil
		},
	})
	if !errors.Is(err, migrationErr) {
		t.Fatalf("executeMaintenance() error = %v, want %v", err, migrationErr)
	}
	if seedCalled {
		t.Fatal("seed ran after migration failure")
	}
}
