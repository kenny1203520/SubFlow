package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"

	validation "ganigeorgiev.com/pocketbase/ozzo-validation/v4"
	"ganigeorgiev.com/pocketbase/pocketbase/core"
	"ganigeorgiev.com/pocketbase/pocketbase/tools/router"

	"subflow/internal/adapters"
	"subflow/internal/application"
	"subflow/internal/publicurl"
)

type pocketBaseSettingsAuditSummary struct {
	ActorType     string   `json:"actorType"`
	ActorID       string   `json:"actorId,omitempty"`
	ChangedFields []string `json:"changedFields"`
	SecretChanged bool     `json:"secretChanged"`
	FailureClass  string   `json:"failureClass,omitempty"`
}

type settingsAuditRequestStateKey struct{}

type settingsAuditRequestState struct{ handled bool }

func registerPocketBaseSettingsAudit(app core.App, encryptionKey string) {
	app.OnSettingsUpdateRequest().BindFunc(func(event *core.SettingsUpdateRequestEvent) error {
		if state, ok := event.Request.Context().Value(settingsAuditRequestStateKey{}).(*settingsAuditRequestState); ok {
			state.handled = true
		}
		original := event.App
		defer func() { event.App = original }()
		ctx := application.WithAuditRequestMeta(event.Request.Context(), application.AuditRequestMeta{IP: event.RealIP(), UserAgent: event.Request.UserAgent()})
		actorType, actorID, auditActorID := pocketBaseAuditActor(event.RequestEvent)

		if parsed, err := publicurl.Parse(event.NewSettings.Meta.AppURL); err == nil {
			event.NewSettings.Meta.AppURL = parsed.String()
		} else if event.NewSettings.Meta.AppURL != event.OldSettings.Meta.AppURL {
			changedFields, secretChanged := changedSettingNames(event.OldSettings, event.NewSettings)
			summary, marshalErr := json.Marshal(pocketBaseSettingsAuditSummary{ActorType: actorType, ActorID: actorID, ChangedFields: changedFields, SecretChanged: secretChanged, FailureClass: "application_url_invalid"})
			if marshalErr != nil {
				return event.InternalServerError("Failed to prepare settings audit.", marshalErr)
			}
			if auditErr := original.RunInTransaction(func(tx core.App) error {
				return createSettingsAudit(ctx, tx, encryptionKey, auditActorID, "failure", string(summary))
			}); auditErr != nil {
				return event.InternalServerError("Settings update audit could not be written.", auditErr)
			}
			fieldError := validation.NewError("validation_invalid_value", "Use an absolute HTTP or HTTPS URL without credentials, query, fragment, or path.")
			return event.BadRequestError("Application URL is invalid.", validation.Errors{
				"meta": validation.Errors{"appURL": fieldError},
			})
		}

		changedFields, secretChanged := changedSettingNames(event.OldSettings, event.NewSettings)
		summary := pocketBaseSettingsAuditSummary{ActorType: actorType, ActorID: actorID, ChangedFields: changedFields, SecretChanged: secretChanged}
		encoded, err := json.Marshal(summary)
		if err != nil {
			return event.InternalServerError("Failed to prepare settings audit.", err)
		}

		var successAuditErr error
		transactionErr := original.RunInTransaction(func(tx core.App) error {
			event.App = tx
			if err := event.Next(); err != nil {
				return err
			}
			if err := createSettingsAudit(ctx, tx, encryptionKey, auditActorID, "success", string(encoded)); err != nil {
				successAuditErr = err
				return err
			}
			return nil
		})
		if transactionErr == nil {
			return nil
		}

		failureClass := pocketBaseFailureClass(transactionErr, http.StatusInternalServerError)
		if successAuditErr != nil {
			failureClass = "audit_write_failed"
		}
		failureSummary, marshalErr := json.Marshal(pocketBaseSettingsAuditSummary{ActorType: actorType, ActorID: actorID, ChangedFields: changedFields, SecretChanged: secretChanged, FailureClass: failureClass})
		if marshalErr != nil {
			return event.InternalServerError("Failed to prepare settings audit.", errors.Join(transactionErr, marshalErr))
		}
		failureAuditErr := original.RunInTransaction(func(tx core.App) error {
			return createSettingsAudit(ctx, tx, encryptionKey, auditActorID, "failure", string(failureSummary))
		})
		if failureAuditErr != nil {
			return event.InternalServerError("Settings update audit could not be written.", errors.Join(transactionErr, failureAuditErr))
		}
		if successAuditErr != nil {
			return event.InternalServerError("Settings update was rolled back because its audit record could not be written.", successAuditErr)
		}
		return transactionErr
	})
	app.OnServe().BindFunc(func(serve *core.ServeEvent) error {
		serve.Router.BindFunc(func(event *core.RequestEvent) error {
			if event.Request.Method == http.MethodPatch && event.Request.URL.Path == "/api/settings" {
				return auditPocketBaseSettingsUpdate(event, encryptionKey)
			}
			if event.Request.Method == http.MethodPost && event.Request.URL.Path == "/api/settings/test/email" {
				return auditPocketBaseTestEmail(event, encryptionKey)
			}
			return event.Next()
		})
		return serve.Next()
	})
}

