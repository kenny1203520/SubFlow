package upgrade

import (
	"encoding/json"
	"fmt"

	"ganigeorgiev.com/pocketbase/pocketbase/core"

	pbadapter "subflow/internal/adapters/pocketbase"
	"subflow/internal/domain"
)

const (
	CaptchaProviderSettingsID       = "0004_captcha_provider_settings"
	CaptchaProviderSettingsChecksum = "sha256:c83305a13480d22a2ae5be311de952f8b156aa4e25cce8c5852ceca337c3029d"
)

func captchaProviderSettings() Migration {
	return Migration{
		ID:       CaptchaProviderSettingsID,
		Checksum: CaptchaProviderSettingsChecksum,
		Requires: []string{SettlementApprovalsID},
		Precondition: func(app core.App) error {
			if _, err := app.FindCollectionByNameOrId(pbadapter.CollectionSystemSettings); err != nil {
				return fmt.Errorf("system settings collection is missing")
			}
			return nil
		},
		Up: func(app core.App) error {
			collection, err := app.FindCollectionByNameOrId(pbadapter.CollectionSystemSettings)
			if err != nil {
				return fmt.Errorf("system settings collection is missing: %w", err)
			}
			shouldBackfillEnabled := collection.Fields.GetByName("captcha_enabled") == nil
			if err := pbadapter.EnsureCaptchaProviderSettingsSchema(app); err != nil {
				return err
			}

			records, err := app.FindRecordsByFilter(pbadapter.CollectionSystemSettings, "", "", 0, 0, nil)
			if err != nil {
				return err
			}
			for _, record := range records {
				changed := false
				provider := record.GetString("captcha_provider")
				if provider == "altcha" {
					provider = "altcha_community"
					record.Set("captcha_provider", provider)
					changed = true
				}

				profiles := map[string]domain.CaptchaProviderSettings{}
				if raw := record.GetString("captcha_providers"); raw != "" && raw != "null" {
					if err := json.Unmarshal([]byte(raw), &profiles); err != nil {
						return fmt.Errorf("decode saved CAPTCHA provider settings: %w", err)
					}
				}
				if profiles == nil {
					profiles = map[string]domain.CaptchaProviderSettings{}
				}
				if alias, ok := profiles["altcha"]; ok {
					community := profiles["altcha_community"]
					mergeMissingCaptchaProfile(&community, alias)
					profiles["altcha_community"] = community
					delete(profiles, "altcha")
					changed = true
				}
				if provider != "" {
					profile := profiles[provider]
					legacy := domain.CaptchaProviderSettings{
						SiteKey:      record.GetString("captcha_site_key"),
						ChallengeURL: record.GetString("captcha_challenge_url"),
						VerifyURL:    record.GetString("captcha_verify_url"),
						Secret:       record.GetString("captcha_secret"),
						Configured:   record.GetString("captcha_secret") != "",
					}
					mergeMissingCaptchaProfile(&profile, legacy)
					profiles[provider] = profile
					changed = true
				}

				if shouldBackfillEnabled {
					record.Set("captcha_enabled", provider != "")
					changed = true
				}
				encoded, err := json.Marshal(profiles)
				if err != nil {
					return fmt.Errorf("encode saved CAPTCHA provider settings: %w", err)
				}
				if record.GetString("captcha_providers") != string(encoded) {
					record.Set("captcha_providers", string(encoded))
					changed = true
				}
				if changed {
					if err := app.Save(record); err != nil {
						return err
					}
				}
			}
			return nil
		},
		Verify: func(app core.App) error {
			collection, err := app.FindCollectionByNameOrId(pbadapter.CollectionSystemSettings)
			if err != nil {
				return err
			}
			if _, ok := collection.Fields.GetByName("captcha_enabled").(*core.BoolField); !ok {
				return fmt.Errorf("CAPTCHA enabled setting is missing")
			}
			if _, ok := collection.Fields.GetByName("captcha_providers").(*core.JSONField); !ok {
				return fmt.Errorf("CAPTCHA provider settings are missing")
			}

			records, err := app.FindRecordsByFilter(pbadapter.CollectionSystemSettings, "", "", 0, 0, nil)
			if err != nil {
				return err
			}
			for _, record := range records {
				provider := record.GetString("captcha_provider")
				if provider == "altcha" {
					return fmt.Errorf("legacy ALTCHA provider alias was not normalized")
				}
				if provider == "" {
					continue
				}
				profiles := map[string]domain.CaptchaProviderSettings{}
				if err := json.Unmarshal([]byte(record.GetString("captcha_providers")), &profiles); err != nil {
					return fmt.Errorf("decode migrated CAPTCHA provider settings: %w", err)
				}
				if _, ok := profiles[provider]; !ok {
					return fmt.Errorf("selected CAPTCHA provider %q has no saved settings", provider)
				}
			}
			return nil
		},
	}
}

func mergeMissingCaptchaProfile(target *domain.CaptchaProviderSettings, fallback domain.CaptchaProviderSettings) {
	if target.SiteKey == "" {
		target.SiteKey = fallback.SiteKey
	}
	if target.ChallengeURL == "" {
		target.ChallengeURL = fallback.ChallengeURL
	}
	if target.VerifyURL == "" {
		target.VerifyURL = fallback.VerifyURL
	}
	if target.Secret == "" {
		target.Secret = fallback.Secret
	}
	target.Configured = target.Configured || fallback.Configured || target.Secret != ""
}
