package main

import (
	"bytes"
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"ganigeorgiev.com/pocketbase/pocketbase/apis"
	"ganigeorgiev.com/pocketbase/pocketbase/core"

	"subflow/internal/application"
	"subflow/internal/domain"
	"subflow/internal/publicurl"
)

type accountSecurityLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	nodes    map[string]*list.Element
	order    *list.List
}

const maxAccountSecurityLimiterKeys = 10000

func (l *accountSecurityLimiter) allow(key string, now time.Time, max int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.attempts == nil {
		l.attempts = make(map[string][]time.Time)
		l.nodes = make(map[string]*list.Element)
		l.order = list.New()
	}
	if node, exists := l.nodes[key]; exists {
		l.order.MoveToFront(node)
	} else {
		if len(l.attempts) >= maxAccountSecurityLimiterKeys {
			oldest := l.order.Back()
			if oldest != nil {
				oldestKey := oldest.Value.(string)
				delete(l.attempts, oldestKey)
				delete(l.nodes, oldestKey)
				l.order.Remove(oldest)
			}
		}
		l.nodes[key] = l.order.PushFront(key)
	}
	cutoff := now.Add(-window)
	kept := l.attempts[key][:0]
	for _, attempt := range l.attempts[key] {
		if attempt.After(cutoff) {
			kept = append(kept, attempt)
		}
	}
	if len(kept) >= max {
		l.attempts[key] = kept
		return false
	}
	l.attempts[key] = append(kept, now)
	return true
}

