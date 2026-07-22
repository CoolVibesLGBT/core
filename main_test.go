package main

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

type concurrentShutdownProbe struct {
	started chan<- struct{}
	release <-chan struct{}
	once    sync.Once
}

func (p *concurrentShutdownProbe) Shutdown(ctx context.Context) error {
	p.once.Do(func() { p.started <- struct{}{} })
	select {
	case <-p.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

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

func TestShutdownStopsProcessorsConcurrently(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	first := &concurrentShutdownProbe{started: started, release: release}
	second := &concurrentShutdownProbe{started: started, release: release}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- shutdown(ctx, nil, nil, first, second) }()

	for range 2 {
		select {
		case <-started:
		case <-time.After(250 * time.Millisecond):
			t.Fatal("shutdown resources were not started concurrently")
		}
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}
