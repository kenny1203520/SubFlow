package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ganigeorgiev.com/pocketbase/pocketbase/apis"
	"ganigeorgiev.com/pocketbase/pocketbase/core"
	"ganigeorgiev.com/pocketbase/pocketbase/tests"

	"subflow/internal/adapters"
	pbadapter "subflow/internal/adapters/pocketbase"
	"subflow/internal/application"
	"subflow/internal/ports"
	"subflow/internal/realtime"
	"subflow/internal/upgrade"
)

func TestAccountSecurityLimiterEnforcesWindow(t *testing.T) {
	limiter := &accountSecurityLimiter{}
	now := time.Date(2026, time.November, 12, 12, 0, 0, 0, time.UTC)
	if !limiter.allow("ip", now, 2, time.Minute) || !limiter.allow("ip", now.Add(time.Second), 2, time.Minute) {
		t.Fatal("the first two requests should be allowed")
	}
	if limiter.allow("ip", now.Add(2*time.Second), 2, time.Minute) {
		t.Fatal("request over the limit was allowed")
	}
	if !limiter.allow("ip", now.Add(2*time.Minute), 2, time.Minute) {
		t.Fatal("request after the window should be allowed")
	}
}

func TestAccountSecurityLimiterCapsDistinctKeysWithLRUEviction(t *testing.T) {
	limiter := &accountSecurityLimiter{}
	now := time.Date(2026, time.November, 12, 12, 0, 0, 0, time.UTC)
	for index := range maxAccountSecurityLimiterKeys {
		if !limiter.allow(fmt.Sprintf("ip-%d", index), now, 5, time.Minute) {
			t.Fatalf("distinct key %d was rejected before reaching the cap", index)
		}
	}
	if !limiter.allow("ip-0", now.Add(time.Second), 5, time.Minute) {
		t.Fatal("existing key was rejected before reaching its rate limit")
	}
	if !limiter.allow("one-more-ip", now.Add(2*time.Second), 5, time.Minute) {
		t.Fatal("limiter rejected a new key after reaching its memory cap")
	}
	if got := len(limiter.attempts); got != maxAccountSecurityLimiterKeys {
		t.Fatalf("tracked keys = %d, want cap %d", got, maxAccountSecurityLimiterKeys)
	}
	if _, exists := limiter.attempts["ip-0"]; !exists {
		t.Fatal("recently used key was evicted")
	}
	if _, exists := limiter.attempts["ip-1"]; exists {
		t.Fatal("least recently used key was not evicted")
	}
	if _, exists := limiter.attempts["one-more-ip"]; !exists {
		t.Fatal("new key was not retained")
	}
}

func TestIsUsersAuthRouteRequiresExactUsersEndpoint(t *testing.T) {
	if !isUsersAuthRoute("/api/collections/users/request-password-reset", "request-password-reset") {
		t.Fatal("expected exact users password reset endpoint to match")
	}
	for _, path := range []string{
		"/api/collections/other/request-password-reset",
		"/api/collections/users/nested/request-password-reset",
		"/api/collections/users/request-password-reset/extra",
	} {
		if isUsersAuthRoute(path, "request-password-reset") {
			t.Fatalf("unexpected path match: %s", path)
		}
	}
}

func TestMFAChallengeEmailRequiresLiveFirstFactorChallenge(t *testing.T) {
	app, user, challenge := newAccountSecurityMFAFixture(t)
	if got, err := mfaChallengeEmail(app, challenge.Id); err != nil || got != user.Email() {
		t.Fatalf("valid challenge email = %q, %v", got, err)
	}

	challenge.SetMethod(core.MFAMethodOTP)
	if err := app.Save(challenge); err != nil {
		t.Fatal(err)
	}
	if _, err := mfaChallengeEmail(app, challenge.Id); err == nil {
		t.Fatal("standalone OTP challenge must not be accepted as an MFA first factor")
	}

	challenge.SetMethod(core.MFAMethodPassword)
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	users.MFA = core.MFAConfig{Enabled: false, Duration: 0}
	if err := app.Save(users); err != nil {
		t.Fatal(err)
	}
	if _, err := mfaChallengeEmail(app, challenge.Id); err == nil {
		t.Fatal("challenge should expire at the configured zero-second lifetime")
	}
}

