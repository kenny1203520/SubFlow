package upgrade

import (
	"context"

	"ganigeorgiev.com/pocketbase/pocketbase/core"

	"subflow/internal/adapters"
	"subflow/internal/application"
)

func migrationAudit(driver string) func(context.Context, core.App, Migration) error {
	return func(ctx context.Context, app core.App, migration Migration) error {
		stores, err := adapters.New(driver, app)
		if err != nil {
			return err
		}
		return application.New(stores).RecordAudit(ctx, "", "", "system.upgrade.migration.applied", "migration", migration.ID, "success", "checksum="+migration.Checksum)
	}
}
