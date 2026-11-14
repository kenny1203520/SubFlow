package upgrade

import (
	"archive/zip"
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
	"time"
)

func inspectPocketBaseZIP(reader io.ReaderAt, size int64) ([]*zip.File, bool, int64, error) {
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, false, 0, err
	}
	seen := map[string]bool{}
	regular := map[string]bool{}
	hasData, nested := false, false
	var total int64
	files := []*zip.File{}
	for _, file := range archive.File {
		name := strings.TrimSuffix(file.Name, "/")
		path, err := manifestPath(name)
		key := strings.ToLower(name)
		if err != nil {
			return nil, false, 0, err
		}
		if seen[key] || (!file.Mode().IsRegular() && !file.Mode().IsDir()) {
			return nil, false, 0, fmt.Errorf("invalid or duplicate ZIP entry")
		}
		seen[key] = true
		if !file.Mode().IsDir() {
			regular[key] = true
		}
		if ExcludedBackupPath(path) {
			nested = true
			continue
		}
		if file.Mode().IsDir() || !restorablePath(path) {
			continue
		}
		if file.UncompressedSize64 > uint64(math.MaxInt64-total-4096) {
			return nil, false, 0, fmt.Errorf("ZIP size overflow")
		}
		total += int64(file.UncompressedSize64) + 4096
		files = append(files, file)
		hasData = hasData || name == "data.db"
	}
	for key := range seen {
		for parent := filepath.ToSlash(filepath.Dir(key)); parent != "."; parent = filepath.ToSlash(filepath.Dir(parent)) {
			if regular[parent] {
				return nil, false, 0, fmt.Errorf("ZIP entry conflicts with parent file")
			}
		}
	}
	if !hasData {
		return nil, false, 0, fmt.Errorf("data.db is missing from the backup")
	}
	return files, nested, total, nil
}

func CanRestorePocketBaseZIP(reader io.ReaderAt, size int64) (int64, error) {
	_, nested, total, err := inspectPocketBaseZIP(reader, size)
	if err == nil && nested {
		err = fmt.Errorf("archive contains legacy backup work files; use subflow upgrade restore --pocketbase-zip")
	}
	return total, err
}

type SeekReaderAt struct{ io.ReadSeeker }

func (reader SeekReaderAt) ReadAt(out []byte, offset int64) (int, error) {
	if _, err := reader.Seek(offset, io.SeekStart); err != nil {
		return 0, err
	}
	return io.ReadFull(reader, out)
}

func (m *Manager) RestorePocketBaseZIP(path, sourceVersion, pbVersion string) (manifest BackupManifest, resultErr error) {
	if err := m.Acquire(); err != nil {
		return manifest, err
	}
	if canonicalVersion(sourceVersion) == "" {
		return manifest, ErrInvalidVersion
	}
	input, err := regularBackupFile(filepath.Dir(path), filepath.Base(path))
	if err != nil {
		return manifest, err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return manifest, err
	}
	if !info.Mode().IsRegular() {
		return manifest, fmt.Errorf("backup ZIP must be a regular file")
	}
	files, _, total, err := inspectPocketBaseZIP(input, info.Size())
	if err != nil {
		return manifest, err
	}
	if err := m.AppendEvent("upgrade.restore.started", "success", "offline PocketBase ZIP validated"); err != nil {
		return manifest, err
	}
	journalWritten := false
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, m.AppendEvent("upgrade.restore.failed", "failure", "offline_pocketbase_restore_failed"))
			if journalWritten {
				resultErr = errors.Join(resultErr, m.Fail("restore_failed"))
			}
		}
	}()
	if err := ensureRealDirectory(m.recoveryDirectory()); err != nil {
		return manifest, err
	}
	if err := checkSpace(m.recoveryDirectory(), total); err != nil {
		return manifest, err
	}
	now := time.Now().UTC()
	id := now.Format("20060102T150405.000000000Z")
	manifest = BackupManifest{ID: id, Status: "pending", AppVersion: sourceVersion, PocketBaseVersion: pbVersion, CreatedAt: now}
	stage := filepath.Join(m.recoveryDirectory(), ".creating-"+id)
	if err := os.Mkdir(stage, 0o700); err != nil {
		return manifest, err
	}
	defer func() {
		if err := safeRemoveTree(m.recoveryDirectory(), filepath.Base(stage)); err != nil {
			m.cleanupWarning(err)
		}
	}()
	stageRoot, err := os.OpenRoot(stage)
	if err != nil {
		return manifest, err
	}
	defer stageRoot.Close()
	for _, file := range files {
		item, err := extractBackupFile(file, stageRoot)
		if err != nil {
			return manifest, spaceError(err)
		}
		manifest.Files = append(manifest.Files, item)
	}
	if err := checkSQLiteFiles(stage); err != nil {
		return manifest, err
	}
	payload, err := json.Marshal(manifest)
	if err != nil {
		return manifest, err
	}
	if err := writeSyncedFile(filepath.Join(stage, "manifest.json"), payload); err != nil {
		return manifest, spaceError(err)
	}
	if err := verifyBackup(stage, manifest); err != nil {
		return manifest, err
	}
	if err := syncDirectory(stage); err != nil {
		return manifest, err
	}
	if err := stageRoot.Close(); err != nil {
		return manifest, err
	}
	root := filepath.Join(m.recoveryDirectory(), id)
	if err := os.Rename(stage, root); err != nil {
		return manifest, err
	}
	if err := syncDirectory(m.recoveryDirectory()); err != nil {
		return manifest, err
	}
	return m.restoreVerifiedTree(root, manifest, &journalWritten)
}

func extractBackupFile(file *zip.File, root *os.Root) (BackupFile, error) {
	item := BackupFile{Path: file.Name, Size: int64(file.UncompressedSize64)}
	target, err := manifestPath(file.Name)
	if err != nil {
		return item, err
	}
	if err := root.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return item, err
	}
	input, err := file.Open()
	if err != nil {
		return item, err
	}
	defer input.Close()
	output, err := root.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return item, err
	}
	hash := sha256.New()
	count, err := io.Copy(io.MultiWriter(output, hash), io.LimitReader(input, item.Size+1))
	if err == nil && count != item.Size {
		err = fmt.Errorf("ZIP entry length mismatch")
	}
	if err == nil {
		err = output.Sync()
	}
	item.SHA256 = hex.EncodeToString(hash.Sum(nil))
	return item, errors.Join(err, output.Close())
}