func TestNativeAuthAndConfirmationErrorsArePreserved(t *testing.T) {
	h := newAccountSecurityHarness(t, true)
	user := createAccountSecurityUser(t, h.app, "error-path@example.test")

	badCredentials := h.request(http.MethodPost, "/api/collections/users/auth-with-password", map[string]any{
		"identity": user.Email(),
		"password": "incorrect-password",
	})
	if badCredentials.Code < http.StatusBadRequest || badCredentials.Body.Len() == 0 {
		t.Fatalf("bad credentials response = %d %q, want an error response", badCredentials.Code, badCredentials.Body.String())
	}

	invalidReset := h.request(http.MethodPost, "/api/collections/users/confirm-password-reset", map[string]any{
		"token": "invalid-reset-token", "password": "new-local-password", "passwordConfirm": "new-local-password",
	})
	if invalidReset.Code < http.StatusBadRequest || invalidReset.Body.Len() == 0 {
		t.Fatalf("invalid reset token response = %d %q, want an error response", invalidReset.Code, invalidReset.Body.String())
	}

	invalidEmailChange := h.request(http.MethodPost, "/api/collections/users/confirm-email-change", map[string]any{
		"token": "invalid-email-change-token", "password": "original-local-password",
	})
	if invalidEmailChange.Code < http.StatusBadRequest || invalidEmailChange.Body.Len() == 0 {
		t.Fatalf("invalid email-change token response = %d %q, want an error response", invalidEmailChange.Code, invalidEmailChange.Body.String())
	}
}

func TestNativePasswordResetKeepsGenericResponsesAndIssuedLinksValidAfterDisable(t *testing.T) {
	h := newAccountSecurityHarness(t, true)
	user := createAccountSecurityUser(t, h.app, "reset-user@example.test")

	unknown := h.request(http.MethodPost, "/api/collections/users/request-password-reset", map[string]any{"email": "unknown@example.test"})
	known := h.request(http.MethodPost, "/api/collections/users/request-password-reset", map[string]any{"email": user.Email()})
	if unknown.Code != http.StatusNoContent || known.Code != http.StatusNoContent || unknown.Body.String() != known.Body.String() {
		t.Fatalf("unknown and known reset responses differ: unknown=%d %q known=%d %q", unknown.Code, unknown.Body.String(), known.Code, known.Body.String())
	}

	var token string
	select {
	case token = <-h.resetTokens:
	case <-time.After(2 * time.Second):
		t.Fatal("PocketBase did not invoke the reset email hook")
	}
	if token == "" {
		t.Fatal("reset email did not include a token")
	}
	var resetMail accountSecurityMail
	select {
	case resetMail = <-h.resetMails:
	case <-time.After(time.Second):
		t.Fatal("reset email content was not captured")
	}
	if resetMail.email != user.Email() || !strings.Contains(resetMail.html, "https://subflow.example.test/auth/reset-password#"+token) || strings.Contains(resetMail.html, "/_/#/auth/confirm-password-reset/") {
		t.Fatalf("reset email link was not rewritten safely: recipient=%q html=%q", resetMail.email, resetMail.html)
	}
	time.Sleep(25 * time.Millisecond)

	cooledDown := h.request(http.MethodPost, "/api/collections/users/request-password-reset", map[string]any{"email": user.Email()})
	if cooledDown.Code != http.StatusNoContent || cooledDown.Body.Len() != 0 {
		t.Fatalf("cooldown response = %d %q, want generic 204", cooledDown.Code, cooledDown.Body.String())
	}

	h.setResetEnabled(t, false)
	disabled := h.request(http.MethodPost, "/api/collections/users/request-password-reset", map[string]any{"email": user.Email()})
	if disabled.Code != http.StatusNoContent || disabled.Body.Len() != 0 {
		t.Fatalf("disabled reset response = %d %q, want generic 204", disabled.Code, disabled.Body.String())
	}

	confirmed := h.request(http.MethodPost, "/api/collections/users/confirm-password-reset", map[string]any{
		"token": token, "password": "new-local-password", "passwordConfirm": "new-local-password",
	})
	if confirmed.Code != http.StatusNoContent {
		t.Fatalf("already issued reset token stopped working after disable: %d %s", confirmed.Code, confirmed.Body.String())
	}
	updated, err := h.app.FindRecordById("users", user.Id)
	if err != nil || !updated.ValidatePassword("new-local-password") {
		t.Fatalf("password reset did not persist: %v", err)
	}
}

