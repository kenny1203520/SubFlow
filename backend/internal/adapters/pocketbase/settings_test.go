package pocketbase

import (
	"context"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// A settings record saved before per-flow captcha gating existed has no
// captcha_flows value at all. Reading it back must reproduce exactly what
// was gated before this feature shipped -- register/passwordReset/
// otpRequest gated iff a provider is configured, login never gated -- so an
// existing install sees zero behavior change until an admin explicitly
// saves new settings (see captchaFlowsFrom).
func TestSettingsFromMigratesLegacyCaptchaFlows(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	if err = EnsureSchema(app); err != nil {
		t.Fatal(err)
	}

	collection, err := app.FindCollectionByNameOrId(CollectionSystemSettings)
	if err != nil {
		t.Fatal(err)
	}
	existing, err := app.FindFirstRecordByFilter(CollectionSystemSettings, "key='primary'", nil)
	if err != nil {
		record := core.NewRecord(collection)
		record.Set("key", "primary")
		existing = record
	}
	// Simulates a pre-migration row: every legacy field set directly via the
	// raw PocketBase API (never going through SaveSystemSettings, which now
	// always writes captcha_flows), leaving captcha_flows genuinely unset.
	existing.Set("captcha_provider", "turnstile")
	existing.Set("captcha_site_key", "site-key")
	if err = app.Save(existing); err != nil {
		t.Fatal(err)
	}

	stores := NewStores(app)
	settings, err := stores.Settings.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !settings.CaptchaFlows.Register.Enabled || settings.CaptchaFlows.Register.Trigger != "load" || settings.CaptchaFlows.Register.Mode != "interactive" {
		t.Fatalf("expected register to migrate to enabled/load/interactive, got %#v", settings.CaptchaFlows.Register)
	}
	if !settings.CaptchaFlows.PasswordReset.Enabled {
		t.Fatalf("expected passwordReset to migrate to enabled, got %#v", settings.CaptchaFlows.PasswordReset)
	}
	if !settings.CaptchaFlows.OTPRequest.Enabled {
		t.Fatalf("expected otpRequest to migrate to enabled, got %#v", settings.CaptchaFlows.OTPRequest)
	}
	if settings.CaptchaFlows.Login.Enabled {
		t.Fatalf("expected login to migrate to disabled (no hook existed for it before), got %#v", settings.CaptchaFlows.Login)
	}
}

// Once an admin saves settings once, the real stored per-flow value takes
// over instead of the legacy migration fallback.
func TestSettingsFromUsesStoredCaptchaFlowsOnceSaved(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	if err = EnsureSchema(app); err != nil {
		t.Fatal(err)
	}

	stores := NewStores(app)
	ctx := context.Background()
	settings, err := stores.Settings.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	settings.SiteName = "SubFlow"
	settings.DefaultTimezone = "UTC"
	settings.DefaultCurrency = "TWD"
	settings.CaptchaProvider = "turnstile"
	settings.CaptchaFlows.Login.Enabled = true
	settings.CaptchaFlows.Login.Trigger = "submit"
	settings.CaptchaFlows.Login.Mode = "invisible"
	settings.CaptchaFlows.Register.Enabled = false
	if err = stores.Settings.Save(ctx, settings); err != nil {
		t.Fatal(err)
	}

	reloaded, err := stores.Settings.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.CaptchaFlows.Login.Enabled || reloaded.CaptchaFlows.Login.Trigger != "submit" || reloaded.CaptchaFlows.Login.Mode != "invisible" {
		t.Fatalf("expected the saved login flow config to round-trip, got %#v", reloaded.CaptchaFlows.Login)
	}
	if reloaded.CaptchaFlows.Register.Enabled {
		t.Fatalf("expected the saved (disabled) register flow config to round-trip, got %#v", reloaded.CaptchaFlows.Register)
	}
}