func passwordResetAvailable(app core.App, service *application.Service, ctx context.Context) bool {
	settings, err := service.SetupStatus(ctx)
	if err != nil || !settings.Initialized || !settings.AllowPasswordReset || !app.Settings().SMTP.Enabled {
		return false
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil || !users.PasswordAuth.Enabled {
		return false
	}
	_, err = publicurl.Parse(app.Settings().Meta.AppURL)
	return err == nil
}

func registerAccountSecurityRoutes(serve *core.ServeEvent, service *application.Service) {
	limiter := &accountSecurityLimiter{}
	serve.Router.BindFunc(func(event *core.RequestEvent) error {
		if event.Request.Method != http.MethodPost {
			return event.Next()
		}
		path := event.Request.URL.Path
		if isUsersAuthRoute(path, "request-password-reset") {
			return handlePasswordResetRequest(event, service, limiter)
		}
		if isUsersAuthRoute(path, "request-otp") {
			return handleOTPRequest(event, service, limiter)
		}
		if isUsersAuthRoute(path, "request-email-change") {
			return auditNativeAuthRequest(event, service, "account.email.change.request", "sent")
		}
		if isUsersAuthRoute(path, "auth-with-password") {
			return auditNativeAuthRequest(event, service, "user.login.password", "success")
		}
		if isUsersAuthRoute(path, "confirm-password-reset") || isUsersAuthRoute(path, "confirm-email-change") {
			return auditNativeConfirmationFailure(event, service)
		}
		if isUsersAuthRoute(path, "auth-with-otp") {
			return auditNativeAuthRequest(event, service, "user.login.otp", "success")
		}
		return event.Next()
	})
}

func registerAccountSecurityMailerAudit(app core.App, service *application.Service) {
	app.OnMailerRecordPasswordResetSend("users").BindFunc(func(event *core.MailerRecordEvent) error {
		rewriteAccountSecurityLink(event, "/auth/reset-password#", "confirm-password-reset")
		err := event.Next()
		recordAccountMailerResult(event, service, "account.password.reset.email", err)
		return err
	})
	app.OnMailerRecordEmailChangeSend("users").BindFunc(func(event *core.MailerRecordEvent) error {
		rewriteAccountSecurityLink(event, "/auth/confirm-email-change#", "confirm-email-change")
		err := event.Next()
		recordAccountMailerResult(event, service, "account.email.change.email", err)
		return err
	})
	app.OnMailerRecordOTPSend("users").BindFunc(func(event *core.MailerRecordEvent) error {
		err := event.Next()
		recordAccountMailerResult(event, service, "user.login.otp.email", err)
		return err
	})
}

func rewriteAccountSecurityLink(event *core.MailerRecordEvent, destination, nativeRoute string) {
	token, _ := event.Meta["token"].(string)
	if token == "" || event.Message == nil {
		return
	}
	base, err := publicurl.Parse(event.App.Settings().Meta.AppURL)
	if err != nil {
		return
	}
	oldLink := base.String() + "/_/#/auth/" + nativeRoute + "/" + token
	newLink := base.String() + destination + token
	event.Message.HTML = strings.ReplaceAll(event.Message.HTML, oldLink, newLink)
}

func recordAccountMailerResult(event *core.MailerRecordEvent, service *application.Service, action string, sendErr error) {
	actorID := ""
	if event.Record != nil {
		actorID = event.Record.Id
	}
	outcome := "success"
	reason := "mail_sent"
	if sendErr != nil {
		outcome = "failure"
		reason = "mail_failed"
	}
	if err := service.RecordAudit(context.Background(), actorID, "", action, "user", actorID, outcome, `{"reason":"`+reason+`"}`); err != nil {
		event.App.Logger().Error("account security mail audit write failed", "action", action, "reason", "audit_write_failed")
	}
}

func isUsersAuthRoute(path, operation string) bool {
	return path == "/api/collections/users/"+operation
}

func handlePasswordResetRequest(event *core.RequestEvent, service *application.Service, limiter *accountSecurityLimiter) error {
	ctx := accountSecurityContext(event)
	if err := writeAccountAudit(ctx, service, "account.password.reset.request", "attempt", "password_reset"); err != nil {
		return event.InternalServerError("Authentication audit is unavailable.", err)
	}
	if !passwordResetAvailable(event.App, service, ctx) {
		if err := writeAccountAudit(ctx, service, "account.password.reset.request", "accepted", "password_reset"); err != nil {
			event.App.Logger().Error("account security result audit failed", "action", "account.password.reset.request", "reason", "audit_write_failed")
			return event.InternalServerError("Authentication audit is unavailable.", err)
		}
		return event.NoContent(http.StatusNoContent)
	}
	if err := service.VerifyCaptcha(ctx, domain.CaptchaFlowPasswordReset, event.Request.Header.Get("X-SubFlow-Captcha"), event.RealIP()); err != nil {
		return auditedAccountFailure(event, service, "account.password.reset.request", "captcha_rejected", event.BadRequestError("captcha_verification_failed", nil))
	}
	if !event.App.IsDev() && !limiter.allow("password-reset:"+event.RealIP(), time.Now().UTC(), 5, 15*time.Minute) {
		if err := writeAccountAudit(ctx, service, "account.password.reset.request", "throttled", "password_reset"); err != nil {
			event.App.Logger().Error("account security result audit failed", "action", "account.password.reset.request", "reason", "audit_write_failed")
			return event.InternalServerError("Authentication audit is unavailable.", err)
		}
		return event.NoContent(http.StatusNoContent)
	}
	return auditNativeAuthRequest(event, service, "account.password.reset.request", "accepted", true)
}

func handleOTPRequest(event *core.RequestEvent, service *application.Service, limiter *accountSecurityLimiter) error {
	ctx := accountSecurityContext(event)
	if err := writeAccountAudit(ctx, service, "user.login.otp.request", "attempt", "otp_request"); err != nil {
		return event.InternalServerError("Authentication audit is unavailable.", err)
	}
	if err := service.VerifyCaptcha(ctx, domain.CaptchaFlowOTPRequest, event.Request.Header.Get("X-SubFlow-Captcha"), event.RealIP()); err != nil {
		return auditedAccountFailure(event, service, "user.login.otp.request", "captcha_rejected", event.BadRequestError("captcha_verification_failed", nil))
	}
	if !event.App.IsDev() && !limiter.allow("otp-request:"+event.RealIP(), time.Now().UTC(), 10, 15*time.Minute) {
		if err := writeAccountAudit(ctx, service, "user.login.otp.request", "throttled", "otp_request"); err != nil {
			event.App.Logger().Error("account security result audit failed", "action", "user.login.otp.request", "reason", "audit_write_failed")
			return event.InternalServerError("Authentication audit is unavailable.", err)
		}
		return event.TooManyRequestsError("Too many verification-code requests. Try again later.", nil)
	}
	fields, err := decodeRequestFields(event.Request)
	if err != nil {
		if auditErr := writeAccountAudit(ctx, service, "user.login.otp.request", "failure", "invalid_request"); auditErr != nil {
			event.App.Logger().Error("account security result audit failed", "action", "user.login.otp.request", "reason", "audit_write_failed")
			return event.InternalServerError("Authentication audit is unavailable.", auditErr)
		}
		return event.Next()
	}
	var email, mfaID string
	_ = json.Unmarshal(fields["email"], &email)
	_ = json.Unmarshal(fields["mfaId"], &mfaID)
	if mfaID != "" {
		email, err = mfaChallengeEmail(event.App, mfaID)
		if err != nil {
			return auditedAccountFailure(event, service, "user.login.otp.request", "invalid_mfa_challenge", event.BadRequestError("Invalid or expired MFA challenge.", nil))
		}
		body, _ := json.Marshal(map[string]string{"email": email})
		event.Request.Body = io.NopCloser(bytes.NewReader(body))
	}
	return auditNativeAuthRequest(event, service, "user.login.otp.request", "accepted", true)
}

func mfaChallengeEmail(app core.App, id string) (string, error) {
	record, err := app.FindRecordById(core.CollectionNameMFAs, id)
	if err != nil {
		return "", err
	}
	mfa := &core.MFA{Record: record}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil || mfa.CollectionRef() != users.Id || mfa.HasExpired(users.MFA.DurationTime()) {
		return "", domain.ErrInvalid
	}
	if mfa.Method() != core.MFAMethodPassword && mfa.Method() != core.MFAMethodOAuth2 {
		return "", domain.ErrInvalid
	}
	user, err := app.FindRecordById(mfa.CollectionRef(), mfa.RecordRef())
	// PocketBase auth records have no active/disabled state field; MFA challenges
	// are created only after a successful first factor and are bound to this user.
	if err != nil || user.Collection().Id != users.Id || user.Email() == "" {
		return "", domain.ErrInvalid
	}
	return user.Email(), nil
}

func newAccountSecurityRecorder() *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	recorder.Code = 0
	return recorder
}

