package upgrade

import (
	"bufio"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ganigeorgiev.com/pocketbase/dbx"
	"ganigeorgiev.com/pocketbase/pocketbase/core"
)

func (m *Manager) recoveryDirectory() string { return filepath.Join(m.DataDir, RecoveryDirectory) }

func (m *Manager) backupParents() []string {
	return []string{m.recoveryDirectory(), filepath.Join(m.Dir, "backups")}
}

func ensureRealDirectory(path string) error {
	if err := os.Mkdir(path, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	return realDirectory(path)
}

func safeDirectoryEntries(path string) ([]os.DirEntry, error) {
	if err := realDirectory(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return os.ReadDir(path)
}

func (m *Manager) findBackup(id string) (string, error) {
	if !backupID(id) {
		return "", fmt.Errorf("invalid backup id")
	}
	if err := realDirectory(m.Dir); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	var found string
	for _, parent := range m.backupParents() {
		if err := realDirectory(parent); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return "", err
		}
		candidate := filepath.Join(parent, id)
		if err := realDirectory(candidate); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return "", err
		}
		if found != "" {
			return "", fmt.Errorf("duplicate backup id")
		}
		found = candidate
	}
	if found == "" {
		return "", os.ErrNotExist
	}
	return found, nil
}

func writeSyncedFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	if err == nil {
		err = file.Sync()
	}
	return errors.Join(err, file.Close())
}

func manifestPath(name string) (string, error) {
	path := filepath.FromSlash(name)
	if name == "" || strings.ContainsAny(name, "\\:\x00") || !filepath.IsLocal(path) ||
		filepath.ToSlash(filepath.Clean(path)) != name || name == "." || name == "manifest.json" {
		return "", fmt.Errorf("unsafe backup path")
	}
	return path, nil
}

func checkSQLiteFiles(root string) error {
	for _, name := range []string{"data.db", "auxiliary.db"} {
		file, err := regularBackupFile(root, name)
		if errors.Is(err, os.ErrNotExist) && name == "auxiliary.db" {
			continue
		}
		if err != nil {
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
		path := filepath.ToSlash(filepath.Join(root, name))
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		uri := (&url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro"}).String()
		if err := checkSQLite(uri, name); err != nil {
			return err
		}
		// A read-only WAL connection can create shared-memory index files.
		_ = os.Remove(filepath.Join(root, name+"-shm"))
	}
	return nil
}

func checkSQLite(uri, name string) error {
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		return err
	}
	defer db.Close()
	var result string
	if err := db.QueryRow("PRAGMA quick_check").Scan(&result); err != nil {
		return fmt.Errorf("verify %s: %w", name, err)
	}
	if result != "ok" {
		return fmt.Errorf("SQLite consistency check failed: %s", name)
	}
	return nil
}

func (m *Manager) cleanupWarning(err error) {
	log.Printf("SubFlow backup cleanup pending: %v", err)
}

func (m *Manager) recoverBackupWork() {
	state, err := m.readState()
	if err != nil {
		m.cleanupWarning(err)
		return
	}
	m.state = state
	entries, err := safeDirectoryEntries(m.recoveryDirectory())
	if err != nil {
		m.cleanupWarning(err)
		return
	}
	for _, entry := range entries {
		id, ok := strings.CutPrefix(entry.Name(), ".creating-")
		if !ok || !entry.IsDir() || !backupID(id) {
			continue
		}
		root := filepath.Join(m.recoveryDirectory(), entry.Name())
		// A complete staged snapshot may be the reference persisted before a crash.
		if manifest, err := readManifest(root); state.BackupID == id && err == nil && manifest.ID == id && verifyBackup(root, manifest) == nil && checkSQLiteFiles(root) == nil {
			if err := os.Rename(root, filepath.Join(m.recoveryDirectory(), id)); err != nil {
				m.cleanupWarning(err)
			} else if err := syncDirectory(m.recoveryDirectory()); err != nil {
				m.cleanupWarning(err)
			}
			continue
		}
		if err := safeRemoveTree(m.recoveryDirectory(), entry.Name()); err != nil {
			m.cleanupWarning(err)
		}
	}
	if state.State == StatusReady && state.Ready && len(state.CleanupIDs) != 0 {
		m.cleanSuccessfulBackups()
	}
}

func safeRemoveTree(parent, name string) error {
	if !filepath.IsLocal(name) || filepath.Base(name) != name {
		return fmt.Errorf("unsafe cleanup target")
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		return err
	}
	defer root.Close()
	if info, err := root.Lstat(name); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	} else if !info.IsDir() {
		return fmt.Errorf("unsafe cleanup directory %s", name)
	}
	if err := root.RemoveAll(name); err != nil {
		return err
	}
	return syncDirectory(parent)
}

