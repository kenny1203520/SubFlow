package main

import (
	"context"
	"encoding/json"
	"errors"
	"sort"

	"ganigeorgiev.com/pocketbase/pocketbase"
	"ganigeorgiev.com/pocketbase/pocketbase/core"

	"subflow/internal/adapters"
	"subflow/internal/application"
)

type pocketBaseSettingsAuditSummary struct {
	ActorType     string   `json:"actorType"`
	ActorID       string   `json:"actorId,omitempty"`
	ChangedFields []string `json:"changedFields"`
	SecretChanged bool     `json:"secretChanged"`
}

func registerPocketBaseSettingsAudit(app *pocketbase.PocketBase, encryptionKey string) {
	app.OnSettingsUpdateRequest().BindFunc(func(event *core.SettingsUpdateRequestEvent) error {
		original := event.App
		ctx := application.WithAuditRequestMeta(event.Request.Context(), application.AuditRequestMeta{IP: event.RealIP(), UserAgent: event.Request.UserAgent()})
		actorID := ""
		if event.Auth != nil {
			actorID = event.Auth.Id
		}
		changedFields, secretChanged := changedSettingNames(event.OldSettings, event.NewSettings)
		summary := pocketBaseSettingsAuditSummary{ActorType: "pocketbase_superuser", ActorID: actorID, ChangedFields: changedFields, SecretChanged: secretChanged}
		encoded, err := json.Marshal(summary)
		if err != nil {
			return event.InternalServerError("Failed to prepare settings audit.", err)
		}
		transactionErr := original.RunInTransaction(func(tx core.App) error {
			event.App = tx
			if err := event.Next(); err != nil {
				return err
			}
			return createSettingsAudit(ctx, tx, encryptionKey, actorID, "success", string(encoded))
		})
		event.App = original
		if transactionErr == nil {
			return nil
		}
		failureErr := original.RunInTransaction(func(tx core.App) error {
			return createSettingsAudit(ctx, tx, encryptionKey, actorID, "failure", string(encoded))
		})
		if failureErr != nil {
			return event.InternalServerError("Settings update audit could not be written.", errors.Join(transactionErr, failureErr))
		}
		return transactionErr
	})
	app.OnServe().BindFunc(func(serve *core.ServeEvent) error {
		serve.Router.BindFunc(func(event *core.RequestEvent) error {
			if event.Request.Method != "POST" || event.Request.URL.Path != "/api/settings/test/email" {
				return event.Next()
			}
			ctx := application.WithAuditRequestMeta(event.Request.Context(), application.AuditRequestMeta{IP: event.RealIP(), UserAgent: event.Request.UserAgent()})
			if err := recordPocketBaseTestEmailAudit(ctx, event, encryptionKey, "attempt"); err != nil {
				return event.InternalServerError("Test email audit could not be written.", err)
			}
			requestErr := event.Next()
			outcome := "success"
			if requestErr != nil {
				outcome = "failure"
			}
			if err := recordPocketBaseTestEmailAudit(ctx, event, encryptionKey, outcome); err != nil {
				return event.InternalServerError("Test email result audit could not be written.", errors.Join(requestErr, err))
			}
			return requestErr
		})
		return serve.Next()
	})
}

func changedSettingNames(oldSettings, newSettings any) ([]string, bool) {
	oldJSON, _ := json.Marshal(oldSettings)
	newJSON, _ := json.Marshal(newSettings)
	var oldValues, newValues map[string]json.RawMessage
	_ = json.Unmarshal(oldJSON, &oldValues)
	_ = json.Unmarshal(newJSON, &newValues)
	changed := make([]string, 0)
	secretChanged := false
	for name, oldValue := range oldValues {
		if string(oldValue) != string(newValues[name]) {
			changed = append(changed, name)
			if name == "smtp" || name == "oauth2" || name == "s3" {
				secretChanged = true
			}
		}
	}
	for name := range newValues {
		if _, exists := oldValues[name]; !exists {
			changed = append(changed, name)
		}
	}
	sort.Strings(changed)
	return changed, secretChanged
}

func createSettingsAudit(ctx context.Context, app core.App, encryptionKey, actorID, outcome, summary string) error {
	stores, err := adapters.New("pocketbase", app)
	if err != nil {
		return err
	}
	return application.New(stores, encryptionKey).RecordAudit(ctx, actorID, "", "pocketbase.settings.updated", "pocketbase_settings", "primary", outcome, summary)
}

func recordPocketBaseTestEmailAudit(ctx context.Context, event *core.RequestEvent, encryptionKey, outcome string) error {
	actorID := ""
	if event.Auth != nil {
		actorID = event.Auth.Id
	}
	summary, err := json.Marshal(pocketBaseSettingsAuditSummary{ActorType: "pocketbase_admin", ActorID: actorID, ChangedFields: []string{"test_email"}})
	if err != nil {
		return err
	}
	stores, err := adapters.New("pocketbase", event.App)
	if err != nil {
		return err
	}
	return application.New(stores, encryptionKey).RecordAudit(ctx, actorID, "", "pocketbase.settings.test_email", "pocketbase_settings", "email_test", outcome, string(summary))
}
