package application

import (
	"testing"

	"subflow/internal/domain"
)

func TestSetupInputValidation(t *testing.T) {
	valid := domain.SetupInput{AdminName: "Admin", Email: "admin@example.com", Password: "correct-horse", SiteName: "SubFlow", DefaultTimezone: "Asia/Taipei", DefaultCurrency: domain.CurrencyTWD}
	if !validSetup(valid) {
		t.Fatal("expected a complete setup input to be valid")
	}
	valid.DefaultTimezone = "not/a-timezone"
	if validSetup(valid) {
		t.Fatal("expected an invalid time zone to be rejected")
	}
}

func TestSetupSecretComparison(t *testing.T) {
	if !equalSecret("deployment-secret", "deployment-secret") {
		t.Fatal("expected matching setup secret")
	}
	if equalSecret("deployment-secret", "incorrect") {
		t.Fatal("expected non-matching setup secret")
	}
	if equalSecret("", "") {
		t.Fatal("empty deployment secret must never enable setup")
	}
}