func TestEmailChangeConfirmationIsTransactionalAndNotifiesPreviousAddress(t *testing.T) {
	h := newAccountSecurityHarness(t, true)
	user := createAccountSecurityUser(t, h.app, "old-address@example.test")
	authToken, err := user.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	requested := h.requestWithToken(http.MethodPost, "/api/collections/users/request-email-change", map[string]any{"newEmail": "new-address@example.test"}, authToken)
	if requested.Code != http.StatusNoContent {
		t.Fatalf("email change request = %d %s", requested.Code, requested.Body.String())
	}
	var confirmation accountSecurityMail
	select {
	case confirmation = <-h.emailMails:
	case <-time.After(2 * time.Second):
		t.Fatal("email-change confirmation was not sent")
	}
	if confirmation.email != "new-address@example.test" || !strings.Contains(confirmation.html, "https://subflow.example.test/auth/confirm-email-change#"+confirmation.token) || strings.Contains(confirmation.html, "/_/#/auth/confirm-email-change/") {
		t.Fatalf("email-change link was not rewritten safely: recipient=%q html=%q", confirmation.email, confirmation.html)
	}

	confirmed := h.request(http.MethodPost, "/api/collections/users/confirm-email-change", map[string]any{"token": confirmation.token, "password": "original-local-password"})
	if confirmed.Code != http.StatusNoContent {
		t.Fatalf("email-change confirmation = %d %s", confirmed.Code, confirmed.Body.String())
	}
	updated, err := h.app.FindRecordById("users", user.Id)
	if err != nil || updated.Email() != "new-address@example.test" || !updated.Verified() {
		t.Fatalf("email change did not persist and verify the new address: record=%v err=%v", updated, err)
	}
	jobs, err := h.app.FindRecordsByFilter(pbadapter.CollectionNotificationEmailOutbox, "user={:user}", "", 10, 0, map[string]any{"user": user.Id})
	if err != nil || len(jobs) != 1 || jobs[0].GetString("recipient_email") != "old-address@example.test" {
		t.Fatalf("previous-address alert outbox = %#v, err=%v", jobs, err)
	}
	entries, err := adapters.New("pocketbase", h.app)
	if err != nil {
		t.Fatal(err)
	}
	logs, err := entries.Audits.List(context.Background(), "", ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 20}, Action: "account.email.changed"})
	if err != nil || len(logs.Items) != 1 || logs.Items[0].Outcome != "success" {
		t.Fatalf("email-change confirmation audit = %#v, err=%v", logs.Items, err)
	}
}

func TestMFAOTPRequestUsesBoundAccountAndRejectsStandaloneChallengeID(t *testing.T) {
	h := newAccountSecurityHarness(t, true)
	users, err := h.app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	users.OTP.Enabled = true
	users.MFA = core.MFAConfig{Enabled: true, Duration: 300}
	if err := h.app.Save(users); err != nil {
		t.Fatal(err)
	}
	user := createAccountSecurityUser(t, h.app, "mfa-challenge@example.test")
	challenge := core.NewMFA(h.app)
	challenge.SetCollectionRef(users.Id)
	challenge.SetRecordRef(user.Id)
	challenge.SetMethod(core.MFAMethodPassword)
	if err := h.app.Save(challenge); err != nil {
		t.Fatal(err)
	}

	response := h.request(http.MethodPost, "/api/collections/users/request-otp", map[string]any{"mfaId": challenge.Id})
	if response.Code != http.StatusOK {
		t.Fatalf("MFA OTP request = %d %s", response.Code, response.Body.String())
	}
	var otpMail accountSecurityMail
	select {
	case otpMail = <-h.otpMails:
	case <-time.After(2 * time.Second):
		t.Fatal("MFA OTP was not sent")
	}
	if otpMail.email != user.Email() || otpMail.otpID == "" || otpMail.otpPassword == "" {
		t.Fatalf("MFA OTP was not sent to the challenge-bound user: %#v", otpMail)
	}

	standalone := core.NewMFA(h.app)
	standalone.SetCollectionRef(users.Id)
	standalone.SetRecordRef(user.Id)
	standalone.SetMethod(core.MFAMethodOTP)
	if err := h.app.Save(standalone); err != nil {
		t.Fatal(err)
	}
	rejected := h.request(http.MethodPost, "/api/collections/users/request-otp", map[string]any{"mfaId": standalone.Id})
	if rejected.Code != http.StatusBadRequest {
		t.Fatalf("standalone OTP challenge was accepted as an MFA first factor: %d %s", rejected.Code, rejected.Body.String())
	}
}

func newAccountSecurityMFAFixture(t *testing.T) (*tests.TestApp, *core.Record, *core.MFA) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	if err := pbadapter.EnsureSchema(app); err != nil {
		t.Fatal(err)
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	users.MFA = core.MFAConfig{Enabled: true, Duration: 300}
	if err := app.Save(users); err != nil {
		t.Fatal(err)
	}
	user := createAccountSecurityUser(t, app, "mfa-user@example.test")
	challenge := core.NewMFA(app)
	challenge.SetCollectionRef(users.Id)
	challenge.SetRecordRef(user.Id)
	challenge.SetMethod(core.MFAMethodPassword)
	if err := app.Save(challenge); err != nil {
		t.Fatal(err)
	}
	return app, user, challenge
}

type accountSecurityHarness struct {
	app         *tests.TestApp
	handler     http.Handler
	resetTokens chan string
	resetMails  chan accountSecurityMail
	emailMails  chan accountSecurityMail
	otpMails    chan accountSecurityMail
}

type accountSecurityMail struct {
	email       string
	token       string
	html        string
	otpID       string
	otpPassword string
}