func auditPocketBaseSettingsUpdate(event *core.RequestEvent, encryptionKey string) error {
	state := &settingsAuditRequestState{}
	event.Request = event.Request.WithContext(context.WithValue(event.Request.Context(), settingsAuditRequestStateKey{}, state))
	originalResponse := event.Response
	bufferedResponse := httptest.NewRecorder()
	event.Response = bufferedResponse
	requestErr := event.Next()
	event.Response = originalResponse
	if !state.handled && (requestErr != nil || bufferedResponse.Code >= http.StatusBadRequest) {
		class := pocketBaseFailureClass(requestErr, bufferedResponse.Code)
		if auditErr := recordPocketBaseSettingsFailureAudit(event, encryptionKey, class); auditErr != nil {
			return event.InternalServerError("Settings update audit could not be written.", errors.Join(requestErr, auditErr))
		}
	}
	if requestErr != nil {
		return requestErr
	}
	replayResponse(originalResponse, bufferedResponse)
	return nil
}

func auditPocketBaseTestEmail(event *core.RequestEvent, encryptionKey string) error {
	ctx := application.WithAuditRequestMeta(event.Request.Context(), application.AuditRequestMeta{IP: event.RealIP(), UserAgent: event.Request.UserAgent()})
	if err := recordPocketBaseTestEmailAudit(ctx, event, encryptionKey, "attempt"); err != nil {
		return event.InternalServerError("Test email audit could not be written.", err)
	}

	originalResponse := event.Response
	bufferedResponse := httptest.NewRecorder()
	event.Response = bufferedResponse
	requestErr := event.Next()
	event.Response = originalResponse
	responseStatus := bufferedResponse.Code
	outcome := "success"
	failureClass := ""
	if requestErr != nil || responseStatus >= http.StatusBadRequest {
		outcome = "failure"
		failureClass = pocketBaseFailureClass(requestErr, responseStatus)
	}
	if err := recordPocketBaseTestEmailAudit(ctx, event, encryptionKey, outcome, failureClass); err != nil {
		message := "Test email result audit could not be written."
		if outcome == "success" {
			message = "郵件可能已寄出，稽核記錄失敗"
		}
		return event.InternalServerError(message, errors.Join(requestErr, err))
	}
	if requestErr != nil {
		return sanitizePocketBaseTestEmailError(event, requestErr)
	}
	replayResponse(originalResponse, bufferedResponse)
	return nil
}

