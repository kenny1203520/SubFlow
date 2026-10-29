package upgrade

import (
	"context"
	"testing"

	"ganigeorgiev.com/pocketbase/pocketbase/core"
)

func TestSettlementApprovalsMigrationAddsSchemaAndIsIdempotent(t *testing.T) {
	app := newMigrationTestApp(t)
	manager, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	setupLink := ""
	migrations := Migrations("pocketbase", "https://localhost:8090", &setupLink)
	for range 2 {
		if err := RunMigrations(context.Background(), app, manager, migrations, MigrationAudit("pocketbase", "test-only-root-key")); err != nil {
			t.Fatal(err)
		}
	}

	groups, err := app.FindCollectionByNameOrId("groups")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := groups.Fields.GetByName("settlement_approval_enabled").(*core.BoolField); !ok {
		t.Fatal("group settlement approval setting was not added")
	}
	settlements, err := app.FindCollectionByNameOrId("settlements")
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]func(core.Field) bool{
		"status":            func(field core.Field) bool { _, ok := field.(*core.SelectField); return ok },
		"approval_required": func(field core.Field) bool { _, ok := field.(*core.BoolField); return ok },
		"version":           func(field core.Field) bool { _, ok := field.(*core.NumberField); return ok },
	} {
		if field := settlements.Fields.GetByName(name); field == nil || !want(field) {
			t.Fatalf("settlement field %q has wrong type or is missing", name)
		}
	}
	if _, err := app.FindCollectionByNameOrId("settlement_reviews"); err != nil {
		t.Fatalf("settlement review collection was not added: %v", err)
	}
	if _, err := app.DB().NewQuery(`SELECT migration_id FROM "` + migrationTable + `" WHERE migration_id={:id}`).Bind(map[string]any{"id": SettlementApprovalsID}).Execute(); err != nil {
		t.Fatal(err)
	}
}
