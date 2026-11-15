package upgrade

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestBackupExcludesBackupHistoryAndPreservesTLS(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"backups/daily.zip", "backups/yesterday.zip", ".pb_temp_to_delete/pb_backup_abcdef", ".subflow/backups/snapshot/data.db", ".subflow/restore-failed-one/data.db", ".subflow/tls/cert.pem", "storage/backups/attachment", "data.db"} {
		writeFixture(t, filepath.Join(dir, filepath.FromSlash(name)), "test")
	}
	m, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := m.CreateBackup("0.2.0-beta.1", "v0.41.0")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]bool{}
	for _, item := range snapshot.Files {
		files[item.Path] = true
	}
	for _, name := range []string{".subflow/tls/cert.pem", "storage/backups/attachment", "data.db"} {
		if !files[name] {
			t.Errorf("missing %s", name)
		}
	}
	if len(files) != 3 {
		t.Errorf("unexpected files: %v", files)
	}
}

func TestSuccessfulCleanupResumesAndPreservesPendingSnapshot(t *testing.T) {
	dir := t.TempDir()
	m, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Acquire(); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(dir, "data.db"), "test")
	backup, err := m.CreateBackup("0.2.0-beta.1", "v0.41.0")
	if err != nil {
		t.Fatal(err)
	}
	pending, err := m.CreateBackup("0.2.0-beta.1", "v0.41.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Save(State{State: StatusReady, Ready: true, CleanupIDs: []string{backup.ID}}); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if err := m.Acquire(); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if state := m.Status(); state.BackupID != "" || len(state.CleanupIDs) != 0 || !state.Ready {
		t.Fatalf("unexpected state: %#v", state)
	}
	if _, err := os.Stat(filepath.Join(m.recoveryDirectory(), backup.ID)); !os.IsNotExist(err) {
		t.Fatalf("successful snapshot still present: %v", err)
	}
	if err := VerifyBackup(filepath.Join(m.recoveryDirectory(), pending.ID)); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyPocketBaseZIPRestore(t *testing.T) {
	dir := t.TempDir()
	m, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(source, "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE proof (value TEXT); INSERT INTO proof VALUES ('restored');"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(source, "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(source, "daily.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	for name, value := range map[string][]byte{"data.db": data, "storage/attachment.txt": []byte("attachment"), ".subflow/backups/old/data.db": []byte("nested")} {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(value); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(dir, "backups", "another.zip"), "untouched")
	writeFixture(t, filepath.Join(dir, ".subflow", "tls", "cert.pem"), "kept")
	if _, err := m.RestorePocketBaseZIP(path, "0.2.0-beta.1", "v0.41.0"); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	for name, want := range map[string]string{"storage/attachment.txt": "attachment", "backups/another.zip": "untouched", ".subflow/tls/cert.pem": "kept"} {
		contents, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil || string(contents) != want {
			t.Fatalf("%s = %q, %v", name, contents, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".subflow", "backups", "old")); !os.IsNotExist(err) {
		t.Fatalf("nested snapshot restored: %v", err)
	}
}

func writeSQLiteFixture(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE proof (value TEXT)"); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestMidWriteDiskFullRemovesStagingAndLeavesData(t *testing.T) {
	dir := t.TempDir()
	m, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	writeSQLiteFixture(t, filepath.Join(dir, "data.db"))
	if err := m.Acquire(); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Save(State{AppVersion: "0.2.0-beta.1", State: StatusReady, Ready: true}); err != nil {
		t.Fatal(err)
	}
	m.copySnapshot = func(_, stage string) ([]BackupFile, error) {
		writeFixture(t, filepath.Join(stage, "partial"), "partial")
		return nil, &os.PathError{Op: "write", Path: "partial", Err: syscall.ENOSPC}
	}
	if err := m.Prepare("0.2.0-beta.2", "0.41.0", false); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("expected ENOSPC, got %v", err)
	}
	entries, err := os.ReadDir(m.recoveryDirectory())
	if err != nil || len(entries) != 0 {
		t.Fatalf("staged files remain: %v %v", entries, err)
	}
	if err := checkSQLiteFiles(dir); err != nil {
		t.Fatal(err)
	}
}

func TestBackupManifestRejectsUnsafePaths(t *testing.T) {
	for _, path := range []string{"../data.db", "storage/../data.db", "/data.db", "C:/data.db", "storage//file", "storage/" + string(rune(92)) + "file"} {
		if _, err := manifestPath(path); err == nil {
			t.Errorf("accepted %q", path)
		}
	}
}

func TestZIPInspectionRejectsFileDirectoryCollision(t *testing.T) {
	var contents bytes.Buffer
	archive := zip.NewWriter(&contents)
	for _, path := range []string{"data.db", "storage", "storage/file"} {
		if _, err := archive.Create(path); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := inspectPocketBaseZIP(bytes.NewReader(contents.Bytes()), int64(contents.Len())); err == nil {
		t.Fatal("accepted colliding archive paths")
	}
}

func TestCompleteRetainsSnapshotUntilAuditSucceeds(t *testing.T) {
	dir := t.TempDir()
	m, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	writeSQLiteFixture(t, filepath.Join(dir, "data.db"))
	if err := m.Acquire(); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err := m.Save(State{State: StatusReady, Ready: true, AppVersion: "0.2.0-beta.1", PocketBaseVersion: "0.41.0"}); err != nil {
		t.Fatal(err)
	}
	if err := m.Prepare("0.2.0-beta.2", "0.41.0", false); err != nil {
		t.Fatal(err)
	}
	id := m.Status().BackupID
	if err := m.SetPhase(StatusVerifying); err != nil {
		t.Fatal(err)
	}
	auditFailure := errors.New("audit failure")
	if err := m.Complete(func() error { return auditFailure }); !errors.Is(err, auditFailure) {
		t.Fatalf("expected audit failure, got %v", err)
	}
	if m.Status().Ready || m.Status().BackupID != id {
		t.Fatalf("recovery reference lost: %#v", m.Status())
	}
	if err := VerifyBackup(filepath.Join(m.recoveryDirectory(), id)); err != nil {
		t.Fatal(err)
	}
	if err := m.Prepare("0.2.0-beta.2", "0.41.0", true); err != nil {
		t.Fatal(err)
	}
	if err := m.Complete(func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	status, backups, err := m.Snapshot()
	if err != nil || !status.Ready || status.BackupID != "" || len(backups) != 0 {
		t.Fatalf("unexpected final snapshot: %v %v %v", status, backups, err)
	}
	events, err := os.ReadFile(filepath.Join(m.Dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(events, []byte("upgrade.completed")) != 1 {
		t.Fatalf("completion event duplicated: %s", events)
	}
}
