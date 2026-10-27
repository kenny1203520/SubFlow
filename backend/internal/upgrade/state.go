package upgrade

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	StatusChecking  = "checking"
	StatusBackingUp = "backing_up"
	StatusMigrating = "migrating"
	StatusVerifying = "verifying"
	StatusReady     = "ready"
	StatusFailed    = "failed"
	StateDirectory  = ".subflow"
	MaxBackupCount  = 3
)

var ErrUnknownSchema = errors.New("unsupported or unknown PocketBase schema")
var ErrRestoreRecovered = errors.New("an interrupted restore was recovered; start the matching older binary before upgrading again")

type State struct {
	State             string           `json:"state"`
	Phase             string           `json:"phase"`
	Ready             bool             `json:"ready"`
	ProblemCode       string           `json:"problemCode,omitempty"`
	AppVersion        string           `json:"appVersion"`
	PocketBaseVersion string           `json:"pocketBaseVersion"`
	SchemaVersion     int              `json:"schemaVersion"`
	StartedAt         time.Time        `json:"startedAt,omitempty"`
	CompletedAt       time.Time        `json:"completedAt,omitempty"`
	BackupID          string           `json:"backupId,omitempty"`
	Source            string           `json:"source,omitempty"`
	History           []MigrationEntry `json:"history,omitempty"`
}

type MigrationEntry struct {
	ID          string    `json:"id"`
	Checksum    string    `json:"checksum"`
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt,omitempty"`
	Status      string    `json:"status"`
}

type BackupManifest struct {
	ID                string       `json:"id"`
	Status            string       `json:"status"`
	AppVersion        string       `json:"appVersion"`
	PocketBaseVersion string       `json:"pocketBaseVersion"`
	CreatedAt         time.Time    `json:"createdAt"`
	Files             []BackupFile `json:"files"`
}

type BackupFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type Event struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	Outcome   string    `json:"outcome"`
	CreatedAt time.Time `json:"createdAt"`
	Summary   string    `json:"summary,omitempty"`
}

type Manager struct {
	DataDir string
	Dir     string
	state   State
	lock    *Lock
}

func New(dataDir string) (*Manager, error) {
	if strings.TrimSpace(dataDir) == "" {
		return nil, fmt.Errorf("data directory is required")
	}
	absolute, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}
	return &Manager{DataDir: absolute, Dir: filepath.Join(absolute, StateDirectory)}, nil
}

func (m *Manager) Status() State { return m.state }

func (m *Manager) ReadState() (State, error) {
	state, err := m.readState()
	if err != nil && m.state.State == StatusFailed {
		return m.state, nil
	}
	return state, err
}

func (m *Manager) HasLock() bool { return m.lock != nil }

func (m *Manager) SetPhase(phase string) error {
	switch phase {
	case StatusChecking, StatusBackingUp, StatusMigrating, StatusVerifying, StatusReady, StatusFailed:
	default:
		return fmt.Errorf("invalid upgrade phase")
	}
	m.state.State, m.state.Phase = phase, phase
	m.state.Ready = phase == StatusReady
	return m.Save(m.state)
}

// BeginStartup clears a stale ready flag before the maintenance listener can
// answer readiness checks, while preserving the previous successful version
// and timestamp for Prepare's same-version idempotence check.
func (m *Manager) BeginStartup() error {
	state, err := m.readState()
	if err != nil {
		return err
	}
	if state.State == StatusFailed || state.ProblemCode == "restore_requires_matching_binary" {
		m.state = state
		return nil
	}
	if state.State == "" {
		state.History = []MigrationEntry{}
	}
	state.State, state.Phase, state.Ready = StatusChecking, StatusChecking, false
	m.state = state
	return m.Save(m.state)
}

func (m *Manager) Snapshot() (State, []BackupManifest, error) {
	state, err := m.readState()
	if err != nil {
		return State{}, nil, err
	}
	backups, err := m.ListBackups()
	if err != nil {
		return State{}, nil, err
	}
	return state, backups, nil
}

