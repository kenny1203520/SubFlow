package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"ganigeorgiev.com/pocketbase/pocketbase"
	"ganigeorgiev.com/pocketbase/pocketbase/core"

	"subflow/internal/config"
	"subflow/internal/security"
	"subflow/internal/upgrade"
)

func registerUpgradeBootstrap(app *pocketbase.PocketBase, manager *upgrade.Manager, driver, setupURL, encryptionKey string) *string {
	registerPocketBaseBackups(app, manager)
	var setupLink string
	app.OnBootstrap().BindFunc(func(event *core.BootstrapEvent) error {
		if err := manager.SetPhase(upgrade.StatusMigrating); err != nil {
			return err
		}
		if err := event.Next(); err != nil {
			return err
		}
		if err := upgrade.RunMigrations(context.Background(), event.App, manager, upgrade.Migrations(driver, setupURL, &setupLink), upgrade.MigrationAudit(driver, encryptionKey)); err != nil {
			return err
		}
		if err := manager.SetPhase(upgrade.StatusVerifying); err != nil {
			return err
		}
		if err := upgrade.VerifyLiveDatabases(event.App); err != nil {
			return err
		}
		if err := upgrade.ReconcileDurableEvents(context.Background(), event.App, driver, manager.Dir, encryptionKey); err != nil {
			return err
		}
		return nil
	})
	return &setupLink
}

type upgradeStatusOutput struct {
	State   upgrade.State  `json:"state"`
	Backups []backupStatus `json:"backups"`
}

type backupStatus struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	AppVersion        string `json:"appVersion"`
	PocketBaseVersion string `json:"pocketBaseVersion"`
	CreatedAt         string `json:"createdAt"`
	FileCount         int    `json:"fileCount"`
}

