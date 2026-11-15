package upgrade

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
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
	backupRoot, err := m.findBackup(id)
	if err != nil {
		return BackupManifest{}, err
	}
	manifest, err = readManifest(backupRoot)
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
	return m.restoreVerifiedTree(backupRoot, manifest, &journalWritten)
}

func (m *Manager) restoreVerifiedTree(backupRoot string, manifest BackupManifest, journalWritten *bool) (BackupManifest, error) {
	id := manifest.ID
	manifest.Files = restoreFiles(manifest.Files)
	required, err := backupSize(m.DataDir)
	if err != nil {
		return BackupManifest{}, err
	}
	for _, item := range manifest.Files {
		if item.Size < 0 || item.Size > math.MaxInt64-required-4096 {
			return BackupManifest{}, fmt.Errorf("restore size overflow")
		}
		required += item.Size + 4096
	}
	if err := checkSpace(m.Dir, required); err != nil {
		return BackupManifest{}, err
	}
	stage := filepath.Join(m.Dir, "restore-stage-"+id)
	failed := filepath.Join(m.Dir, "restore-failed-"+id)
	if err := os.Mkdir(stage, 0o700); err != nil {
		return BackupManifest{}, err
	}
	defer func() {
		if !*journalWritten {
			_ = safeRemoveTree(m.Dir, filepath.Base(stage))
		}
	}()
	if err := os.Mkdir(failed, 0o700); err != nil {
		return BackupManifest{}, err
	}
	defer func() {
		if !*journalWritten {
			_ = safeRemoveTree(m.Dir, filepath.Base(failed))
		}
	}()
	if _, err := copyDataTree(m.DataDir, filepath.Join(failed, "current")); err != nil {
		return BackupManifest{}, fmt.Errorf("failed to preserve current data: %w", err)
	}
	for _, item := range manifest.Files {
		relative, err := safeBackupPath(item.Path)
		if err != nil {
			return BackupManifest{}, err
		}
		if err := copyVerifiedFile(backupRoot, relative, filepath.Join(stage, relative)); err != nil {
			return BackupManifest{}, err
		}
	}
	journal := restoreJournal{BackupID: id, Stage: filepath.Base(stage), Failed: filepath.Base(failed), Files: manifest.Files}
	if err := writeRestoreJournal(filepath.Join(m.Dir, "restore.json"), journal); err != nil {
		return BackupManifest{}, err
	}
	*journalWritten = true
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
	if err := safeRemoveTree(m.Dir, filepath.Base(stage)); err != nil {
		return BackupManifest{}, err
	}
	return manifest, nil
}

func (m *Manager) recoverRestoreJournal() error {
	path := filepath.Join(m.Dir, "restore.json")
	file, err := regularBackupFile(m.Dir, filepath.Base(path))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	data, err := io.ReadAll(file)
	if err := errors.Join(err, file.Close()); err != nil {
		return err
	}
	var journal restoreJournal
	if err := json.Unmarshal(data, &journal); err != nil {
		return fmt.Errorf("restore journal is corrupt; data is preserved for manual recovery")
	}
	if !backupID(journal.BackupID) || journal.Stage != "restore-stage-"+journal.BackupID || journal.Failed != "restore-failed-"+journal.BackupID {
		return fmt.Errorf("restore journal contains an unsafe path")
	}
	root, err := m.findBackup(journal.BackupID)
	if err != nil {
		return err
	}
	manifest, err := readManifest(root)
	if err != nil || manifest.ID != journal.BackupID || verifyBackup(root, manifest) != nil {
		return fmt.Errorf("restore journal backup no longer verifies; preserved data requires manual recovery")
	}
	manifest.Files = restoreFiles(manifest.Files)
	journal.Files = restoreFiles(journal.Files)
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
	if err := safeRemoveTree(m.Dir, journal.Stage); err != nil {
		return err
	}
	return ErrRestoreRecovered
}

func (m *Manager) applyRestoreJournal(journal restoreJournal) error {
	stage, failed := filepath.Join(m.Dir, journal.Stage), filepath.Join(m.Dir, journal.Failed)
	if err := realDirectory(stage); err != nil {
		return err
	}
	if err := realDirectory(failed); err != nil {
		return err
	}
	live, err := os.OpenRoot(m.DataDir)
	if err != nil {
		return err
	}
	defer live.Close()
	for _, item := range journal.Files {
		if _, err := safeBackupPath(item.Path); err != nil {
			return err
		}
		if err := validateRestoreDestination(live, item.Path); err != nil {
			return err
		}
	}
	expected := make(map[string]bool, len(journal.Files))
	hasTLS := false
	for _, item := range journal.Files {
		relative, err := safeBackupPath(item.Path)
		if err != nil {
			return err
		}
		expected[filepath.Clean(relative)] = true
		hasTLS = hasTLS || strings.HasPrefix(filepath.ToSlash(relative), ".subflow/tls/")
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
		if err := live.MkdirAll(filepath.Dir(relative), 0o700); err != nil {
			return err
		}
		if _, err := os.Stat(destination); err == nil {
			if err := live.Remove(relative); err != nil {
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
	err = walkBackupTree(m.DataDir, func(relative string, entry os.DirEntry) error {
		if entry.IsDir() || !restorablePath(relative) || (!hasTLS && strings.HasPrefix(filepath.ToSlash(relative), ".subflow/tls/")) {
			return nil
		}
		current := filepath.Join(m.DataDir, relative)
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
	file, err := os.CreateTemp(filepath.Dir(path), ".restore-journal-*.tmp")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	_, err = file.Write(append(data, '\n'))
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err := replaceFile(temporary, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func restorablePath(path string) bool {
	path = filepath.ToSlash(path)
	return !ExcludedBackupPath(path) && path != ".subflow/state.json" && path != ".subflow/events.jsonl"
}

func restoreFiles(files []BackupFile) []BackupFile {
	var result []BackupFile
	for _, file := range files {
		if restorablePath(file.Path) {
			result = append(result, file)
		}
	}
	return result
}

func safeBackupPath(value string) (string, error) {
	if !restorablePath(value) {
		return "", fmt.Errorf("excluded restore file")
	}
	return manifestPath(value)
}

func validateRestoreDestination(root *os.Root, value string) error {
	parts := strings.Split(value, "/")
	for i := range parts {
		info, err := root.Lstat(filepath.Join(parts[:i+1]...))
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if (i < len(parts)-1 && !info.IsDir()) || (i == len(parts)-1 && !info.Mode().IsRegular()) {
			return fmt.Errorf("unsafe restore destination")
		}
	}
	return nil
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
	file, err := regularBackupFile(root, path)
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
