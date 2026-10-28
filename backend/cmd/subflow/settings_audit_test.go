package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"ganigeorgiev.com/pocketbase/pocketbase/core"
	"ganigeorgiev.com/pocketbase/pocketbase/tests"

	"subflow/internal/adapters"
	"subflow/internal/adapters/pocketbase"
	"subflow/internal/application"
	"subflow/internal/ports"
)

func TestChangedSettingNamesOnlyRecordFieldNamesAndSecretChange(t *testing.T) {
	oldSettings := map[string]any{
		"meta": map[string]any{"appName": "SubFlow"},
		"smtp": map[string]any{"password": "old-private-value"},
	}
	newSettings := map[string]any{
		"meta": map[string]any{"appName": "SubFlow Cloud"},
		"smtp": map[string]any{"password": "new-private-value"},
	}
	fields, secretChanged := changedSettingNames(oldSettings, newSettings)
	if !reflect.DeepEqual(fields, []string{"meta", "smtp"}) || !secretChanged {
		t.Fatalf("changedSettingNames = (%v, %v), want ([meta smtp], true)", fields, secretChanged)
	}
	encoded, err := json.Marshal(pocketBaseSettingsAuditSummary{ChangedFields: fields, SecretChanged: secretChanged})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "private-value") {
		t.Fatalf("settings audit summary contains a secret value: %s", encoded)
	}
}

func TestPocketBaseTestEmailAuditRecordsOutcomeWithoutRecipient(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	if err := pocketbase.EnsureSchema(app); err != nil {
		t.Fatal(err)
	}
	event := &core.RequestEvent{App: app, Request: httptest.NewRequest("POST", "/api/settings/test/email", nil)}
	ctx := application.WithAuditRequestMeta(context.Background(), application.AuditRequestMeta{IP: "203.0.113.8", UserAgent: "settings-audit-test"})
	for _, outcome := range []string{"success", "failure"} {
		if err := recordPocketBaseTestEmailAudit(ctx, event, "test-root-key", outcome); err != nil {
			t.Fatal(err)
		}
	}
	stores, err := adapters.New("pocketbase", app)
	if err != nil {
		t.Fatal(err)
	}
	logs, err := stores.Audits.List(context.Background(), "", ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 10}})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, entry := range logs.Items {
		if entry.Action != "pocketbase.settings.test_email" {
			continue
		}
		seen[entry.Outcome] = true
		if strings.Contains(entry.Summary, "@") || strings.Contains(entry.Summary, "test@example") {
			t.Fatalf("test email audit contains a recipient: %s", entry.Summary)
		}
	}
	if !seen["success"] || !seen["failure"] {
		t.Fatalf("test email audit outcomes = %v, want success and failure", seen)
	}
}