func newAccountSecurityHarness(t *testing.T, resetEnabled bool) *accountSecurityHarness {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	if err := pbadapter.EnsureSchema(app); err != nil {
		t.Fatal(err)
	}
	setupLink := ""
	migrations := upgrade.Migrations("pocketbase", "https://subflow.example.test", &setupLink)
	if err := migrations[len(migrations)-1].Up(app); err != nil {
		t.Fatal(err)
	}
	settings, err := app.FindFirstRecordByFilter(pbadapter.CollectionSystemSettings, "key='primary'", nil)
	if err != nil {
		t.Fatal(err)
	}
	settings.Set("initialized", true)
	settings.Set("allow_password_reset", resetEnabled)
	if err := app.Save(settings); err != nil {
		t.Fatal(err)
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	users.PasswordAuth = core.PasswordAuthConfig{Enabled: true, IdentityFields: []string{"email"}}
	if err := app.Save(users); err != nil {
		t.Fatal(err)
	}
	app.Settings().Meta.AppURL = "https://subflow.example.test"
	app.Settings().SMTP.Enabled = true

	stores, err := adapters.New("pocketbase", app)
	if err != nil {
		t.Fatal(err)
	}
	service := application.New(stores, "test-account-security-key")
	resetTokens := make(chan string, 4)
	resetMails := make(chan accountSecurityMail, 4)
	emailMails := make(chan accountSecurityMail, 4)
	otpMails := make(chan accountSecurityMail, 4)
	events := realtime.NewBus()
	registerAccountSecurityMailerAudit(app, service)
	app.OnMailerRecordPasswordResetSend("users").BindFunc(func(event *core.MailerRecordEvent) error {
		mail := accountSecurityMail{html: event.Message.HTML}
		mail.token, _ = event.Meta["token"].(string)
		if len(event.Message.To) > 0 {
			mail.email = event.Message.To[0].Address
		}
		resetTokens <- mail.token
		resetMails <- mail
		return nil
	})
	app.OnMailerRecordEmailChangeSend("users").BindFunc(func(event *core.MailerRecordEvent) error {
		mail := accountSecurityMail{html: event.Message.HTML}
		mail.token, _ = event.Meta["token"].(string)
		if len(event.Message.To) > 0 {
			mail.email = event.Message.To[0].Address
		}
		emailMails <- mail
		return nil
	})
	app.OnMailerRecordOTPSend("users").BindFunc(func(event *core.MailerRecordEvent) error {
		mail := accountSecurityMail{}
		mail.otpID, _ = event.Meta["otpId"].(string)
		mail.otpPassword, _ = event.Meta["password"].(string)
		if len(event.Message.To) > 0 {
			mail.email = event.Message.To[0].Address
		}
		otpMails <- mail
		return nil
	})
	registerAccountSecurityConfirmationHooks(app, service, events, "test-account-security-key")
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	serve := &core.ServeEvent{App: app, Router: router}
	registerAccountSecurityRoutes(serve, service)
	handler, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return &accountSecurityHarness{app: app, handler: handler, resetTokens: resetTokens, resetMails: resetMails, emailMails: emailMails, otpMails: otpMails}
}

func (h *accountSecurityHarness) request(method, path string, value any) *httptest.ResponseRecorder {
	return h.requestWithToken(method, path, value, "")
}

func (h *accountSecurityHarness) requestWithToken(method, path string, value any, token string) *httptest.ResponseRecorder {
	body, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", token)
	}
	response := httptest.NewRecorder()
	h.handler.ServeHTTP(response, request)
	return response
}

func (h *accountSecurityHarness) setResetEnabled(t *testing.T, enabled bool) {
	t.Helper()
	settings, err := h.app.FindFirstRecordByFilter(pbadapter.CollectionSystemSettings, "key='primary'", nil)
	if err != nil {
		t.Fatal(err)
	}
	settings.Set("allow_password_reset", enabled)
	if err := h.app.Save(settings); err != nil {
		t.Fatal(err)
	}
}

func createAccountSecurityUser(t *testing.T, app core.App, email string) *core.Record {
	t.Helper()
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	user := core.NewRecord(users)
	user.Set("email", email)
	user.Set("name", "Account Security")
	user.SetPassword("original-local-password")
	user.SetVerified(true)
	if err := app.Save(user); err != nil {
		t.Fatal(err)
	}
	return user
}

func TestAccountSecurityAuditSummaryDoesNotIncludeSensitiveRequestValues(t *testing.T) {
	secret := []string{"secret-password", "123456", "jwt-token", "https://example.test/reset#private-token"}
	summary := `{"reason":"authentication_request"}`
	for _, value := range secret {
		if strings.Contains(summary, value) {
			t.Fatalf("audit summary contains sensitive request value %q", value)
		}
	}
	if strings.Contains(summary, "email") {
		t.Fatalf("audit summary should not store the account address: %s", summary)
	}
}
