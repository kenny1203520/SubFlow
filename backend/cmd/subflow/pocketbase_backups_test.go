package main

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"ganigeorgiev.com/pocketbase/pocketbase/core"
	"subflow/internal/upgrade"
)

func TestPocketBaseBackupHookKeepsTLSAndExcludesRecovery(t *testing.T) {
	dir := t.TempDir()
	m, err := upgrade.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Acquire(); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: dir})
	registerPocketBaseBackups(app, m)
	if err = app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	defer app.ClearBootstrap()
	for _, name := range []string{".subflow/tls/cert.pem", ".subflow/backups/old/data.db", ".subflow-recovery/old/data.db", ".subflow/restore-failed-test/current/data.db", "storage/attachment.txt"} {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err = app.CreateBackup(context.Background(), "test.zip"); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(filepath.Join(dir, "backups", "test.zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	files := map[string]bool{}
	for _, entry := range archive.File {
		files[entry.Name] = true
		if upgrade.ExcludedBackupPath(entry.Name) {
			t.Errorf("included work file %s", entry.Name)
		}
	}
	for _, required := range []string{"data.db", "auxiliary.db", "storage/attachment.txt", ".subflow/tls/cert.pem"} {
		if !files[required] {
			t.Errorf("missing %s", required)
		}
	}
}
