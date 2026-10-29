package pocketbase

import (
	"fmt"

	"ganigeorgiev.com/pocketbase/pocketbase/core"
)

// EnsureCaptchaProviderSettingsSchema adds the settings introduced after the
// frozen alpha baseline. Keep this separate from EnsureSchema so the released
// baseline schema and checksum remain immutable.
func EnsureCaptchaProviderSettingsSchema(app core.App) error {
	collection, err := app.FindCollectionByNameOrId(CollectionSystemSettings)
	if err != nil {
		return fmt.Errorf("system settings collection is missing: %w", err)
	}

	if field := collection.Fields.GetByName("captcha_enabled"); field != nil {
		if _, ok := field.(*core.BoolField); !ok {
			return fmt.Errorf("system settings captcha_enabled field must be a boolean")
		}
	}
	if field := collection.Fields.GetByName("captcha_providers"); field != nil {
		if _, ok := field.(*core.JSONField); !ok {
			return fmt.Errorf("system settings captcha_providers field must be JSON")
		}
	}

	_, err = ensureCollection(app, CollectionSystemSettings, func(c *core.Collection) {
		c.Fields.Add(&core.BoolField{Name: "captcha_enabled"}, &core.JSONField{Name: "captcha_providers", MaxSize: 1024 * 32})
	})
	return err
}