func (m *Manager) Acquire() error {
	if m.lock != nil {
		return nil
	}
	if err := os.MkdirAll(m.DataDir, 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(m.Dir, 0o700); err != nil {
		return err
	}
	lock, err := acquireLock(filepath.Join(m.Dir, "upgrade.lock"))
	if err != nil {
		return fmt.Errorf("data directory is already in use; stop the other SubFlow process before upgrading")
	}
	m.lock = lock
	if err := m.recoverRestoreJournal(); err != nil {
		if errors.Is(err, ErrRestoreRecovered) {
			return err
		}
		m.state, _ = m.readState()
		m.state.State, m.state.Phase, m.state.Ready = StatusFailed, StatusFailed, false
		m.state.ProblemCode = "restore_recovery_failed"
		_ = m.Save(m.state)
		_ = m.AppendEvent("upgrade.restore.recovery_failed", "failure", "restore_recovery_failed")
		return err
	}
	return nil
}

func (m *Manager) AddHistory(entry MigrationEntry) error {
	for i := range m.state.History {
		if m.state.History[i].ID == entry.ID {
			if m.state.History[i].Checksum != entry.Checksum {
				return fmt.Errorf("migration checksum mismatch for %s", entry.ID)
			}
			m.state.History[i] = entry
			return m.Save(m.state)
		}
	}
	m.state.History = append(m.state.History, entry)
	return m.Save(m.state)
}

func (m *Manager) readState() (State, error) {
	data, err := os.ReadFile(filepath.Join(m.Dir, "state.json"))
	if errors.Is(err, os.ErrNotExist) {
		return State{}, nil
	}
	if err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("stored upgrade state is corrupt")
	}
	return state, nil
}

