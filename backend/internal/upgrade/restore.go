package upgrade

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type restoreJournal struct {
	BackupID string       `json:"backupId"`
	Stage    string       `json:"stage"`
	Failed   string       `json:"failed"`
	Files    []BackupFile `json:"files"`
}

func (m *Manager) RestoreBackup(id string) (manifest BackupManifest, resultErr error) {
	if err := m.Acquire(); err != nil {
		return BackupManifest{}, err
	}
	journalWritten := false
	defer func() {
		if resultErr == nil {
			return
		}
		if auditErr := m.AppendEvent("upgrade.restore.failed", "failure", "restore_failed"); auditErr != nil {
			resultErr = fmt.Errorf("%v; durable restore failure audit unavailable: %w", resultErr, auditErr)
		}
		if journalWritten {
			m.state.State, m.state.Phase, m.state.Ready = StatusFailed, StatusFailed, false
			m.state.ProblemCode = "restore_failed"
			if stateErr := m.Save(m.state); stateErr != nil {
				resultErr = fmt.Errorf("%v; could not persist restore failure state: %w", resultErr, stateErr)
			}
		}
	}()
	if !backupID(id) {
		return BackupManifest{}, fmt.Errorf("invalid backup id")
	}
	backupRoot := filepath.Join(m.Dir, "backups", id)
	manifest, err := readManifest(backupRoot)
	if err != nil {
		return BackupManifest{}, fmt.Errorf("backup is unavailable")
	}
	if manifest.ID != id {
		return BackupManifest{}, fmt.Errorf("backup id does not match manifest")
	}
	if err := verifyBackup(backupRoot, manifest); err != nil {
		return BackupManifest{}, fmt.Errorf("backup verification failed")
	}
	if err := m.AppendEvent("upgrade.restore.started", "success", "verified backup "+id); err != nil {
		return BackupManifest{}, fmt.Errorf("durable audit log unavailable: %w", err)
	}
	stage := filepath.Join(m.Dir, "restore-stage-"+id)
	failed := filepath.Join(m.Dir, "restore-failed-"+id)
	if err := os.MkdirAll(stage, 0o700); err != nil {
		return BackupManifest{}, err
	}
	if err := os.MkdirAll(failed, 0o700); err != nil {
		return BackupManifest{}, err
	}
	if _, err := copyDataTree(m.DataDir, filepath.Join(failed, "current")); err != nil {
		return BackupManifest{}, fmt.Errorf("failed to preserve current data: %w", err)
	}
	for _, item := range manifest.Files {
		relative, err := safeBackupPath(item.Path)
		if err != nil {
			return BackupManifest{}, err
		}
		if err := copyFile(filepath.Join(backupRoot, relative), filepath.Join(stage, relative)); err != nil {
			return BackupManifest{}, err
		}
	}
	journal := restoreJournal{BackupID: id, Stage: filepath.Base(stage), Failed: filepath.Base(failed), Files: manifest.Files}
	if err := writeRestoreJournal(filepath.Join(m.Dir, "restore.json"), journal); err != nil {
		return BackupManifest{}, err
	}
	journalWritten = true
	if err := m.applyRestoreJournal(journal); err != nil {
		return BackupManifest{}, err
	}
	m.state = State{State: StatusFailed, Phase: StatusFailed, Ready: false, ProblemCode: "restore_requires_matching_binary", AppVersion: manifest.AppVersion, PocketBaseVersion: manifest.PocketBaseVersion, SchemaVersion: 1, BackupID: manifest.ID, Source: "restored", History: []MigrationEntry{}}
	if err := m.Save(m.state); err != nil {
		return BackupManifest{}, err
	}
	if err := m.AppendEvent("upgrade.restore.completed", "success", "restore completed; matching application binary required"); err != nil {
		return BackupManifest{}, fmt.Errorf("restore completed but durable audit log failed: %w", err)
	}
	if err := os.Remove(filepath.Join(m.Dir, "restore.json")); err != nil {
		return BackupManifest{}, err
	}
	if err := os.RemoveAll(stage); err != nil {
		return BackupManifest{}, err
	}
	return manifest, nil
}

