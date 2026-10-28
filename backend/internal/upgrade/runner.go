package upgrade

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"ganigeorgiev.com/pocketbase/pocketbase/core"
)

const (
	migrationTable           = "_subflow_migrations"
	migrationCheckpointTable = "_subflow_migration_checkpoints"
)

type Migration struct {
	ID           string
	Checksum     string
	Requires     []string
	Precondition func(core.App) error
	Up           func(core.App) error
	// BatchUp applies one bounded batch and returns a durable cursor. The batch
	// data and cursor are committed in the same transaction. Returning done=true
	// runs verification, audit, and completion journaling in that transaction.
	BatchUp   func(core.App, string, int) (nextCursor string, done bool, err error)
	BatchSize int
	Verify    func(core.App) error
}

type migrationRow struct {
	ID        string `db:"migration_id"`
	Checksum  string `db:"checksum"`
	Status    string `db:"status"`
	AppliedAt string `db:"applied_at"`
}

type migrationCheckpoint struct {
	ID        string `db:"migration_id"`
	Checksum  string `db:"checksum"`
	Cursor    string `db:"cursor"`
	UpdatedAt string `db:"updated_at"`
}

// RunMigrations runs each immutable migration and its audit event in the same
// PocketBase transaction. The database journal is authoritative; the JSON
// journal only supports status display if the process is interrupted.
func RunMigrations(ctx context.Context, app core.App, manager *Manager, migrations []Migration, audit func(context.Context, core.App, Migration) error) error {
	if len(migrations) == 0 {
		return nil
	}
	if _, err := app.DB().NewQuery(`CREATE TABLE IF NOT EXISTS "` + migrationTable + `" (migration_id TEXT PRIMARY KEY NOT NULL, checksum TEXT NOT NULL, status TEXT NOT NULL, applied_at TEXT NOT NULL, source TEXT NOT NULL)`).Execute(); err != nil {
		return fmt.Errorf("create migration journal: %w", err)
	}
	if _, err := app.DB().NewQuery(`CREATE TABLE IF NOT EXISTS "` + migrationCheckpointTable + `" (migration_id TEXT PRIMARY KEY NOT NULL, checksum TEXT NOT NULL, cursor TEXT NOT NULL, updated_at TEXT NOT NULL)`).Execute(); err != nil {
		return fmt.Errorf("create migration checkpoint journal: %w", err)
	}
	registered := make(map[string]Migration, len(migrations))
	for _, migration := range migrations {
		validUp := migration.Up != nil && migration.BatchUp == nil && migration.BatchSize == 0
		validBatch := migration.Up == nil && migration.BatchUp != nil && migration.BatchSize > 0
		if migration.ID == "" || migration.Checksum == "" || (!validUp && !validBatch) {
			return fmt.Errorf("invalid migration definition")
		}
		if _, exists := registered[migration.ID]; exists {
			return fmt.Errorf("duplicate migration id %q", migration.ID)
		}
		registered[migration.ID] = migration
	}
	var checkpoints []migrationCheckpoint
	if err := app.DB().NewQuery(`SELECT migration_id, checksum, cursor, updated_at FROM "` + migrationCheckpointTable + `"`).All(&checkpoints); err != nil {
		return fmt.Errorf("read migration checkpoints: %w", err)
	}
	checkpointByID := make(map[string]migrationCheckpoint, len(checkpoints))
	for _, checkpoint := range checkpoints {
		migration, ok := registered[checkpoint.ID]
		if !ok {
			return fmt.Errorf("unknown migration checkpoint %q", checkpoint.ID)
		}
		if migration.BatchUp == nil {
			return fmt.Errorf("migration %s has a checkpoint but is not batched", checkpoint.ID)
		}
		if checkpoint.Checksum != migration.Checksum {
			return fmt.Errorf("migration checkpoint checksum mismatch for %s", checkpoint.ID)
		}
		checkpointByID[checkpoint.ID] = checkpoint
	}
	var existing []migrationRow
	if err := app.DB().NewQuery(`SELECT migration_id, checksum, status, applied_at FROM "` + migrationTable + `"`).All(&existing); err != nil {
		return fmt.Errorf("read migration journal: %w", err)
	}
	for _, row := range existing {
		migration, ok := registered[row.ID]
		if !ok {
			return fmt.Errorf("unknown migration %q in database journal", row.ID)
		}
		if row.Checksum != migration.Checksum {
			return fmt.Errorf("migration checksum mismatch for %s", row.ID)
		}
		if row.Status != "complete" {
			return fmt.Errorf("migration %s has incomplete journal state", row.ID)
		}
		if _, hasCheckpoint := checkpointByID[row.ID]; hasCheckpoint {
			return fmt.Errorf("completed migration %s has an unexpected checkpoint", row.ID)
		}
		appliedAt, err := time.Parse(time.RFC3339Nano, row.AppliedAt)
		if err != nil {
			return fmt.Errorf("migration %s has an invalid completion timestamp", row.ID)
		}
		entry := MigrationEntry{ID: row.ID, Checksum: row.Checksum, StartedAt: appliedAt, CompletedAt: appliedAt, Status: "complete"}
		if err := manager.AddHistory(entry); err != nil {
			return fmt.Errorf("reconcile migration history: %w", err)
		}
	}
	completed := make(map[string]bool, len(existing))
	for _, row := range existing {
		completed[row.ID] = true
	}
	for _, migration := range migrations {
		for _, dependency := range migration.Requires {
			if !completed[dependency] {
				return fmt.Errorf("migration %s requires unapplied migration %s", migration.ID, dependency)
			}
		}
		if completed[migration.ID] {
			continue
		}
		if migration.Precondition != nil {
			if err := migration.Precondition(app); err != nil {
				return fmt.Errorf("migration %s precondition failed: %w", migration.ID, err)
			}
		}
		started := time.Now().UTC()
		entry := MigrationEntry{ID: migration.ID, Checksum: migration.Checksum, StartedAt: started, Status: "running"}
		if err := manager.AddHistory(entry); err != nil {
			return fmt.Errorf("persist migration status: %w", err)
		}
		err := runMigration(ctx, app, migration, audit)
		if err != nil {
			entry.Status = "failed"
			entry.CompletedAt = time.Now().UTC()
			if persistErr := manager.AddHistory(entry); persistErr != nil {
				return fmt.Errorf("migration %s failed: %w; persist failure status: %v", migration.ID, err, persistErr)
			}
			return fmt.Errorf("migration %s failed: %w", migration.ID, err)
		}
		entry.Status, entry.CompletedAt = "complete", time.Now().UTC()
		if err := manager.AddHistory(entry); err != nil {
			return fmt.Errorf("persist completed migration status: %w", err)
		}
		completed[migration.ID] = true
	}
	return nil
}

