package upgrade

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const RecoveryDirectory = ".subflow-recovery"

func ExcludedBackupPath(relative string) bool {
	parts := strings.Split(filepath.ToSlash(relative), "/")
	switch parts[0] {
	case RecoveryDirectory, "backups", ".pb_temp_to_delete", ".notify", ".autocert_cache", "lost+found", "data.db-shm", "auxiliary.db-shm":
		return true
	case StateDirectory:
		if len(parts) == 1 {
			return false
		}
		name := parts[1]
		return name == "backups" || name == "upgrade.lock" || name == "restore.json" || name == "restore.json.tmp" ||
			strings.HasPrefix(name, ".state-") || strings.HasPrefix(name, ".restore-journal-") || strings.HasPrefix(name, "restore-stage-") || strings.HasPrefix(name, "restore-failed-")
	}
	return false
}

func walkBackupTree(root string, visit func(string, fs.DirEntry) error) error {
	if err := realDirectory(root); err != nil {
		return err
	}
	return filepath.WalkDir(root, func(current string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, current)
		if err != nil || relative == "." {
			return err
		}
		if ExcludedBackupPath(relative) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() && !entry.Type().IsRegular() {
			return fmt.Errorf("backup tree contains an unsupported file type: %s", relative)
		}
		return visit(relative, entry)
	})
}

func realDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("unsafe backup directory: %s", filepath.Base(path))
	}
	return nil
}

func checkSpace(path string, bytes int64) error {
	reserve := max(int64(64<<20), bytes/10)
	free, err := diskFree(path)
	if err != nil {
		return err
	}
	if bytes < 0 || bytes > math.MaxInt64-reserve || free < bytes+reserve {
		return fmt.Errorf("insufficient_disk_space: need %d bytes plus %d bytes reserve, have %d", bytes, reserve, free)
	}
	return nil
}

func spaceError(err error) error {
	if errors.Is(err, syscall.ENOSPC) {
		return fmt.Errorf("insufficient_disk_space: %w", err)
	}
	return err
}

// Reject symlink components even when their destination stays under the root.
func regularBackupFile(root, relative string) (*os.File, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	parts := strings.Split(filepath.ToSlash(relative), "/")
	for i := range parts {
		info, err := r.Lstat(filepath.Join(parts[:i+1]...))
		if err != nil {
			return nil, err
		}
		if (i < len(parts)-1 && !info.IsDir()) || (i == len(parts)-1 && !info.Mode().IsRegular()) {
			return nil, fmt.Errorf("unsafe backup file")
		}
	}
	return r.Open(relative)
}