func (m *Manager) recoverRestoreJournal() error {
	path := filepath.Join(m.Dir, "restore.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var journal restoreJournal
	if err := json.Unmarshal(data, &journal); err != nil {
		return fmt.Errorf("restore journal is corrupt; data is preserved for manual recovery")
	}
	if !backupID(journal.BackupID) || filepath.Base(journal.Stage) != journal.Stage || filepath.Base(journal.Failed) != journal.Failed {
		return fmt.Errorf("restore journal contains an unsafe path")
	}
	manifest, err := readManifest(filepath.Join(m.Dir, "backups", journal.BackupID))
	if err != nil || manifest.ID != journal.BackupID || verifyBackup(filepath.Join(m.Dir, "backups", journal.BackupID), manifest) != nil {
		return fmt.Errorf("restore journal backup no longer verifies; preserved data requires manual recovery")
	}
	if len(manifest.Files) != len(journal.Files) {
		return fmt.Errorf("restore journal does not match its verified backup")
	}
	verifiedFiles := make(map[string]BackupFile, len(manifest.Files))
	for _, item := range manifest.Files {
		verifiedFiles[item.Path] = item
	}
	for _, item := range journal.Files {
		if verifiedFiles[item.Path] != item {
			return fmt.Errorf("restore journal does not match its verified backup")
		}
	}
	if err := m.applyRestoreJournal(journal); err != nil {
		return err
	}
	m.state = State{State: StatusFailed, Phase: StatusFailed, Ready: false, ProblemCode: "restore_requires_matching_binary", AppVersion: manifest.AppVersion, PocketBaseVersion: manifest.PocketBaseVersion, SchemaVersion: 1, BackupID: manifest.ID, Source: "restored", History: []MigrationEntry{}}
	if err := m.Save(m.state); err != nil {
		return err
	}
	if err := m.AppendEvent("upgrade.restore.recovered", "success", "interrupted restore completed; matching binary required"); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(m.Dir, journal.Stage)); err != nil {
		return err
	}
	return ErrRestoreRecovered
}

func (m *Manager) applyRestoreJournal(journal restoreJournal) error {
	stage, failed := filepath.Join(m.Dir, journal.Stage), filepath.Join(m.Dir, journal.Failed)
	expected := make(map[string]bool, len(journal.Files))
	for _, item := range journal.Files {
		relative, err := safeBackupPath(item.Path)
		if err != nil {
			return err
		}
		expected[filepath.Clean(relative)] = true
		source := filepath.Join(stage, relative)
		destination := filepath.Join(m.DataDir, relative)
		if _, err := os.Stat(source); os.IsNotExist(err) {
			if err := verifyBackupFile(filepath.Dir(destination), BackupFile{Path: filepath.Base(destination), Size: item.Size, SHA256: item.SHA256}); err != nil {
				return fmt.Errorf("restore interrupted and a staged file is missing")
			}
			continue
		} else if err != nil {
			return err
		}
		if err := verifyBackupFile(stage, item); err != nil {
			return fmt.Errorf("staged restore data failed verification")
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return err
		}
		if _, err := os.Stat(destination); err == nil {
			if err := os.Remove(destination); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := os.Rename(source, destination); err != nil {
			if err := copyFile(source, destination); err != nil {
				return err
			}
			if err := os.Remove(source); err != nil {
				return err
			}
		}
	}
	// Move files absent from the snapshot out of the live data tree. This makes
	// the restored tree match the verified file manifest while retaining the
	// pre-restore scene in .subflow/restore-failed-<id>/current.
	err := filepath.WalkDir(m.DataDir, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(m.DataDir, current)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if entry.Name() == StateDirectory && entry.IsDir() {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("restored data tree contains an unsupported file type")
		}
		key := filepath.Clean(relative)
		if expected[key] {
			return nil
		}
		if err := os.MkdirAll(filepath.Join(failed, "unlisted", filepath.Dir(relative)), 0o700); err != nil {
			return err
		}
		return os.Rename(current, filepath.Join(failed, "unlisted", relative))
	})
	if err != nil {
		return err
	}
	for _, item := range journal.Files {
		relative, err := safeBackupPath(item.Path)
		if err != nil {
			return err
		}
		if err := verifyFile(filepath.Join(m.DataDir, relative), item); err != nil {
			return err
		}
	}
	return syncDirectory(m.DataDir)
}

func writeRestoreJournal(path string, journal restoreJournal) error {
	data, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return err
	}
	temporary := path + ".tmp"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(append(data, '\n'))
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := replaceFile(temporary, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func safeBackupPath(value string) (string, error) {
	if strings.TrimSpace(value) != value || strings.Contains(value, `\`) {
		return "", fmt.Errorf("backup manifest has an unsafe file path")
	}
	clean := filepath.Clean(filepath.FromSlash(value))
	if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == StateDirectory || strings.HasPrefix(clean, StateDirectory+string(filepath.Separator)) {
		return "", fmt.Errorf("backup manifest has an unsafe file path")
	}
	return clean, nil
}

func copyFile(source, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	if copyErr == nil {
		copyErr = output.Sync()
	}
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func verifyFile(path string, item BackupFile) error {
	return verifyBackupFile(filepath.Dir(path), BackupFile{Path: filepath.Base(path), Size: item.Size, SHA256: item.SHA256})
}

func verifyBackupFile(root string, item BackupFile) error {
	path, err := safeBackupPath(item.Path)
	if err != nil {
		return err
	}
	info, err := os.Lstat(filepath.Join(root, path))
	if err != nil || !info.Mode().IsRegular() || info.Size() != item.Size {
		return fmt.Errorf("restore verification failed")
	}
	file, err := os.Open(filepath.Join(root, path))
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if hex.EncodeToString(hash.Sum(nil)) != item.SHA256 {
		return fmt.Errorf("restore verification failed")
	}
	return nil
}
