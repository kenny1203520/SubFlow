package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"subflow/internal/upgrade"
)

func TestGuardBackgroundSurvivesPanicFromClosedDatabase(t *testing.T) {
	var logged []string
	logf := func(format string, args ...any) { logged = append(logged, fmt.Sprintf(format, args...)) }
	job := guardBackground(context.Background(), "initial_exchange_rates", logf, func(context.Context) {
		var closed *struct{ rows int }
		_ = closed.rows // the nil dereference a query hits once PocketBase released the database
	})
	job()
	if len(logged) != 1 || !strings.Contains(logged[0], "initial_exchange_rates") {
		t.Fatalf("panic was not contained and reported: %q", logged)
	}
}

func TestGuardBackgroundSkipsWorkAfterTermination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ran := false
	guardBackground(ctx, "subflow_exchange_rates", t.Logf, func(context.Context) { ran = true })()
	if ran {
		t.Fatal("background work started after the application terminated")
	}
}

func readyManager(t *testing.T) *upgrade.Manager {
	t.Helper()
	manager, err := upgrade.New(filepath.Join(t.TempDir(), "pb_data"))
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Acquire(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	if err := manager.BeginStartup(); err != nil {
		t.Fatal(err)
	}
	return manager
}

func TestServeFailureAfterReadyLeavesUpgradeStateReady(t *testing.T) {
	manager := readyManager(t)
	if err := manager.Complete(); err != nil {
		t.Fatal(err)
	}
	// The exact error beta.1 mislabelled: an unrelated path that mentions "backups".
	serveErr := errors.New("open /app/pb_data/.subflow/backups/20261113T095935.203701863Z/manifest.json: no such file or directory")
	if code := recordServeFailure(manager, true, serveErr, t.Logf); code != "serve_failed" {
		t.Fatalf("post-ready failure code = %q", code)
	}
	state, err := manager.ReadState()
	if err != nil {
		t.Fatal(err)
	}
	if state.State != upgrade.StatusReady || !state.Ready || state.ProblemCode != "" {
		t.Fatalf("a serve error after readiness locked the next start: %#v", state)
	}
}

func TestServeFailureBeforeReadyStillLocksUpgrade(t *testing.T) {
	manager := readyManager(t)
	if err := manager.SetPhase(upgrade.StatusMigrating); err != nil {
		t.Fatal(err)
	}
	serveErr := fmt.Errorf("%w: schema change rejected", upgrade.ErrMigrationFailed)
	if code := recordServeFailure(manager, false, serveErr, t.Logf); code != "migration_failed" {
		t.Fatalf("pre-ready failure code = %q", code)
	}
	state, err := manager.ReadState()
	if err != nil {
		t.Fatal(err)
	}
	if state.State != upgrade.StatusFailed || state.ProblemCode != "migration_failed" || state.FailurePhase != upgrade.StatusMigrating {
		t.Fatalf("pre-ready failure was not recorded: %#v", state)
	}
}

func TestProblemCodeIgnoresWordsInUnrelatedMessages(t *testing.T) {
	cases := map[string]error{
		"upgrade_failed":    errors.New("open /app/pb_data/.subflow/backups/x/manifest.json: no such file or directory"),
		"backup_failed":     fmt.Errorf("%w: copy failed", upgrade.ErrBackupFailed),
		"audit_unavailable": fmt.Errorf("%w: event log is read-only", upgrade.ErrAuditUnavailable),
		"migration_failed":  fmt.Errorf("%w: precondition failed", upgrade.ErrMigrationFailed),
		// Disk exhaustion during the backup keeps its more specific code.
		"insufficient_disk_space": fmt.Errorf("%w: %w", upgrade.ErrBackupFailed, errors.New("insufficient_disk_space: need 10 bytes")),
	}
	for want, err := range cases {
		if got := problemCode(err); got != want {
			t.Errorf("problemCode(%v) = %q, want %q", err, got, want)
		}
	}
}
