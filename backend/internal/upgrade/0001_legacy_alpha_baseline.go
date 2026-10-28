package upgrade

import (
	"fmt"

	"ganigeorgiev.com/pocketbase/pocketbase/core"

	pbadapter "subflow/internal/adapters/pocketbase"
)

const (
	LegacyBaselineID       = "0001_legacy_alpha_baseline"
	LegacyBaselineChecksum = "sha256:87be9994377a9b558efefe9704b4eaae1cd23673a649fc499ceef2663dc1b51d"
)

func legacyAlphaBaseline(driver, appURL string, setupLink *string) Migration {
	return Migration{
		ID: LegacyBaselineID, Checksum: LegacyBaselineChecksum,
		Precondition: func(app core.App) error {
			if _, err := app.FindCollectionByNameOrId("users"); err != nil {
				return fmt.Errorf("PocketBase users collection is missing")
			}
			return nil
		},
		Up: func(app core.App) error {
			link, err := pbadapter.EnsureSchemaWithSetupURL(app, appURL)
			if err == nil && setupLink != nil {
				*setupLink = link
			}
			return err
		},
		Verify: func(app core.App) error {
			for _, collection := range []string{"users", "groups", "expenses", "incomes", "subscriptions", "settlements", "audit_logs", "system_settings", "categories"} {
				if _, err := app.FindCollectionByNameOrId(collection); err != nil {
					return fmt.Errorf("required collection %s is missing", collection)
				}
			}
			return nil
		},
	}
}
