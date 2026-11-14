package upgrade

import (
	"fmt"

	"ganigeorgiev.com/pocketbase/pocketbase/core"

	pbadapter "subflow/internal/adapters/pocketbase"
)

const (
	PasswordResetSettingsID = "0005_password_reset_settings"
	Migration0005Checksum   = "sha256:f98c7b758d86416804c59a4279b41ae071d3f463ff58dcb5755a9fd896a7be67"
)

func passwordResetSettings() Migration {
	return Migration{
		ID:       PasswordResetSettingsID,
		Checksum: Migration0005Checksum,
		Requires: []string{CaptchaProviderSettingsID},
		Up: func(app core.App) error {
			settings, err := app.FindCollectionByNameOrId(pbadapter.CollectionSystemSettings)
			if err != nil {
				return fmt.Errorf("system settings collection is missing: %w", err)
			}
			if settings.Fields.GetByName("allow_password_reset") == nil {
				settings.Fields.Add(&core.BoolField{Name: "allow_password_reset"})
				if err := app.Save(settings); err != nil {
					return err
				}
			}
			outbox, err := app.FindCollectionByNameOrId(pbadapter.CollectionNotificationEmailOutbox)
			if err != nil {
				return fmt.Errorf("notification email outbox collection is missing: %w", err)
			}
			if outbox.Fields.GetByName("recipient_email") == nil {
				outbox.Fields.Add(&core.TextField{Name: "recipient_email", Hidden: true, Max: 255})
				if err := app.Save(outbox); err != nil {
					return err
				}
			}
			records, err := app.FindRecordsByFilter(pbadapter.CollectionSystemSettings, "", "", 0, 0, nil)
			if err != nil {
				return err
			}
			for _, record := range records {
				// Existing installations are enabled by default. The setup record
				// for a new installation remains disabled until an administrator opts in.
				record.Set("allow_password_reset", record.GetBool("initialized"))
				if err := app.Save(record); err != nil {
					return err
				}
			}
			return nil
		},
		Verify: func(app core.App) error {
			settings, err := app.FindCollectionByNameOrId(pbadapter.CollectionSystemSettings)
			if err != nil {
				return err
			}
			if _, ok := settings.Fields.GetByName("allow_password_reset").(*core.BoolField); !ok {
				return fmt.Errorf("password reset setting is missing")
			}
			outbox, err := app.FindCollectionByNameOrId(pbadapter.CollectionNotificationEmailOutbox)
			if err != nil {
				return err
			}
			if _, ok := outbox.Fields.GetByName("recipient_email").(*core.TextField); !ok {
				return fmt.Errorf("notification email recipient field is missing")
			}
			records, err := app.FindRecordsByFilter(pbadapter.CollectionSystemSettings, "", "", 0, 0, nil)
			if err != nil {
				return err
			}
			for _, record := range records {
				if record.GetBool("allow_password_reset") != record.GetBool("initialized") {
					return fmt.Errorf("password reset default does not match installation state")
				}
			}
			return nil
		},
	}
}
