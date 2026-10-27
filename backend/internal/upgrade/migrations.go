package upgrade

import (
	"context"

	"ganigeorgiev.com/pocketbase/pocketbase/core"
)

// Migrations returns the immutable sequence supported by this application
// build. New schema changes append a new migration; they never alter the
// alpha.1-alpha.5 baseline body or its checksum.
func Migrations(driver, appURL string, setupLink *string) []Migration {
	return []Migration{legacyAlphaBaseline(driver, appURL, setupLink)}
}

func MigrationAudit(driver string) func(context.Context, core.App, Migration) error {
	return migrationAudit(driver)
}
