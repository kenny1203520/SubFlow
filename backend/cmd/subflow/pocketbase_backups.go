package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"ganigeorgiev.com/pocketbase/pocketbase/core"
	"subflow/internal/upgrade"
)

var pbBackupTemporary = regexp.MustCompile("^pb_backup_[A-Za-z0-9]{6}$")

func registerPocketBaseBackups(app core.App, manager *upgrade.Manager) {
	var operations sync.Mutex
	app.OnBackupCreate().BindFunc(func(e *core.BackupEvent) error {
		operations.Lock()
		defer operations.Unlock()
		exclusions, err := manager.PocketBaseBackupExclusions()
		if err != nil {
			return err
		}
		e.Exclude = append(e.Exclude, exclusions...)
		temp := filepath.Join(e.App.DataDir(), core.LocalTempDirName)
		if err := cleanPocketBaseBackupTemporary(temp); err != nil {
			return err
		}
		if err := manager.CheckPocketBaseBackupSpace(e.App.Settings().Backups.S3.Enabled); err != nil {
			return err
		}
		err = e.Next()
		if cleanupErr := cleanPocketBaseBackupTemporary(temp); cleanupErr != nil {
			e.App.Logger().Warn("PocketBase backup temporary cleanup pending", "error", cleanupErr.Error())
			if err != nil {
				return errors.Join(err, cleanupErr)
			}
		}
		return err
	})
	app.OnBackupRestore().BindFunc(func(e *core.BackupEvent) error {
		operations.Lock()
		defer operations.Unlock()
		if err := manager.AllowNativePocketBaseRestore(); err != nil {
			return err
		}
		fs, err := e.App.NewBackupsFilesystem()
		if err != nil {
			return err
		}
		defer fs.Close()
		fs.SetContext(e.Context)
		reader, err := fs.GetReader(e.Name)
		if err != nil {
			return err
		}
		size := reader.Size()
		total, inspectErr := upgrade.CanRestorePocketBaseZIP(upgrade.SeekReaderAt{ReadSeeker: reader}, size)
		if err := errors.Join(inspectErr, reader.Close()); err != nil {
			return err
		}
		if err := manager.CheckPocketBaseRestoreSpace(total, size, e.App.Settings().Backups.S3.Enabled); err != nil {
			return err
		}
		e.Exclude = append(e.Exclude, upgrade.RecoveryDirectory)
		return e.Next()
	})
}

func cleanPocketBaseBackupTemporary(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("unsafe PocketBase temporary directory")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return err
	}
	defer root.Close()
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	var result error
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		if pbBackupTemporary.MatchString(entry.Name()) || entry.Name() == "data.db" || entry.Name() == "auxiliary.db" {
			result = errors.Join(result, root.Remove(entry.Name()))
		}
	}
	return result
}

func reclaimPocketBaseBackupTemporary(manager *upgrade.Manager) {
	if !manager.HasLock() {
		return
	}
	if err := cleanPocketBaseBackupTemporary(filepath.Join(manager.DataDir, core.LocalTempDirName)); err != nil {
		log.Printf("SubFlow PocketBase backup cleanup pending: %v", err)
	}
}