func (m *Manager) cleanSuccessfulBackups() {
	if m.lock == nil || !m.state.Ready {
		return
	}
	if _, err := os.Lstat(filepath.Join(m.Dir, "restore.json")); !errors.Is(err, os.ErrNotExist) {
		return
	}
	pending := append([]string(nil), m.state.CleanupIDs...)
	// Legacy success is established by its manifest and a verified live startup.
	for _, parent := range m.backupParents() {
		entries, err := safeDirectoryEntries(parent)
		if err != nil {
			m.cleanupWarning(err)
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() || !backupID(entry.Name()) {
				continue
			}
			manifest, err := readManifest(filepath.Join(parent, entry.Name()))
			if err == nil && manifest.ID == entry.Name() && manifest.Status == "successful" && validLegacyCleanupManifest(manifest) {
				pending = append(pending, entry.Name())
			}
		}
	}
	if len(pending) == 0 {
		return
	}
	seen := map[string]bool{}
	m.state.CleanupIDs = nil
	for _, id := range pending {
		if seen[id] {
			continue
		}
		seen[id] = true
		if !backupID(id) || id == m.state.BackupID {
			continue
		}
		m.state.CleanupIDs = append(m.state.CleanupIDs, id)
	}
	if err := m.Save(m.state); err != nil {
		m.cleanupWarning(err)
		return
	}
	retry := []string{}
	for _, id := range m.state.CleanupIDs {
		root, err := m.findBackup(id)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err == nil {
			err = safeRemoveTree(filepath.Dir(root), id)
		}
		if err != nil {
			m.cleanupWarning(err)
			retry = append(retry, id)
		}
	}
	m.state.CleanupIDs = retry
	if err := m.Save(m.state); err != nil {
		m.cleanupWarning(err)
	}
}

func copyVerifiedFile(root, relative, destination string) error {
	source, err := regularBackupFile(root, relative)
	if err != nil {
		return err
	}
	defer source.Close()
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, source)
	if err == nil {
		err = out.Sync()
	}
	return errors.Join(err, out.Close())
}

func VerifyLiveDatabases(app core.App) error {
	for _, db := range []interface{ NewQuery(string) *dbx.Query }{app.DB(), app.AuxDB()} {
		var result string
		if err := db.NewQuery("PRAGMA quick_check").Row(&result); err != nil {
			return err
		}
		if result != "ok" {
			return fmt.Errorf("SQLite consistency check failed")
		}
	}
	return nil
}

func (m *Manager) PocketBaseBackupExclusions() ([]string, error) {
	if !m.HasLock() {
		return nil, fmt.Errorf("data directory lock is required")
	}
	entries, err := safeDirectoryEntries(m.Dir)
	if err != nil {
		return nil, err
	}
	exclusions := []string{RecoveryDirectory}
	for _, entry := range entries {
		name := StateDirectory + "/" + entry.Name()
		if ExcludedBackupPath(name) {
			exclusions = append(exclusions, name)
		}
	}
	return exclusions, nil
}

func (m *Manager) CheckPocketBaseBackupSpace(remote bool) error {
	bytes, err := backupSize(m.DataDir)
	if err != nil {
		return err
	}
	// A concurrently written SQLite snapshot and the ZIP can coexist.
	if bytes > (1<<63-1)/3 {
		return fmt.Errorf("backup size overflow")
	}
	if err := checkSpace(m.DataDir, 2*bytes); err != nil {
		return err
	}
	if remote {
		return nil
	}
	parent := filepath.Join(m.DataDir, "backups")
	if err := ensureRealDirectory(parent); err != nil {
		return err
	}
	return checkSpace(parent, 3*bytes)
}

func (m *Manager) CheckPocketBaseRestoreSpace(total, compressed int64, remote bool) error {
	if remote {
		if compressed > (1<<63)-1-total {
			return fmt.Errorf("restore size overflow")
		}
		total += compressed
	}
	return checkSpace(m.DataDir, total)
}

func (m *Manager) AllowNativePocketBaseRestore() error {
	state, err := m.readState()
	if err != nil {
		return err
	}
	if !state.Ready || state.BackupID != "" {
		return fmt.Errorf("use offline restore while a recovery backup is still active")
	}
	entries, err := safeDirectoryEntries(filepath.Join(m.Dir, "backups"))
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("legacy recovery data exists; use subflow upgrade restore --pocketbase-zip")
	}
	return nil
}

func (m *Manager) appendCompletionEvent() error {
	summary := fmt.Sprintf("upgrade verified app=%s pocketbase=%s snapshot=%s", m.state.AppVersion, m.state.PocketBaseVersion, m.state.BackupID)
	if m.state.BackupID == "" {
		return m.AppendEvent("upgrade.completed", "success", summary)
	}
	// Stable across retries of the same verified recovery snapshot.
	id := fmt.Sprintf("%x", sha256.Sum256([]byte("upgrade.completed:"+m.state.BackupID+":"+m.state.AppVersion)))[:40]
	path := filepath.Join(m.Dir, "events.jsonl")
	file, err := os.Open(path)
	if err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var event Event
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				_ = file.Close()
				return err
			}
			if event.ID == id {
				return file.Close()
			}
		}
		if err := errors.Join(scanner.Err(), file.Close()); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return m.appendDurableEvent(Event{ID: id, Action: "upgrade.completed", Outcome: "success", CreatedAt: time.Now().UTC(), Summary: summary})
}

func validLegacyCleanupManifest(manifest BackupManifest) bool {
	seen := map[string]bool{}
	for _, item := range manifest.Files {
		if _, err := manifestPath(item.Path); err != nil || seen[strings.ToLower(item.Path)] || item.Size < 0 || len(item.SHA256) != 64 {
			return false
		}
		seen[strings.ToLower(item.Path)] = true
	}
	return seen["data.db"]
}