func auditNativeAuthRequest(event *core.RequestEvent, service *application.Service, action, successOutcome string, attemptAlreadyRecorded ...bool) error {
	ctx := accountSecurityContext(event)
	actorID := ""
	if event.Auth != nil {
		actorID = event.Auth.Id
	}
	if len(attemptAlreadyRecorded) == 0 || !attemptAlreadyRecorded[0] {
		if err := writeAccountAudit(ctx, service, action, "attempt", "authentication_request", actorID); err != nil {
			return event.InternalServerError("Authentication audit is unavailable.", err)
		}
	}
	originalResponse := event.Response
	buffered := newAccountSecurityRecorder()
	event.Response = buffered
	requestErr := event.Next()
	event.Response = originalResponse
	outcome := successOutcome
	if requestErr != nil || buffered.Code >= http.StatusBadRequest {
		outcome = "failure"
	}
	if errors.Is(requestErr, apis.ErrMFA) || strings.Contains(buffered.Body.String(), `"mfaId"`) {
		outcome = "mfa_required"
	}
	if event.Auth != nil {
		actorID = event.Auth.Id
	}
	if err := writeAccountAudit(ctx, service, action, outcome, "authentication_request", actorID); err != nil {
		event.App.Logger().Error("account security result audit failed", "action", action, "reason", "audit_write_failed")
		if strings.HasPrefix(action, "user.login.") {
			return event.InternalServerError("Authentication audit is unavailable.", err)
		}
	}
	if requestErr != nil && buffered.Code == 0 {
		return requestErr
	}
	replayResponse(originalResponse, buffered)
	return nil
}

func auditNativeConfirmationFailure(event *core.RequestEvent, service *application.Service) error {
	path := event.Request.URL.Path
	action := "account.password.reset.confirm"
	if strings.HasSuffix(path, "/confirm-email-change") {
		action = "account.email.change.confirm"
	}
	originalResponse := event.Response
	buffered := newAccountSecurityRecorder()
	event.Response = buffered
	requestErr := event.Next()
	event.Response = originalResponse
	if requestErr != nil || buffered.Code >= http.StatusBadRequest {
		actorID := ""
		if event.Auth != nil {
			actorID = event.Auth.Id
		}
		if err := service.RecordAudit(accountSecurityContext(event), actorID, "", action, "user", actorID, "failure", "credentials_invalid"); err != nil {
			event.App.Logger().Error("account security failure audit write failed", "action", action, "reason", "audit_write_failed")
		}
	}
	if requestErr != nil && buffered.Code == 0 {
		return requestErr
	}
	replayResponse(originalResponse, buffered)
	return nil
}

func decodeRequestFields(request *http.Request) (map[string]json.RawMessage, error) {
	body, err := io.ReadAll(io.LimitReader(request.Body, 64<<10))
	if err != nil || len(body) == 64<<10 {
		return nil, domain.ErrInvalid
	}
	request.Body = io.NopCloser(bytes.NewReader(body))
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func accountSecurityContext(event *core.RequestEvent) context.Context {
	return application.WithAuditRequestMeta(event.Request.Context(), application.AuditRequestMeta{IP: event.RealIP(), UserAgent: event.Request.UserAgent()})
}

func writeAccountAudit(ctx context.Context, service *application.Service, action, outcome, reason string, actors ...string) error {
	actorID := ""
	if len(actors) > 0 {
		actorID = actors[0]
	}
	return service.RecordAudit(ctx, actorID, "", action, "auth", actorID, outcome, `{"reason":"`+reason+`"}`)
}

func auditedAccountFailure(event *core.RequestEvent, service *application.Service, action, reason string, responseErr error) error {
	if err := writeAccountAudit(accountSecurityContext(event), service, action, "failure", reason); err != nil {
		event.App.Logger().Error("account security result audit failed", "action", action, "reason", "audit_write_failed")
		return event.InternalServerError("Authentication audit is unavailable.", err)
	}
	return responseErr
}
