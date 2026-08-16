package application

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	altcha "github.com/altcha-org/altcha-lib-go"

	"subflow/internal/adapters"
	"subflow/internal/captcha"
	"subflow/internal/domain"
	"subflow/internal/security"
)

// fakeSettingsStore is a minimal in-memory ports.SystemSettingsRepository,
// letting VerifyCaptcha's flow-gating logic be tested without spinning up a
// full PocketBase test app.
type fakeSettingsStore struct{ value domain.SystemSettings }

func (f *fakeSettingsStore) Get(context.Context) (domain.SystemSettings, error) { return f.value, nil }
func (f *fakeSettingsStore) Save(_ context.Context, value domain.SystemSettings) error {
	f.value = value
	return nil
}

func newCaptchaTestService(t *testing.T, flows domain.CaptchaFlowSettings) *Service {
	t.Helper()
	settings := domain.SystemSettings{
		CaptchaProvider:         "altcha_community",
		CaptchaSecretCiphertext: "plain:test-secret",
		CaptchaFlows:            flows,
	}
	store := &fakeSettingsStore{value: settings}
	return &Service{
		Stores:  adapters.Stores{Settings: store},
		Captcha: captcha.NewVerifier(),
		Cipher:  security.NewSettingsCipher(""),
	}
}

func solvedAltchaToken(t *testing.T) string {
	t.Helper()
	v := captcha.NewVerifier()
	challenge, err := v.CreateCommunityChallenge("test-secret", "authentication")
	if err != nil {
		t.Fatal(err)
	}
	solution, err := altcha.SolveChallenge(challenge.Challenge, challenge.Salt, altcha.Algorithm(challenge.Algorithm), int(challenge.MaxNumber), 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	payload := altcha.Payload{Algorithm: challenge.Algorithm, Challenge: challenge.Challenge, Number: int64(solution.Number), Salt: challenge.Salt, Signature: challenge.Signature}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(encoded)
}

// A flow left disabled must be skipped entirely, even with a garbage token
// and even though a provider is globally configured -- this is the whole
// point of per-flow gating layered on top of the pre-existing global switch.
func TestVerifyCaptchaSkipsDisabledFlow(t *testing.T) {
	service := newCaptchaTestService(t, domain.CaptchaFlowSettings{
		Login: domain.CaptchaFlowConfig{Enabled: false, Trigger: "load", Mode: "interactive"},
	})
	if err := service.VerifyCaptcha(context.Background(), domain.CaptchaFlowLogin, "not-a-real-token", ""); err != nil {
		t.Fatalf("expected a disabled flow to no-op regardless of token validity, got %v", err)
	}
}

// An enabled flow must still fully verify the token against the real
// provider -- disabling one flow must never accidentally weaken another.
func TestVerifyCaptchaChecksEnabledFlow(t *testing.T) {
	service := newCaptchaTestService(t, domain.CaptchaFlowSettings{
		Register: domain.CaptchaFlowConfig{Enabled: true, Trigger: "load", Mode: "interactive"},
	})
	ctx := context.Background()
	if err := service.VerifyCaptcha(ctx, domain.CaptchaFlowRegister, "not-a-real-token", ""); err == nil {
		t.Fatal("expected an enabled flow to reject a garbage token")
	}
	if err := service.VerifyCaptcha(ctx, domain.CaptchaFlowRegister, solvedAltchaToken(t), ""); err != nil {
		t.Fatalf("expected an enabled flow to accept a genuinely solved token, got %v", err)
	}
}

// An unrecognized flow identifier must fail safe (treated as disabled --
// see CaptchaFlowSettings.For's default case) rather than accidentally
// bypassing every flow's gate or panicking.
func TestVerifyCaptchaTreatsUnknownFlowAsDisabled(t *testing.T) {
	service := newCaptchaTestService(t, domain.CaptchaFlowSettings{})
	if err := service.VerifyCaptcha(context.Background(), "not-a-real-flow", "not-a-real-token", ""); err != nil {
		t.Fatalf("expected an unknown flow to no-op safely, got %v", err)
	}
}

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
