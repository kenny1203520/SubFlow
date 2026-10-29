package upgrade

import (
	"context"
	"encoding/json"
	"testing"

	"ganigeorgiev.com/pocketbase/pocketbase/core"

	pbadapter "subflow/internal/adapters/pocketbase"
	"subflow/internal/domain"
)

func TestCaptchaProviderSettingsMigrationPreservesCredentialsAndNormalizesLegacyAlias(t *testing.T) {
	app := newMigrationTestApp(t)
	settingsCollection, err := app.FindCollectionByNameOrId(pbadapter.CollectionSystemSettings)
	if err != nil {
		t.Fatal(err)
	}
	turnstile := core.NewRecord(settingsCollection)
	turnstile.Set("key", "legacy-turnstile")
	turnstile.Set("captcha_provider", "turnstile")
	turnstile.Set("captcha_site_key", "turnstile-site")
	turnstile.Set("captcha_secret", "ciphertext:turnstile-secret")
	turnstile.Set("captcha_flows", map[string]any{"login": map[string]any{"enabled": true}})
	if err := app.Save(turnstile); err != nil {
		t.Fatal(err)
	}
	altcha := core.NewRecord(settingsCollection)
	altcha.Set("key", "legacy-altcha")
	altcha.Set("captcha_provider", "altcha")
	altcha.Set("captcha_secret", "ciphertext:altcha-secret")
	if err := app.Save(altcha); err != nil {
		t.Fatal(err)
	}

	manager, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	setupLink := ""
	migrations := Migrations("pocketbase", "https://localhost:8090", &setupLink)
	for range 2 {
		if err := RunMigrations(context.Background(), app, manager, migrations, MigrationAudit("pocketbase", "test-only-root-key")); err != nil {
			t.Fatal(err)
		}
	}

	for _, tc := range []struct {
		key      string
		provider string
		secret   string
	}{
		{key: "legacy-turnstile", provider: "turnstile", secret: "ciphertext:turnstile-secret"},
		{key: "legacy-altcha", provider: "altcha_community", secret: "ciphertext:altcha-secret"},
	} {
		record, err := app.FindFirstRecordByFilter(pbadapter.CollectionSystemSettings, "key={:key}", map[string]any{"key": tc.key})
		if err != nil {
			t.Fatal(err)
		}
		if got := record.GetString("captcha_provider"); got != tc.provider {
			t.Fatalf("%s provider = %q, want %q", tc.key, got, tc.provider)
		}
		if !record.GetBool("captcha_enabled") {
			t.Fatalf("%s should be enabled based on its existing provider", tc.key)
		}
		var profiles map[string]domain.CaptchaProviderSettings
		if err := json.Unmarshal([]byte(record.GetString("captcha_providers")), &profiles); err != nil {
			t.Fatal(err)
		}
		profile, ok := profiles[tc.provider]
		if !ok || profile.Secret != tc.secret || !profile.Configured {
			t.Fatalf("%s provider profile lost its secret: %#v", tc.key, profiles)
		}
		if tc.provider == "turnstile" && profile.SiteKey != "turnstile-site" {
			t.Fatalf("turnstile site key = %q", profile.SiteKey)
		}
		if tc.key == "legacy-turnstile" && record.GetString("captcha_flows") == "" {
			t.Fatal("shared CAPTCHA flow settings were lost")
		}
	}

	collection, err := app.FindCollectionByNameOrId(pbadapter.CollectionSystemSettings)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := collection.Fields.GetByName("captcha_enabled").(*core.BoolField); !ok {
		t.Fatal("captcha_enabled field was not added")
	}
	if _, ok := collection.Fields.GetByName("captcha_providers").(*core.JSONField); !ok {
		t.Fatal("captcha_providers field was not added")
	}
}
