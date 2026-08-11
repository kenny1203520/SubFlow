package application

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"subflow/internal/domain"
	"subflow/internal/security"
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

func TestSetupTokenComparison(t *testing.T) {
	sum := sha256.Sum256([]byte("installer-token"))
	if !equalSetupToken(fmt.Sprintf("%x", sum), "installer-token") {
		t.Fatal("expected matching installer token")
	}
	if equalSetupToken(fmt.Sprintf("%x", sum), "incorrect-token") {
		t.Fatal("expected non-matching installer token")
	}
	if equalSetupToken("", "") {
		t.Fatal("empty token hash must never enable setup")
	}
}

func TestCaptchaSecretValueSupportsPlaintextFallback(t *testing.T) {
	service := &Service{Cipher: security.NewSettingsCipher("")}
	secret, err := service.captchaSecretValue("plain:shared-secret")
	if err != nil {
		t.Fatal(err)
	}
	if secret != "shared-secret" {
		t.Fatalf("expected plaintext secret fallback, got %q", secret)
	}
}
