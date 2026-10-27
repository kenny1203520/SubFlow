package upgrade

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"ganigeorgiev.com/pocketbase/pocketbase/core"
	"ganigeorgiev.com/pocketbase/pocketbase/tests"

	pbadapter "subflow/internal/adapters/pocketbase"
)

func TestMigrationJournalIsIdempotentAndChecksumProtected(t *testing.T) {
	app := newMigrationTestApp(t)
	manager, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	auditCalls := 0
	migration := testMigration()
	audit := func(context.Context, core.App, Migration) error { auditCalls++; return nil }
	for i := 0; i < 2; i++ {
		if err := RunMigrations(context.Background(), app, manager, []Migration{migration}, audit); err != nil {
			t.Fatal(err)
		}
	}
	if auditCalls != 1 {
		t.Fatalf("migration audit ran %d times", auditCalls)
	}
	var inserted int
	if err := app.DB().NewQuery("SELECT COUNT(*) FROM upgrade_fixture").Row(&inserted); err != nil {
		t.Fatal(err)
	}
	if inserted != 1 {
		t.Fatalf("migration ran %d times", inserted)
	}
	migration.Checksum = "sha256:changed"
	if err := RunMigrations(context.Background(), app, manager, []Migration{migration}, audit); err == nil {
		t.Fatal("changed migration checksum unexpectedly passed")
	}
}

func TestMigrationAuditFailureRollsBackDataAndCompletionJournal(t *testing.T) {
	app := newMigrationTestApp(t)
	manager, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.DB().NewQuery("CREATE TABLE upgrade_fixture (id INTEGER PRIMARY KEY)").Execute(); err != nil {
		t.Fatal(err)
	}
	migration := testMigration()
	if err := RunMigrations(context.Background(), app, manager, []Migration{migration}, func(context.Context, core.App, Migration) error {
		return errors.New("audit unavailable")
	}); err == nil {
		t.Fatal("audit failure unexpectedly completed migration")
	}
	var inserted, journal int
	if err := app.DB().NewQuery("SELECT COUNT(*) FROM upgrade_fixture").Row(&inserted); err != nil {
		t.Fatal(err)
	}
	if err := app.DB().NewQuery("SELECT COUNT(*) FROM " + migrationTable).Row(&journal); err != nil {
		t.Fatal(err)
	}
	if inserted != 0 || journal != 0 {
		t.Fatalf("migration did not roll back atomically: data=%d journal=%d", inserted, journal)
	}
}

func TestBatchedMigrationResumesFromAtomicallyCommittedCheckpoint(t *testing.T) {
	app := newMigrationTestApp(t)
	manager, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.DB().NewQuery("CREATE TABLE upgrade_fixture (id INTEGER PRIMARY KEY)").Execute(); err != nil {
		t.Fatal(err)
	}

	failAfterFirstBatch := true
	auditCalls := 0
	migration := Migration{
		ID: "0001_batched_fixture", Checksum: "sha256:batched-fixture", BatchSize: 2,
		BatchUp: func(app core.App, cursor string, limit int) (string, bool, error) {
			start := 0
			if cursor != "" {
				parsed, err := strconv.Atoi(cursor)
				if err != nil {
					return cursor, false, err
				}
				start = parsed
			}
			end := start + limit
			if end > 5 {
				end = 5
			}
			for id := start + 1; id <= end; id++ {
				if _, err := app.DB().NewQuery("INSERT INTO upgrade_fixture (id) VALUES ({:id})").Bind(map[string]any{"id": id}).Execute(); err != nil {
					return cursor, false, err
				}
			}
			if failAfterFirstBatch && start == 2 {
				return cursor, false, errors.New("simulated interrupted batch")
			}
			return strconv.Itoa(end), end == 5, nil
		},
		Verify: func(app core.App) error {
			var count int
			if err := app.DB().NewQuery("SELECT COUNT(*) FROM upgrade_fixture").Row(&count); err != nil {
				return err
			}
			if count != 5 {
				return errors.New("batched migration verification failed")
			}
			return nil
		},
	}
	audit := func(context.Context, core.App, Migration) error { auditCalls++; return nil }
	if err := RunMigrations(context.Background(), app, manager, []Migration{migration}, audit); err == nil {
		t.Fatal("simulated interrupted batch unexpectedly completed")
	}
	var inserted int
	if err := app.DB().NewQuery("SELECT COUNT(*) FROM upgrade_fixture").Row(&inserted); err != nil {
		t.Fatal(err)
	}
	var checkpoint migrationCheckpoint
	if err := app.DB().NewQuery(`SELECT migration_id, checksum, cursor, updated_at FROM "` + migrationCheckpointTable + `" WHERE migration_id={:id}`).Bind(map[string]any{"id": migration.ID}).One(&checkpoint); err != nil {
		t.Fatal(err)
	}
	if inserted != 2 || checkpoint.Cursor != "2" || auditCalls != 0 {
		t.Fatalf("failed batch wasn't rolled back with its cursor: rows=%d cursor=%q audits=%d", inserted, checkpoint.Cursor, auditCalls)
	}

	changed := migration
	changed.Checksum = "sha256:mutated-batch"
	if err := RunMigrations(context.Background(), app, manager, []Migration{changed}, audit); err == nil {
		t.Fatal("changed migration checkpoint checksum unexpectedly passed")
	}

	failAfterFirstBatch = false
	if err := RunMigrations(context.Background(), app, manager, []Migration{migration}, audit); err != nil {
		t.Fatal(err)
	}
	if err := app.DB().NewQuery("SELECT COUNT(*) FROM upgrade_fixture").Row(&inserted); err != nil {
		t.Fatal(err)
	}
	var journalCount, checkpointCount int
	if err := app.DB().NewQuery("SELECT COUNT(*) FROM " + migrationTable).Row(&journalCount); err != nil {
		t.Fatal(err)
	}
	if err := app.DB().NewQuery("SELECT COUNT(*) FROM " + migrationCheckpointTable).Row(&checkpointCount); err != nil {
		t.Fatal(err)
	}
	if inserted != 5 || journalCount != 1 || checkpointCount != 0 || auditCalls != 1 {
		t.Fatalf("resumed batch migration didn't finish cleanly: rows=%d journal=%d checkpoint=%d audits=%d", inserted, journalCount, checkpointCount, auditCalls)
	}
}

func newMigrationTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	if err := pbadapter.EnsureSchema(app); err != nil {
		t.Fatal(err)
	}
	return app
}

func testMigration() Migration {
	return Migration{
		ID: "0001_fixture", Checksum: "sha256:fixture",
		Up: func(app core.App) error {
			if _, err := app.DB().NewQuery("CREATE TABLE IF NOT EXISTS upgrade_fixture (id INTEGER PRIMARY KEY)").Execute(); err != nil {
				return err
			}
			_, err := app.DB().NewQuery("INSERT INTO upgrade_fixture (id) VALUES (1)").Execute()
			return err
		},
		Verify: func(app core.App) error {
			var count int
			if err := app.DB().NewQuery("SELECT COUNT(*) FROM upgrade_fixture").Row(&count); err != nil {
				return err
			}
			if count != 1 {
				return errors.New("migration fixture verification failed")
			}
			return nil
		},
	}
}
