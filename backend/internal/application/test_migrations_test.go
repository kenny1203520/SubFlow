package application_test

import (
	"context"
	"testing"

	"ganigeorgiev.com/pocketbase/pocketbase/tests"

	"subflow/internal/upgrade"
)

func applyLatestTestMigrations(t *testing.T, app *tests.TestApp) {
	t.Helper()
	manager, err := upgrade.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := manager.Close(); err != nil {
			t.Errorf("close test migration manager: %v", err)
		}
	})
	setupLink := ""
	if err := upgrade.RunMigrations(
		context.Background(),
		app,
		manager,
		upgrade.Migrations("pocketbase", "http://localhost:8080", &setupLink),
		upgrade.MigrationAudit("pocketbase"),
	); err != nil {
		t.Fatal(err)
	}
}