func sanitizePocketBaseTestEmailError(event *core.RequestEvent, requestErr error) error {
	var apiErr *router.ApiError
	if errors.As(requestErr, &apiErr) && apiErr.Status == http.StatusBadRequest && apiErr.RawData() == nil {
		// PocketBase includes raw SMTP errors in the public message. Keep its
		// established 400 response while ensuring SMTP credentials or server
		// diagnostics cannot be reflected back to the caller.
		return event.BadRequestError("Failed to send the test email.", nil)
	}
	return requestErr
}

func replayResponse(destination http.ResponseWriter, buffered *httptest.ResponseRecorder) {
	for name, values := range buffered.Header() {
		destination.Header()[name] = append([]string(nil), values...)
	}
	status := buffered.Code
	if status == 0 {
		status = http.StatusOK
	}
	destination.WriteHeader(status)
	_, _ = destination.Write(buffered.Body.Bytes())
}

func pocketBaseFailureClass(err error, status int) string {
	var apiErr *router.ApiError
	if errors.As(err, &apiErr) {
		status = apiErr.Status
		if status == http.StatusBadRequest && apiErr.RawData() == nil {
			return pocketBaseSMTPFailureClass(apiErr.Message)
		}
	}
	switch status {
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusBadRequest:
		return "request_invalid"
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return "request_failed"
	default:
		if err != nil || status >= http.StatusBadRequest {
			return "request_failed"
		}
		return "smtp_rejected"
	}
}

func pocketBaseSMTPFailureClass(message string) string {
	message = strings.ToLower(message)
	for _, indicator := range []string{"dial tcp", "connection refused", "connection reset", "connection timed out", "network is unreachable", "no such host"} {
		if strings.Contains(message, indicator) {
			return "smtp_connection_failed"
		}
	}
	return "smtp_rejected"
}

func recordPocketBaseSettingsFailureAudit(event *core.RequestEvent, encryptionKey, failureClass string) error {
	actorType, actorID, auditActorID := pocketBaseAuditActor(event)
	summary, err := json.Marshal(pocketBaseSettingsAuditSummary{ActorType: actorType, ActorID: actorID, ChangedFields: []string{}, FailureClass: failureClass})
	if err != nil {
		return err
	}
	ctx := application.WithAuditRequestMeta(event.Request.Context(), application.AuditRequestMeta{IP: event.RealIP(), UserAgent: event.Request.UserAgent()})
	return createSettingsAudit(ctx, event.App, encryptionKey, auditActorID, "failure", string(summary))
}

func pocketBaseAuditActor(event *core.RequestEvent) (actorType, actorID, auditActorID string) {
	if event != nil && event.Auth != nil {
		if event.Auth.IsSuperuser() {
			return "pocketbase_superuser", event.Auth.Id, ""
		}
		return "pocketbase_user", event.Auth.Id, event.Auth.Id
	}
	if event != nil && event.App != nil && event.Request != nil {
		token := strings.TrimSpace(event.Request.Header.Get("Authorization"))
		if len(token) > 7 && strings.EqualFold(token[:7], "Bearer ") {
			token = strings.TrimSpace(token[7:])
		}
		if token != "" {
			if record, err := event.App.FindAuthRecordByToken(token, core.TokenTypeAuth); err == nil && record != nil {
				if record.IsSuperuser() {
					return "pocketbase_superuser", record.Id, ""
				}
				return "pocketbase_user", record.Id, record.Id
			}
		}
	}
	return "unauthenticated", "", ""
}