func runMigration(ctx context.Context, app core.App, migration Migration, audit func(context.Context, core.App, Migration) error) error {
	if migration.BatchUp == nil {
		return app.RunInTransaction(func(tx core.App) error {
			if err := migration.Up(tx); err != nil {
				return err
			}
			return completeMigration(ctx, tx, migration, audit)
		})
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		done := false
		err := app.RunInTransaction(func(tx core.App) error {
			var checkpoint migrationCheckpoint
			err := tx.DB().NewQuery(`SELECT migration_id, checksum, cursor, updated_at FROM "` + migrationCheckpointTable + `" WHERE migration_id={:id}`).Bind(map[string]any{"id": migration.ID}).One(&checkpoint)
			cursor := ""
			if err == nil {
				if checkpoint.Checksum != migration.Checksum {
					return fmt.Errorf("migration checkpoint checksum mismatch for %s", migration.ID)
				}
				cursor = checkpoint.Cursor
			} else if err != sql.ErrNoRows {
				return fmt.Errorf("read migration checkpoint: %w", err)
			}

			nextCursor, batchDone, err := migration.BatchUp(tx, cursor, migration.BatchSize)
			if err != nil {
				return err
			}
			if !batchDone && nextCursor == cursor {
				return fmt.Errorf("migration %s batch did not advance its checkpoint", migration.ID)
			}
			done = batchDone
			if batchDone {
				if err := completeMigration(ctx, tx, migration, audit); err != nil {
					return err
				}
				_, err := tx.DB().NewQuery(`DELETE FROM "` + migrationCheckpointTable + `" WHERE migration_id={:id}`).Bind(map[string]any{"id": migration.ID}).Execute()
				return err
			}

			_, err = tx.DB().NewQuery(`INSERT INTO "` + migrationCheckpointTable + `" (migration_id, checksum, cursor, updated_at) VALUES ({:id}, {:checksum}, {:cursor}, {:updatedAt}) ON CONFLICT(migration_id) DO UPDATE SET checksum=excluded.checksum, cursor=excluded.cursor, updated_at=excluded.updated_at`).Bind(map[string]any{
				"id": migration.ID, "checksum": migration.Checksum, "cursor": nextCursor, "updatedAt": time.Now().UTC().Format(time.RFC3339Nano),
			}).Execute()
			return err
		})
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
}

func completeMigration(ctx context.Context, tx core.App, migration Migration, audit func(context.Context, core.App, Migration) error) error {
	if migration.Verify != nil {
		if err := migration.Verify(tx); err != nil {
			return fmt.Errorf("migration verification failed: %w", err)
		}
	}
	if audit != nil {
		if err := audit(ctx, tx, migration); err != nil {
			return fmt.Errorf("migration audit failed: %w", err)
		}
	}
	_, err := tx.DB().NewQuery(`INSERT INTO "` + migrationTable + `" (migration_id, checksum, status, applied_at, source) VALUES ({:id}, {:checksum}, 'complete', {:appliedAt}, {:source})`).Bind(map[string]any{"id": migration.ID, "checksum": migration.Checksum, "appliedAt": time.Now().UTC().Format(time.RFC3339Nano), "source": "subflow"}).Execute()
	return err
}

func MigrationChecksum(app core.App, id string) (string, error) {
	var row migrationRow
	err := app.DB().NewQuery(`SELECT migration_id, checksum, status FROM "` + migrationTable + `" WHERE migration_id={:id}`).Bind(map[string]any{"id": id}).One(&row)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return row.Checksum, nil
}
