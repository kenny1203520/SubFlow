package upgrade

import (
	"context"
	"testing"

	"ganigeorgiev.com/pocketbase/pocketbase/core"

	pbadapter "subflow/internal/adapters/pocketbase"
)

func TestPasswordResetSettingsMigrationUsesInstallationStateDefaults(t *testing.T) {
	for _, initialized := range []bool{false, true} {
		t.Run(map[bool]string{false: "new_install", true: "existing_install"}[initialized], func(t *testing.T) {
			app := newMigrationTestApp(t)
			settings, err := app.FindFirstRecordByFilter(pbadapter.CollectionSystemSettings, "key='primary'", nil)
			if err != nil {
				t.Fatal(err)
			}
			settings.Set("initialized", initialized)
			if err := app.Save(settings); err != nil {
				t.Fatal(err)
			}
			outbox, err := app.FindCollectionByNameOrId(pbadapter.CollectionNotificationEmailOutbox)
			if err != nil {
				t.Fatal(err)
			}
			if outbox.Fields.GetByName("recipient_email") != nil {
				outbox.Fields.RemoveByName("recipient_email")
				if err := app.Save(outbox); err != nil {
					t.Fatal(err)
				}
			}

			manager, err := New(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			setupLink := ""
			if err := RunMigrations(context.Background(), app, manager, Migrations("pocketbase", "https://localhost:8090", &setupLink), MigrationAudit("pocketbase", "test-only-root-key")); err != nil {
				t.Fatal(err)
			}
			settings, err = app.FindRecordById(pbadapter.CollectionSystemSettings, settings.Id)
			if err != nil {
				t.Fatal(err)
			}
			if got := settings.GetBool("allow_password_reset"); got != initialized {
				t.Fatalf("allow_password_reset = %t, want %t", got, initialized)
			}
			settingsCollection, err := app.FindCollectionByNameOrId(pbadapter.CollectionSystemSettings)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := settingsCollection.Fields.GetByName("allow_password_reset").(*core.BoolField); !ok {
				t.Fatal("allow_password_reset field was not added")
			}
			outbox, err = app.FindCollectionByNameOrId(pbadapter.CollectionNotificationEmailOutbox)
			if err != nil {
				t.Fatal(err)
			}
			if recipient, ok := outbox.Fields.GetByName("recipient_email").(*core.TextField); !ok || !recipient.Hidden {
				t.Fatal("private notification recipient field was not added")
			}
		})
	}
}