func (m *Manager) Save(state State) error {
	if err := os.MkdirAll(m.Dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(m.Dir, ".state-*.tmp")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if err = file.Chmod(0o600); err == nil {
		_, err = file.Write(append(data, '\n'))
	}
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
	if err := replaceFile(name, filepath.Join(m.Dir, "state.json")); err != nil {
		return err
	}
	return syncDirectory(m.Dir)
}

func (m *Manager) Fail(problemCode string) error {
	m.state.State, m.state.Phase, m.state.Ready = StatusFailed, StatusFailed, false
	m.state.ProblemCode = safeProblem(problemCode)
	if err := m.Save(m.state); err != nil {
		return err
	}
	return m.AppendEvent("upgrade.failed", "failure", m.state.ProblemCode)
}

func (m *Manager) Prepare(appVersion, pbVersion string, retry bool) error {
	if err := m.Acquire(); err != nil {
		return err
	}
	var err error
	m.state, err = m.readState()
	if err != nil {
		return err
	}
	if m.state.State == StatusFailed && !retry {
		return fmt.Errorf("previous upgrade failed (%s); inspect with `subflow upgrade status`, then run `subflow upgrade retry` or restore a verified backup", safeProblem(m.state.ProblemCode))
	}
	if m.state.ProblemCode == "restore_requires_matching_binary" {
		return fmt.Errorf("restored data requires its matching older SubFlow binary and original encryption key before another upgrade")
	}
	previousVersion, previousPB := m.state.AppVersion, m.state.PocketBaseVersion
	previousProblem := m.state.ProblemCode
	previousCompleted := !m.state.CompletedAt.IsZero()
	if previousVersion != "" && previousVersion != appVersion && compareVersions(previousVersion, appVersion) > 0 {
		return fmt.Errorf("downgrades are not supported (data is at %s, executable is %s)", previousVersion, appVersion)
	}
	previousBackup := m.state.BackupID
	if previousVersion != "" && previousVersion != appVersion {
		// Do not expose an older successful backup as the snapshot for a new
		// attempt that may fail before its own backup is complete.
		m.state.BackupID = ""
	}
	m.state.AppVersion, m.state.PocketBaseVersion, m.state.SchemaVersion = appVersion, pbVersion, 1
	m.state.Ready, m.state.State, m.state.Phase = false, StatusChecking, StatusChecking
	m.state.ProblemCode, m.state.StartedAt, m.state.CompletedAt = "", time.Now().UTC(), time.Time{}
	if m.state.History == nil {
		m.state.History = []MigrationEntry{}
	}
	if err := m.Save(m.state); err != nil {
		return err
	}
	dataDB := filepath.Join(m.DataDir, "data.db")
	info, statErr := os.Stat(dataDB)
	if errors.Is(statErr, os.ErrNotExist) || (statErr == nil && info.Size() == 0) {
		if previousVersion != "" {
			return fmt.Errorf("stored upgrade metadata exists but the PocketBase database is missing")
		}
		m.state.Source = "fresh"
		return m.Save(m.state)
	}
	if statErr != nil {
		return statErr
	}
	if retry && previousProblem == "unknown_schema" {
		if err := CheckLegacyFingerprint(dataDB); err != nil {
			return err
		}
		m.state.Source = "legacy"
	}
	if previousCompleted && previousVersion == appVersion {
		m.state.State, m.state.Phase, m.state.Ready = StatusChecking, StatusChecking, false
		return m.Save(m.state)
	}
	if previousVersion == "" {
		if err := CheckLegacyFingerprint(dataDB); err != nil {
			return err
		}
		m.state.Source = "legacy"
	} else {
		m.state.Source = "upgrade"
	}
	if err := m.Save(m.state); err != nil {
		return err
	}
	if retry && previousVersion == appVersion && previousBackup != "" {
		if err := VerifyBackup(filepath.Join(m.Dir, "backups", previousBackup)); err == nil {
			m.state.BackupID = previousBackup
			return m.Save(m.state)
		}
	}
	required, err := backupSize(m.DataDir)
	if err != nil {
		return err
	}
	available, err := diskFree(m.DataDir)
	if err != nil {
		return err
	}
	if available < required+(16<<20) {
		return fmt.Errorf("insufficient_disk_space")
	}
	m.state.State, m.state.Phase = StatusBackingUp, StatusBackingUp
	if err := m.Save(m.state); err != nil {
		return err
	}
	sourceApp, sourcePB := previousVersion, previousPB
	if sourceApp == "" {
		sourceApp, sourcePB = "legacy", "unknown"
	}
	manifest, err := m.CreateBackup(sourceApp, sourcePB)
	if err != nil {
		return err
	}
	m.state.BackupID = manifest.ID
	if err := m.Save(m.state); err != nil {
		return err
	}
	return m.AppendEvent("upgrade.backup.created", "success", "backup verified")
}

func (m *Manager) Complete(reconcile ...func() error) error {
	if err := m.AppendEvent("upgrade.completed", "success", "upgrade verified"); err != nil {
		m.state.State, m.state.Phase, m.state.Ready = StatusFailed, StatusFailed, false
		m.state.ProblemCode = "audit_unavailable"
		_ = m.Save(m.state)
		return err
	}
	for _, syncAudit := range reconcile {
		if syncAudit == nil {
			continue
		}
		if err := syncAudit(); err != nil {
			m.state.State, m.state.Phase, m.state.Ready = StatusFailed, StatusFailed, false
			m.state.ProblemCode = "audit_unavailable"
			_ = m.Save(m.state)
			return fmt.Errorf("durable upgrade audit could not be imported: %w", err)
		}
	}
	m.state.State, m.state.Phase, m.state.Ready = StatusReady, StatusReady, true
	m.state.ProblemCode, m.state.CompletedAt = "", time.Now().UTC()
	if err := m.Save(m.state); err != nil {
		return err
	}
	if m.state.BackupID != "" {
		if err := m.SetBackupStatus(m.state.BackupID, "successful"); err != nil {
			return err
		}
	}
	return m.PruneBackups(MaxBackupCount)
}

func (m *Manager) Close() error {
	if m.lock == nil {
		return nil
	}
	err := m.lock.Close()
	m.lock = nil
	return err
}

func (m *Manager) CreateBackup(appVersion, pbVersion string) (BackupManifest, error) {
	if err := os.MkdirAll(filepath.Join(m.Dir, "backups"), 0o700); err != nil {
		return BackupManifest{}, err
	}
	createdAt := time.Now().UTC()
	manifest := BackupManifest{ID: createdAt.Format("20060102T150405.000000000Z"), Status: "pending", AppVersion: appVersion, PocketBaseVersion: pbVersion, CreatedAt: createdAt, Files: []BackupFile{}}
	root := filepath.Join(m.Dir, "backups", manifest.ID)
	required, err := backupSize(m.DataDir)
	if err != nil {
		return BackupManifest{}, err
	}
	available, err := diskFree(m.DataDir)
	if err != nil {
		return BackupManifest{}, err
	}
	if available < required+(16<<20) {
		return BackupManifest{}, fmt.Errorf("insufficient_disk_space")
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		return BackupManifest{}, err
	}
	files, err := copyDataTree(m.DataDir, root)
	if err != nil {
		return BackupManifest{}, err
	}
	manifest.Files = files
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return BackupManifest{}, err
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), append(data, '\n'), 0o600); err != nil {
		return BackupManifest{}, err
	}
	if err := syncDirectory(root); err != nil {
		return BackupManifest{}, err
	}
	if err := verifyBackup(root, manifest); err != nil {
		return BackupManifest{}, err
	}
	return manifest, nil
}

func (m *Manager) ListBackups() ([]BackupManifest, error) {
	entries, err := os.ReadDir(filepath.Join(m.Dir, "backups"))
	if errors.Is(err, os.ErrNotExist) {
		return []BackupManifest{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]BackupManifest, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifest, err := readManifest(filepath.Join(m.Dir, "backups", entry.Name()))
		if err == nil && manifest.ID == entry.Name() && verifyBackup(filepath.Join(m.Dir, "backups", entry.Name()), manifest) == nil {
			result = append(result, manifest)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID > result[j].ID
		}
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func readManifest(root string) (BackupManifest, error) {
	data, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		return BackupManifest{}, err
	}
	var manifest BackupManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return BackupManifest{}, err
	}
	return manifest, nil
}

