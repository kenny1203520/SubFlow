package upgrade

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// lockedAfterReady reproduces the state 0.2.0-beta.1 left behind when a
// post-upgrade restart could not find its recovery snapshot: the upgrade had
// completed, but the serve error was recorded as an upgrade failure.
func lockedAfterReady(t *testing.T) *Manager {
	t.Helper()
	manager, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeSQLiteFixture(t, filepath.Join(manager.DataDir, "data.db"))
	if err := manager.Save(State{
		State: StatusFailed, Phase: StatusFailed, FailurePhase: StatusReady, ProblemCode: "backup_failed",
		AppVersion: "0.2.0-beta.1", PocketBaseVersion: "0.41.0", Source: "upgrade",
		CompletedAt: time.Now().UTC().Add(-time.Hour), BackupID: "20261113T095935.203701863Z",
	}); err != nil {
		t.Fatal(err)
	}
	if err := manager.Acquire(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	return manager
}

func durableEvents(t *testing.T, manager *Manager) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(manager.Dir, "events.jsonl"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return string(data)
}

func TestStartupRecoversFailureRecordedAfterReady(t *testing.T) {
	for _, executable := range []string{"0.2.0-beta.1", "0.2.0-beta.3"} {
		t.Run(executable, func(t *testing.T) {
			manager := lockedAfterReady(t)
			if err := manager.BeginStartup(); err != nil {
				t.Fatal(err)
			}
			if err := manager.Prepare(executable, "0.41.0", false); err != nil {
				t.Fatalf("startup stayed locked after a post-ready failure: %v", err)
			}
			state, err := manager.ReadState()
			if err != nil {
				t.Fatal(err)
			}
			if state.State == StatusFailed || state.ProblemCode != "" || state.FailurePhase != "" || state.FirstFailure != nil {
				t.Fatalf("failure was not cleared: %#v", state)
			}
			if state.BackupID == "20261113T095935.203701863Z" {
				t.Fatalf("missing recovery snapshot is still referenced: %#v", state)
			}
			events := durableEvents(t, manager)
			if strings.Count(events, `"action":"upgrade.recovered"`) != 1 || !strings.Contains(events, "code=backup_failed") {
				t.Fatalf("recovery was not audited exactly once: %s", events)
			}
			if err := manager.Complete(); err != nil {
				t.Fatalf("recovered startup could not complete: %v", err)
			}
		})
	}
}

func TestStartupKeepsFailuresFromBeforeReadyLocked(t *testing.T) {
	for _, phase := range []string{StatusBackingUp, StatusMigrating, StatusVerifying, ""} {
		t.Run("phase="+phase, func(t *testing.T) {
			manager := lockedAfterReady(t)
			state, err := manager.ReadState()
			if err != nil {
				t.Fatal(err)
			}
			state.FailurePhase = phase
			if err := manager.Save(state); err != nil {
				t.Fatal(err)
			}
			if err := manager.BeginStartup(); err != nil {
				t.Fatal(err)
			}
			if err := manager.Prepare("0.2.0-beta.3", "0.41.0", false); !errors.Is(err, ErrPreviousFailure) {
				t.Fatalf("a failure before readiness must require an operator retry: %v", err)
			}
			if events := durableEvents(t, manager); strings.Contains(events, "upgrade.recovered") {
				t.Fatalf("unexpected recovery event: %s", events)
			}
		})
	}
}

// An operator command can also record a failure while the data is ready: an
// older executable refusing the newer data leaves failurePhase "ready" too.
// That is a deliberate block, not a serve error, and must wait for an operator.
func TestStartupKeepsOperatorBlocksRecordedWhileReadyLocked(t *testing.T) {
	for _, code := range []string{"unsupported_downgrade", "invalid_upgrade_version", "restore_recovery_failed", "some_future_code"} {
		t.Run(code, func(t *testing.T) {
			manager := lockedAfterReady(t)
			state, err := manager.ReadState()
			if err != nil {
				t.Fatal(err)
			}
			state.ProblemCode = code
			if err := manager.Save(state); err != nil {
				t.Fatal(err)
			}
			if err := manager.BeginStartup(); err != nil {
				t.Fatal(err)
			}
			if err := manager.Prepare("0.2.0-beta.3", "0.41.0", false); !errors.Is(err, ErrPreviousFailure) {
				t.Fatalf("%s recorded while ready was cleared without an operator: %v", code, err)
			}
			if events := durableEvents(t, manager); strings.Contains(events, "upgrade.recovered") {
				t.Fatalf("unexpected recovery event: %s", events)
			}
		})
	}
}

func TestStartupRecoversEveryServeFailureCode(t *testing.T) {
	for _, code := range []string{"backup_failed", "audit_unavailable", "migration_failed", "upgrade_failed", "upgrade_finalize_failed", "maintenance_server_unavailable"} {
		t.Run(code, func(t *testing.T) {
			manager := lockedAfterReady(t)
			state, err := manager.ReadState()
			if err != nil {
				t.Fatal(err)
			}
			state.ProblemCode = code
			if err := manager.Save(state); err != nil {
				t.Fatal(err)
			}
			if err := manager.BeginStartup(); err != nil {
				t.Fatal(err)
			}
			if err := manager.Prepare("0.2.0-beta.3", "0.41.0", false); err != nil {
				t.Fatalf("%s recorded after readiness stayed locked: %v", code, err)
			}
		})
	}
}

func TestStartupStaysLockedWhenRecoveryCannotBeAudited(t *testing.T) {
	manager := lockedAfterReady(t)
	// A directory in place of the event log makes every durable append fail.
	if err := os.Mkdir(filepath.Join(manager.Dir, "events.jsonl"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := manager.BeginStartup(); err != nil {
		t.Fatal(err)
	}
	if err := manager.Prepare("0.2.0-beta.3", "0.41.0", false); !errors.Is(err, ErrPreviousFailure) {
		t.Fatalf("recovery without an audit record must not unlock startup: %v", err)
	}
	state, err := manager.ReadState()
	if err != nil {
		t.Fatal(err)
	}
	if state.State != StatusFailed || state.ProblemCode != "backup_failed" || state.FailurePhase != StatusReady {
		t.Fatalf("failed state was altered without an audit record: %#v", state)
	}
}

func TestInterruptedRoutineRestartDoesNotLockStartup(t *testing.T) {
	for _, phase := range []string{StatusMigrating, StatusVerifying} {
		t.Run(phase, func(t *testing.T) {
			directory := t.TempDir()
			manager, err := New(directory)
			if err != nil {
				t.Fatal(err)
			}
			writeSQLiteFixture(t, filepath.Join(manager.DataDir, "data.db"))
			completedAt := time.Now().UTC().Add(-time.Hour)
			if err := manager.Save(State{
				State: StatusReady, Phase: StatusReady, Ready: true, AppVersion: "0.2.0-beta.3", PocketBaseVersion: "0.41.0",
				Source: "upgrade", CompletedAt: completedAt,
			}); err != nil {
				t.Fatal(err)
			}
			if err := manager.Acquire(); err != nil {
				t.Fatal(err)
			}
			if err := manager.BeginStartup(); err != nil {
				t.Fatal(err)
			}
			if err := manager.Prepare("0.2.0-beta.3", "0.41.0", false); err != nil {
				t.Fatal(err)
			}
			// Every start reports these phases while PocketBase bootstraps; the
			// process is killed here without anything left to migrate.
			if err := manager.SetPhase(phase); err != nil {
				t.Fatal(err)
			}
			if err := manager.Close(); err != nil {
				t.Fatal(err)
			}

			next, err := New(directory)
			if err != nil {
				t.Fatal(err)
			}
			if err := next.Acquire(); err != nil {
				t.Fatal(err)
			}
			defer next.Close()
			if err := next.BeginStartup(); err != nil {
				t.Fatal(err)
			}
			if err := next.Prepare("0.2.0-beta.3", "0.41.0", false); err != nil {
				t.Fatalf("an interrupted ordinary restart locked startup: %v", err)
			}
			state, err := next.ReadState()
			if err != nil {
				t.Fatal(err)
			}
			if state.State == StatusFailed || state.BackupID != "" || !state.CompletedAt.Equal(completedAt) {
				t.Fatalf("ordinary restart was treated as an upgrade: %#v", state)
			}
		})
	}
}

func TestInterruptedUpgradeStillRequiresRetry(t *testing.T) {
	directory := t.TempDir()
	manager, err := New(directory)
	if err != nil {
		t.Fatal(err)
	}
	writeSQLiteFixture(t, filepath.Join(manager.DataDir, "data.db"))
	if err := manager.Save(State{
		State: StatusReady, Phase: StatusReady, Ready: true, AppVersion: "0.2.0-beta.2", PocketBaseVersion: "0.41.0",
		Source: "upgrade", CompletedAt: time.Now().UTC().Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err := manager.Acquire(); err != nil {
		t.Fatal(err)
	}
	if err := manager.BeginStartup(); err != nil {
		t.Fatal(err)
	}
	if err := manager.Prepare("0.2.0-beta.3", "0.41.0", false); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetPhase(StatusMigrating); err != nil {
		t.Fatal(err)
	}
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}

	next, err := New(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := next.Acquire(); err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	if err := next.BeginStartup(); err != nil {
		t.Fatal(err)
	}
	if err := next.Prepare("0.2.0-beta.3", "0.41.0", false); !errors.Is(err, ErrPreviousFailure) {
		t.Fatalf("an interrupted version upgrade must stay locked: %v", err)
	}
	state, err := next.ReadState()
	if err != nil {
		t.Fatal(err)
	}
	if state.ProblemCode != "upgrade_interrupted" || state.FailurePhase != StatusMigrating {
		t.Fatalf("interrupted upgrade was not recorded: %#v", state)
	}
}
