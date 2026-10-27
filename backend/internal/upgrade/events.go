package upgrade

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"ganigeorgiev.com/pocketbase/pocketbase/core"

	"subflow/internal/adapters"
	"subflow/internal/application"
)

// ReconcileDurableEvents imports pre-database upgrade events after the audit
// collection is available. The source_event_id unique index makes repeated
// startup imports idempotent, and import failure blocks readiness.
func ReconcileDurableEvents(ctx context.Context, app core.App, driver, directory string) error {
	file, err := os.Open(filepath.Join(directory, "events.jsonl"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 64*1024)
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return fmt.Errorf("durable upgrade event log is corrupt")
		}
		if len(event.ID) < 16 || len(event.ID) > 64 || !safeEventValue(event.Action) || !safeEventValue(event.Outcome) || len(event.Summary) > 256 || event.CreatedAt.IsZero() {
			return fmt.Errorf("durable upgrade event is invalid")
		}
		for _, character := range event.ID {
			if !(character >= '0' && character <= '9' || character >= 'a' && character <= 'f' || character == '-') {
				return fmt.Errorf("durable upgrade event ID is invalid")
			}
		}
		if err := importEvent(ctx, app, driver, event); err != nil {
			return fmt.Errorf("durable upgrade event import failed")
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("durable upgrade event log could not be read")
	}
	return nil
}

func importEvent(ctx context.Context, app core.App, driver string, event Event) error {
	return app.RunInTransaction(func(tx core.App) error {
		var existing struct {
			ID string `db:"id"`
		}
		err := tx.DB().NewQuery("SELECT id FROM audit_logs WHERE source_event_id={:eventId}").Bind(map[string]any{"eventId": event.ID}).One(&existing)
		if err == nil {
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		stores, err := adapters.New(driver, tx)
		if err != nil {
			return err
		}
		return application.New(stores).RecordAuditEvent(ctx, event.ID, event.Action, event.Outcome, event.Summary)
	})
}