func verifyBackup(root string, manifest BackupManifest) error {
	if !backupID(manifest.ID) || (manifest.Status != "pending" && manifest.Status != "successful") || len(manifest.Files) == 0 {
		return fmt.Errorf("backup manifest is invalid")
	}
	seen := make(map[string]bool, len(manifest.Files))
	for _, item := range manifest.Files {
		clean := filepath.Clean(filepath.FromSlash(item.Path))
		if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || seen[clean] {
			return fmt.Errorf("backup manifest has an unsafe file path")
		}
		seen[clean] = true
		filePath := filepath.Join(root, clean)
		info, err := os.Lstat(filePath)
		if err != nil || !info.Mode().IsRegular() || info.Size() != item.Size {
			return fmt.Errorf("backup verification failed")
		}
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if hex.EncodeToString(hash.Sum(nil)) != item.SHA256 {
			return fmt.Errorf("backup verification failed")
		}
	}
	return nil
}

func copyDataTree(source, destination string) ([]BackupFile, error) {
	files := []BackupFile{}
	err := filepath.WalkDir(source, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, current)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if entry.Name() == StateDirectory {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("backup source contains an unsupported symbolic link")
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("backup source contains an unsupported file type")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		input, err := os.Open(current)
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			_ = input.Close()
			return err
		}
		hash := sha256.New()
		written, copyErr := io.Copy(io.MultiWriter(output, hash), input)
		if copyErr == nil {
			copyErr = output.Sync()
		}
		inputErr, outputErr := input.Close(), output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inputErr != nil {
			return inputErr
		}
		if outputErr != nil {
			return outputErr
		}
		files = append(files, BackupFile{Path: filepath.ToSlash(relative), Size: written, SHA256: hex.EncodeToString(hash.Sum(nil))})
		return nil
	})
	return files, err
}

func backupSize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != root && entry.Name() == StateDirectory && entry.IsDir() {
			return filepath.SkipDir
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("backup source contains an unsupported symbolic link")
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("backup source contains an unsupported file type")
		}
		total += info.Size()
		return nil
	})
	return total, err
}

func (m *Manager) SetBackupStatus(id, status string) error {
	if !backupID(id) || (status != "pending" && status != "successful") {
		return fmt.Errorf("invalid backup status")
	}
	root := filepath.Join(m.Dir, "backups", id)
	manifest, err := readManifest(root)
	if err != nil {
		return err
	}
	if manifest.ID != id {
		return fmt.Errorf("backup id does not match manifest")
	}
	if err := verifyBackup(root, manifest); err != nil {
		return err
	}
	manifest.Status = status
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), append(data, '\n'), 0o600); err != nil {
		return err
	}
	return syncDirectory(root)
}

func (m *Manager) PruneBackups(keep int) error {
	if keep < 0 {
		return fmt.Errorf("backup retention cannot be negative")
	}
	backups, err := m.ListBackups()
	if err != nil {
		return err
	}
	successful := make([]BackupManifest, 0, len(backups))
	for _, backup := range backups {
		if backup.Status == "successful" {
			successful = append(successful, backup)
		}
	}
	for _, backup := range successful[min(keep, len(successful)):] {
		if err := os.RemoveAll(filepath.Join(m.Dir, "backups", backup.ID)); err != nil {
			return err
		}
	}
	return nil
}

func VerifyBackup(root string) error {
	manifest, err := readManifest(root)
	if err != nil {
		return err
	}
	return verifyBackup(root, manifest)
}

func (m *Manager) AppendEvent(action, outcome, summary string) error {
	if !safeEventValue(action) || !safeEventValue(outcome) || len(summary) > 256 {
		return fmt.Errorf("invalid upgrade event")
	}
	if err := os.MkdirAll(m.Dir, 0o700); err != nil {
		return err
	}
	event := Event{ID: fmt.Sprintf("%d-%x", time.Now().UTC().UnixNano(), sha256.Sum256([]byte(action+outcome+summary)))[:40], Action: action, Outcome: outcome, CreatedAt: time.Now().UTC(), Summary: summary}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(m.Dir, "events.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func safeEventValue(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if !(char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '.' || char == '_' || char == '-') {
			return false
		}
	}
	return true
}

func safeProblem(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var result strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result.WriteRune(r)
		}
		if result.Len() >= 64 {
			break
		}
	}
	if result.Len() == 0 {
		return "upgrade_failed"
	}
	return result.String()
}

func backupID(value string) bool {
	if len(value) != len("20060102T150405.000000000Z") || strings.ContainsAny(value, `/\\`) {
		return false
	}
	_, err := time.Parse("20060102T150405.000000000Z", value)
	return err == nil
}
