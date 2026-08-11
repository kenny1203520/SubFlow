package mailer

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"
)

func TestConfigurePocketBaseSMTP(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	smtp := &SMTP{Host: "smtp.example.com", Port: "465", User: "user", Password: "secret", From: "noreply@example.com"}
	if err = ConfigurePocketBase(app, smtp); err != nil {
		t.Fatal(err)
	}
	settings := app.Settings()
	if !settings.SMTP.Enabled || settings.SMTP.Host != smtp.Host || !settings.SMTP.TLS || settings.Meta.SenderAddress != smtp.From {
		t.Fatalf("PocketBase SMTP settings not applied: %#v", settings.SMTP)
	}
}
