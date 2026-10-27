package upgrade

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestFailureRetainsLastStartupPhase(t *testing.T) {
	manager, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.BeginStartup(); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetPhase(StatusMigrating); err != nil {
		t.Fatal(err)
	}
	if err := manager.Fail("upgrade_failed"); err != nil {
		t.Fatal(err)
	}
	state, err := manager.ReadState()
	if err != nil {
		t.Fatal(err)
	}
	if state.State != StatusFailed || state.FailurePhase != StatusMigrating || state.Ready {
		t.Fatalf("failed state did not retain the startup phase: %#v", state)
	}
}

func TestBackupVerifiesDataAndExcludesUpgradeControlDirectory(t *testing.T) {
	manager, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(manager.DataDir, "data.db"), "sqlite-data")
	writeFixture(t, filepath.Join(manager.DataDir, "storage", "receipt.bin"), "attachment-bytes")
	writeFixture(t, filepath.Join(manager.Dir, "state.json"), `{"secret":"must-not-be-copied"}`)
	manifest, err := manager.CreateBackup("0.2.0-alpha.5", "v0.40.6")
	if err != nil {
		t.Fatal(err)
	}
	if manifest.AppVersion != "0.2.0-alpha.5" || manifest.PocketBaseVersion != "v0.40.6" {
		t.Fatalf("source versions not preserved: %#v", manifest)
	}
	if err := VerifyBackup(filepath.Join(manager.Dir, "backups", manifest.ID)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(manager.Dir, "backups", manifest.ID, StateDirectory, "state.json")); !os.IsNotExist(err) {
		t.Fatalf("control state was copied: %v", err)
	}
	writeFixture(t, filepath.Join(manager.Dir, "backups", manifest.ID, "data.db"), "tampered")
	if err := VerifyBackup(filepath.Join(manager.Dir, "backups", manifest.ID)); err == nil {
		t.Fatal("tampered backup unexpectedly verified")
	}
}

func TestBackupRetentionKeepsThreeSuccessfulAndAllPending(t *testing.T) {
	manager, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(manager.DataDir, "data.db"), "sqlite-data")
	ids := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		manifest, createErr := manager.CreateBackup("legacy", "unknown")
		if createErr != nil {
			t.Fatal(createErr)
		}
		ids = append(ids, manifest.ID)
		if i < 4 {
			if statusErr := manager.SetBackupStatus(manifest.ID, "successful"); statusErr != nil {
				t.Fatal(statusErr)
			}
		}
	}
	if err := manager.PruneBackups(3); err != nil {
		t.Fatal(err)
	}
	backups, err := manager.ListBackups()
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 4 {
		t.Fatalf("expected 3 successful backups and 1 pending backup, got %d", len(backups))
	}
	for _, id := range []string{ids[0]} {
		if _, err := os.Stat(filepath.Join(manager.Dir, "backups", id)); err == nil {
			ordered := make([]string, 0, len(backups))
			for _, backup := range backups {
				ordered = append(ordered, backup.ID+"/"+backup.Status)
			}
			t.Fatalf("older successful backup %s was retained; created=%v ordered=%v", id, ids, ordered)
		}
	}
	if _, err := os.Stat(filepath.Join(manager.Dir, "backups", ids[4])); err != nil {
		t.Fatalf("pending backup was pruned: %v", err)
	}
}

func TestRestoreRestoresOnlyDataAndRecordsRecoveryRequirement(t *testing.T) {
	manager, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(manager.DataDir, "data.db"), "before-upgrade")
	manifest, err := manager.CreateBackup("0.2.0-alpha.5", "v0.40.6")
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(manager.DataDir, "data.db"), "failed-upgrade")
	writeFixture(t, filepath.Join(manager.DataDir, "storage", "new.bin"), "failure-scene")
	if _, err := manager.RestoreBackup(manifest.ID); err != nil {
		t.Fatal(err)
	}
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(manager.DataDir, "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "before-upgrade" {
		t.Fatalf("restored data = %q", data)
	}
	failed, err := os.ReadFile(filepath.Join(manager.Dir, "restore-failed-"+manifest.ID, "current", "storage", "new.bin"))
	if err != nil || string(failed) != "failure-scene" {
		t.Fatalf("failed scene was not preserved: data=%q err=%v", failed, err)
	}
	state, err := manager.ReadState()
	if err != nil {
		t.Fatal(err)
	}
	if state.State != StatusFailed || state.Ready || state.ProblemCode != "restore_requires_matching_binary" {
		t.Fatalf("unsafe restore state: %#v", state)
	}
}

func TestDataDirectoryLockIsExclusiveAndReleased(t *testing.T) {
	directory := t.TempDir()
	first, err := New(directory)
	if err != nil {
		t.Fatal(err)
	}
	second, err := New(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Acquire(); err != nil {
		t.Fatal(err)
	}
	if err := second.Acquire(); err == nil {
		t.Fatal("parallel process acquired the same data directory")
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if err := second.Acquire(); err != nil {
		t.Fatalf("lock was not released: %v", err)
	}
	_ = second.Close()
}

func TestLegacyFingerprintAcceptsKnownPocketBaseDomainAndRejectsUnknown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")
	database, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{"CREATE TABLE _collections (id TEXT)", "CREATE TABLE users (id TEXT)", "CREATE TABLE groups (id TEXT)", "CREATE TABLE expenses (id TEXT)"} {
		if _, err := database.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	if err := CheckLegacyFingerprint(path); err != nil {
		t.Fatalf("known legacy schema rejected: %v", err)
	}
	unknown := filepath.Join(t.TempDir(), "unknown.db")
	writeFixture(t, unknown, "not a sqlite database")
	if err := CheckLegacyFingerprint(unknown); err == nil {
		t.Fatal("unknown schema unexpectedly accepted")
	}
}

func TestSemanticVersionComparisonHandlesAlphaOrdering(t *testing.T) {
	if compareVersions("0.2.0-alpha.5", "0.2.0-alpha.6") >= 0 {
		t.Fatal("alpha.5 must compare below alpha.6")
	}
	if compareVersions("0.2.0-alpha.6", "0.2.0") >= 0 {
		t.Fatal("prerelease must compare below release")
	}
	if compareVersions("0.2.0", "0.1.9") <= 0 {
		t.Fatal("0.2.0 must compare above 0.1.9")
	}
}

func writeFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
