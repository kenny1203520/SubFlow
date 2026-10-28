package upgrade

import (
	"context"
	"testing"

	"ganigeorgiev.com/pocketbase/pocketbase/core"
	pbadapter "subflow/internal/adapters/pocketbase"

	"ganigeorgiev.com/pocketbase/pocketbase/tests"
)

func TestDurableUpgradeEventsImportExactlyOnce(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	if err := pbadapter.EnsureSchema(app); err != nil {
		t.Fatal(err)
	}
	manager, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.AppendEvent("upgrade.backup.created", "success", "verified backup"); err != nil {
		t.Fatal(err)
	}
	if err := ReconcileDurableEvents(context.Background(), app, "pocketbase", manager.Dir, "test-only-root-key"); err != nil {
		t.Fatal(err)
	}
	if err := ReconcileDurableEvents(context.Background(), app, "pocketbase", manager.Dir, "test-only-root-key"); err != nil {
		t.Fatal(err)
	}
	events, err := app.FindRecordsByFilter("audit_logs", "action={:action}", "", 0, 0, map[string]any{"action": "upgrade.backup.created"})
	if err != nil || len(events) != 1 {
		t.Fatalf("expected one imported durable event, found %d: %v", len(events), err)
	}
	if events[0].GetString("source_event_id") == "" || events[0].GetString("outcome") != "success" {
		t.Fatalf("durable event metadata was not stored: %#v", events[0].PublicExport())
	}
}

func TestAlpha1AuditResourceIDLimitMigrationAllowsDurableEventImport(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	if err := pbadapter.EnsureSchema(app); err != nil {
		t.Fatal(err)
	}
	collection, err := app.FindCollectionByNameOrId("audit_logs")
	if err != nil {
		t.Fatal(err)
	}
	resourceID, ok := collection.Fields.GetByName("resource_id").(*core.TextField)
	if !ok {
		t.Fatal("audit resource_id field is not a text field")
	}
	resourceID.Max = 32 // alpha.1 schema
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}
	manager, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.AppendEvent("upgrade.backup.created", "success", "verified backup"); err != nil {
		t.Fatal(err)
	}
	setupLink := ""
	if err := RunMigrations(context.Background(), app, manager, []Migration{
		legacyAlphaBaseline("pocketbase", "http://localhost:8080", &setupLink),
		auditIdLimit(),
	}, MigrationAudit("pocketbase", "test-only-root-key")); err != nil {
		t.Fatal(err)
	}
	if err := ReconcileDurableEvents(context.Background(), app, "pocketbase", manager.Dir, "test-only-root-key"); err != nil {
		t.Fatal(err)
	}
	events, err := app.FindRecordsByFilter("audit_logs", "action={:action}", "", 0, 0, map[string]any{"action": "upgrade.backup.created"})
	if err != nil || len(events) != 1 {
		t.Fatalf("expected one imported durable event, found %d: %v", len(events), err)
	}
	if len(events[0].GetString("resource_id")) != 40 {
		t.Fatalf("durable event resource ID was truncated: %q", events[0].GetString("resource_id"))
	}
}