func changedSettingNames(oldSettings, newSettings any) ([]string, bool) {
	oldJSON, _ := json.Marshal(oldSettings)
	newJSON, _ := json.Marshal(newSettings)
	var oldValues, newValues map[string]json.RawMessage
	_ = json.Unmarshal(oldJSON, &oldValues)
	_ = json.Unmarshal(newJSON, &newValues)
	changedSet := make(map[string]struct{})
	for name, oldValue := range oldValues {
		if string(oldValue) != string(newValues[name]) {
			changedSet[name] = struct{}{}
		}
	}
	for name := range newValues {
		if _, exists := oldValues[name]; !exists {
			changedSet[name] = struct{}{}
		}
	}
	secretChanged := sensitiveSettingsValuesChanged(oldJSON, newJSON)
	if before, ok := oldSettings.(*core.Settings); ok {
		if after, ok := newSettings.(*core.Settings); ok {
			if before.SMTP.Password != after.SMTP.Password {
				secretChanged = true
				changedSet["smtp"] = struct{}{}
			}
			if before.S3.Secret != after.S3.Secret {
				secretChanged = true
				changedSet["s3"] = struct{}{}
			}
			if before.Backups.S3.Secret != after.Backups.S3.Secret {
				secretChanged = true
				changedSet["backups"] = struct{}{}
			}
		}
	}
	changed := make([]string, 0, len(changedSet))
	for name := range changedSet {
		changed = append(changed, name)
	}
	sort.Strings(changed)
	return changed, secretChanged
}

func sensitiveSettingsValuesChanged(oldJSON, newJSON []byte) bool {
	var oldValue, newValue any
	if json.Unmarshal(oldJSON, &oldValue) != nil || json.Unmarshal(newJSON, &newValue) != nil {
		return false
	}
	return sensitiveValueChanged(oldValue, newValue)
}

func sensitiveValueChanged(oldValue, newValue any) bool {
	switch before := oldValue.(type) {
	case map[string]any:
		after, ok := newValue.(map[string]any)
		if !ok {
			return containsSensitiveValue(before)
		}
		for name, value := range before {
			other, exists := after[name]
			if isSensitiveSettingName(name) && (!exists || !reflect.DeepEqual(value, other)) {
				return true
			}
			if exists && sensitiveValueChanged(value, other) {
				return true
			}
		}
		for name := range after {
			if _, exists := before[name]; !exists && isSensitiveSettingName(name) {
				return true
			}
		}
	case []any:
		after, ok := newValue.([]any)
		for index, value := range before {
			if !ok || index >= len(after) {
				return containsSensitiveValue(value)
			}
			if sensitiveValueChanged(value, after[index]) {
				return true
			}
		}
	}
	return false
}

func containsSensitiveValue(value any) bool {
	switch item := value.(type) {
	case map[string]any:
		for name, child := range item {
			if isSensitiveSettingName(name) || containsSensitiveValue(child) {
				return true
			}
		}
	case []any:
		for _, child := range item {
			if containsSensitiveValue(child) {
				return true
			}
		}
	}
	return false
}

func isSensitiveSettingName(name string) bool {
	switch strings.ToLower(name) {
	case "password", "secret", "clientsecret", "accesskey", "secretkey":
		return true
	default:
		return false
	}
}

func createSettingsAudit(ctx context.Context, app core.App, encryptionKey, actorID, outcome, summary string) error {
	stores, err := adapters.New("pocketbase", app)
	if err != nil {
		return err
	}
	return application.New(stores, encryptionKey).RecordAudit(ctx, actorID, "", "pocketbase.settings.updated", "pocketbase_settings", "primary", outcome, summary)
}

func recordPocketBaseTestEmailAudit(ctx context.Context, event *core.RequestEvent, encryptionKey, outcome string, failureClasses ...string) error {
	actorType, actorID, auditActorID := pocketBaseAuditActor(event)
	failureClass := ""
	if len(failureClasses) > 0 {
		failureClass = failureClasses[0]
	}
	summary, err := json.Marshal(pocketBaseSettingsAuditSummary{ActorType: actorType, ActorID: actorID, ChangedFields: []string{"test_email"}, FailureClass: failureClass})
	if err != nil {
		return err
	}
	stores, err := adapters.New("pocketbase", event.App)
	if err != nil {
		return err
	}
	return application.New(stores, encryptionKey).RecordAudit(ctx, auditActorID, "", "pocketbase.settings.test_email", "pocketbase_settings", "email_test", outcome, string(summary))
}