func dispatchUpgradeCommand(args []string, driver, environment, httpPort, encryptionKey string) (bool, error) {
	if len(args) < 2 || args[1] != "upgrade" {
		return false, nil
	}
	if len(args) < 3 {
		return true, fmt.Errorf("usage: subflow upgrade <status|preflight|retry|restore>")
	}
	manager, err := upgrade.New(config.DataDirFromArgs(args))
	if err != nil {
		return true, err
	}
	switch args[2] {
	case "status":
		for _, arg := range args[3:] {
			if arg == "--state-only" {
				state, err := manager.ReadState()
				if err != nil {
					return true, err
				}
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				return true, encoder.Encode(state)
			}
		}
		state, backups, err := manager.Snapshot()
		if err != nil {
			return true, err
		}
		output := upgradeStatusOutput{State: state, Backups: make([]backupStatus, 0, len(backups))}
		for _, backup := range backups {
			output.Backups = append(output.Backups, backupStatus{ID: backup.ID, Status: backup.Status, AppVersion: backup.AppVersion, PocketBaseVersion: backup.PocketBaseVersion, CreatedAt: backup.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"), FileCount: len(backup.Files)})
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return true, encoder.Encode(output)
	case "restore":
		if len(args) >= 5 && args[3] == "--pocketbase-zip" {
			sourceVersion := ""
			pbVersion := upgrade.RequiredPocketBaseVersion
			for _, arg := range args[5:] {
				if value, ok := strings.CutPrefix(arg, "--source-version="); ok {
					sourceVersion = value
				}
				if value, ok := strings.CutPrefix(arg, "--pocketbase-version="); ok {
					pbVersion = value
				}
			}
			if sourceVersion == "" {
				return true, fmt.Errorf("usage: subflow upgrade restore --pocketbase-zip <path> --source-version=<version>")
			}
			defer manager.Close()
			manifest, err := manager.RestorePocketBaseZIP(args[4], sourceVersion, pbVersion)
			if err != nil {
				return true, err
			}
			fmt.Printf("Restored %s. Use SubFlow %s / PocketBase %s and the original encryption key.\n", manifest.ID, manifest.AppVersion, manifest.PocketBaseVersion)
			return true, nil
		}
		backupID := ""
		for index := 3; index < len(args); index++ {
			if strings.HasPrefix(args[index], "--backup=") {
				backupID = strings.TrimPrefix(args[index], "--backup=")
			} else if args[index] == "--backup" && index+1 < len(args) {
				backupID = args[index+1]
				index++
			}
		}
		if backupID == "" {
			return true, fmt.Errorf("usage: subflow upgrade restore --backup <id>")
		}
		manifest, err := manager.RestoreBackup(backupID)
		if err != nil {
			return true, err
		}
		if _, err := fmt.Fprintf(os.Stdout, "Restored data from %s. Start the matching SubFlow %s / PocketBase %s binary and use its original encryption key.\n", manifest.ID, manifest.AppVersion, manifest.PocketBaseVersion); err != nil {
			return true, err
		}
		return true, manager.Close()
	case "retry":
		return true, retryUpgrade(manager, driver, environment, httpPort, encryptionKey)
	case "preflight":
		report := manager.Preflight(upgrade.AppVersion())
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			return true, err
		}
		if len(report.Blockers) != 0 {
			return true, fmt.Errorf("upgrade preflight found blockers")
		}
		return true, nil
	default:
		return true, fmt.Errorf("unknown upgrade command; use status, preflight, retry, or restore")
	}
}

func retryUpgrade(manager *upgrade.Manager, driver, environment, httpPort, encryptionKey string) (result error) {
	stopProgress := manager.StartProgress(log.Printf)
	defer stopProgress()
	if _, err := security.DeriveKeyring(encryptionKey); err != nil {
		return err
	}
	defer func() {
		if err := manager.Close(); result == nil && err != nil {
			result = err
		}
		if result == nil {
			fmt.Fprintln(os.Stdout, "SubFlow upgrade retry completed. Start SubFlow with the same image, data volume and encryption key, then check readiness.")
		}
	}()
	pbVersion, err := upgrade.PocketBaseVersion()
	if err != nil {
		return err
	}
	if pbVersion != upgrade.RequiredPocketBaseVersion {
		return fmt.Errorf("this executable uses PocketBase %s; expected %s", pbVersion, upgrade.RequiredPocketBaseVersion)
	}
	if err := manager.Acquire(); err != nil {
		return err
	}
	reclaimPocketBaseBackupTemporary(manager)
	if err := manager.Prepare(upgrade.AppVersion(), pbVersion, true); err != nil {
		if errors.Is(err, upgrade.ErrRestoreRecovered) {
			return err
		}
		if manager.HasLock() {
			recordRetryFailure(manager, problemCode(err))
		}
		if errors.Is(err, upgrade.ErrUnsupportedDowngrade) {
			return err
		}
		return fmt.Errorf("upgrade retry failed (%s)", problemCode(err))
	}
	if err := manager.AppendEvent("upgrade.retry.started", "success", "operator initiated retry"); err != nil {
		recordRetryFailure(manager, "audit_unavailable")
		return fmt.Errorf("upgrade retry failed (audit_unavailable)")
	}
	app := pocketbase.New()
	registerUpgradeBootstrap(app, manager, driver, config.AppURL("", httpPort), encryptionKey)
	if err := app.Bootstrap(); err != nil {
		recordRetryFailure(manager, problemCode(err))
		_ = app.ClearBootstrap()
		return fmt.Errorf("upgrade retry failed (%s)", problemCode(err))
	}
	if err := manager.Complete(func() error {
		return upgrade.ReconcileDurableEvents(context.Background(), app, driver, manager.Dir, encryptionKey)
	}); err != nil {
		recordRetryFailure(manager, "upgrade_finalize_failed")
		_ = app.ClearBootstrap()
		return fmt.Errorf("upgrade retry failed (upgrade_finalize_failed)")
	}
	return app.ClearBootstrap()
}

func recordRetryFailure(manager *upgrade.Manager, code string) {
	if err := manager.Fail(code); err != nil {
		log.Print("SubFlow could not persist the retry failure (upgrade_state_unavailable)")
	}
}

func problemCode(err error) string {
	if err == nil {
		return "upgrade_failed"
	}
	message := strings.ToLower(err.Error())
	switch {
	case errors.Is(err, upgrade.ErrInvalidVersion):
		return "invalid_upgrade_version"
	case errors.Is(err, upgrade.ErrUnsupportedDowngrade):
		return "unsupported_downgrade"
	case strings.Contains(message, "insufficient_disk_space"):
		return "insufficient_disk_space"
	case errors.Is(err, upgrade.ErrUnknownSchema):
		return "unknown_schema"
	case strings.Contains(message, "checksum mismatch"):
		return "migration_checksum_mismatch"
	case strings.Contains(message, "backup"):
		return "backup_failed"
	case strings.Contains(message, "audit"):
		return "audit_unavailable"
	case strings.Contains(message, "migration"):
		return "migration_failed"
	default:
		return "upgrade_failed"
	}
}
