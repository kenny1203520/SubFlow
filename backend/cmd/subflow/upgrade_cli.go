package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"ganigeorgiev.com/pocketbase/pocketbase"
	"ganigeorgiev.com/pocketbase/pocketbase/core"

	"subflow/internal/config"
	"subflow/internal/mailer"
	"subflow/internal/upgrade"
)

func registerUpgradeBootstrap(app *pocketbase.PocketBase, manager *upgrade.Manager, driver, appURL string, smtpMailer *mailer.SMTP) *string {
	var setupLink string
	app.OnBootstrap().BindFunc(func(event *core.BootstrapEvent) error {
		if err := event.Next(); err != nil {
			return err
		}
		if err := manager.SetPhase(upgrade.StatusMigrating); err != nil {
			return err
		}
		if err := upgrade.RunMigrations(context.Background(), event.App, manager, upgrade.Migrations(driver, appURL, &setupLink), upgrade.MigrationAudit(driver)); err != nil {
			return err
		}
		if err := upgrade.ReconcileDurableEvents(context.Background(), event.App, driver, manager.Dir); err != nil {
			return err
		}
		if err := manager.SetPhase(upgrade.StatusVerifying); err != nil {
			return err
		}
		return mailer.ConfigurePocketBase(event.App, smtpMailer)
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

func dispatchUpgradeCommand(args []string, driver, environment string, smtpMailer *mailer.SMTP) (bool, error) {
	if len(args) < 2 || args[1] != "upgrade" {
		return false, nil
	}
	if len(args) < 3 {
		return true, fmt.Errorf("usage: subflow upgrade <status|retry|restore>")
	}
	manager, err := upgrade.New(config.DataDirFromArgs(args))
	if err != nil {
		return true, err
	}
	switch args[2] {
	case "status":
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
		return true, retryUpgrade(manager, driver, environment, smtpMailer)
	default:
		return true, fmt.Errorf("unknown upgrade command %q; use status, retry, or restore", args[2])
	}
}

func retryUpgrade(manager *upgrade.Manager, driver, environment string, smtpMailer *mailer.SMTP) (result error) {
	defer func() {
		if err := manager.Close(); result == nil && err != nil {
			result = err
		}
	}()
	pbVersion, err := upgrade.PocketBaseVersion()
	if err != nil {
		return err
	}
	if pbVersion != upgrade.RequiredPocketBaseVersion {
		return fmt.Errorf("this executable uses PocketBase %s; expected %s", pbVersion, upgrade.RequiredPocketBaseVersion)
	}
	if err := manager.Prepare(upgrade.AppVersion(), pbVersion, true); err != nil {
		if errors.Is(err, upgrade.ErrRestoreRecovered) {
			return err
		}
		if manager.HasLock() {
			_ = manager.Fail(problemCode(err))
		}
		return err
	}
	if err := manager.AppendEvent("upgrade.retry.started", "success", "operator initiated retry"); err != nil {
		_ = manager.Fail("audit_unavailable")
		return err
	}
	app := pocketbase.New()
	registerUpgradeBootstrap(app, manager, driver, config.AppURL(os.Getenv("SUBFLOW_APP_URL"), os.Getenv("SUBFLOW_HTTP_PORT")), smtpMailer)
	if err := app.Bootstrap(); err != nil {
		_ = manager.Fail(problemCode(err))
		_ = app.ClearBootstrap()
		return err
	}
	if err := manager.Complete(func() error {
		return upgrade.ReconcileDurableEvents(context.Background(), app, driver, manager.Dir)
	}); err != nil {
		_ = manager.Fail("upgrade_finalize_failed")
		_ = app.ClearBootstrap()
		return err
	}
	return app.ClearBootstrap()
}

func problemCode(err error) string {
	if err == nil {
		return "upgrade_failed"
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "insufficient_disk_space"):
		return "insufficient_disk_space"
	case errors.Is(err, upgrade.ErrUnknownSchema):
		return "unknown_schema"
	case strings.Contains(message, "checksum mismatch"):
		return "migration_checksum_mismatch"
	case strings.Contains(message, "downgrade"):
		return "unsupported_downgrade"
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
