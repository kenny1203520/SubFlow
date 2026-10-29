package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	pbapis "ganigeorgiev.com/pocketbase/pocketbase/apis"
	"ganigeorgiev.com/pocketbase/pocketbase/core"
	"ganigeorgiev.com/pocketbase/pocketbase/tests"

	"subflow/internal/adapters"
	pbadapter "subflow/internal/adapters/pocketbase"
	"subflow/internal/domain"
	"subflow/internal/ports"
)

const settingsAuditTestKey = "test-root-key-for-settings-audit"

type settingsAuditHarness struct {
	app        *tests.TestApp
	handler    http.Handler
	adminID    string
	adminToken string
}

func newSettingsAuditHarness(t *testing.T) *settingsAuditHarness {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	if err := pbadapter.EnsureSchema(app); err != nil {
		t.Fatal(err)
	}
	admin := createSettingsAuditAuthRecord(t, app, core.CollectionNameSuperusers, "settings-admin@example.test")
	token, err := admin.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	registerPocketBaseSettingsAudit(app, settingsAuditTestKey)
	router, err := pbapis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	serve := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serve, func(*core.ServeEvent) error { return nil }); err != nil {
		t.Fatal(err)
	}
	handler, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return &settingsAuditHarness{app: app, handler: handler, adminID: admin.Id, adminToken: token}
}

func createSettingsAuditAuthRecord(t *testing.T, app core.App, collectionName, email string) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(collection)
	record.Set("email", email)
	record.Set("name", "Settings audit test")
	record.SetPassword("correct-horse-battery-staple")
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func (h *settingsAuditHarness) request(method, path, token string, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", token)
	}
	response := httptest.NewRecorder()
	h.handler.ServeHTTP(response, request)
	return response
}

func (h *settingsAuditHarness) settingsBody(t *testing.T, appURL, smtpPassword string) []byte {
	t.Helper()
	settings, err := h.app.Settings().Clone()
	if err != nil {
		t.Fatal(err)
	}
	settings.Meta.AppURL = appURL
	body, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	if smtpPassword != "" {
		var settingsData map[string]json.RawMessage
		if err := json.Unmarshal(body, &settingsData); err != nil {
			t.Fatal(err)
		}
		var smtpData map[string]any
		if err := json.Unmarshal(settingsData["smtp"], &smtpData); err != nil {
			t.Fatal(err)
		}
		smtpData["password"] = smtpPassword
		settingsData["smtp"], err = json.Marshal(smtpData)
		if err != nil {
			t.Fatal(err)
		}
		body, err = json.Marshal(settingsData)
		if err != nil {
			t.Fatal(err)
		}
	}
	return body
}

func (h *settingsAuditHarness) auditEntries(t *testing.T, action string) []domain.AuditLog {
	t.Helper()
	stores, err := adapters.New("pocketbase", h.app)
	if err != nil {
		t.Fatal(err)
	}
	logs, err := stores.Audits.List(context.Background(), "", ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 100}, Action: action})
	if err != nil {
		t.Fatal(err)
	}
	return logs.Items
}

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
	if strings.Join(fields, ",") != "meta,smtp" || !secretChanged {
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

func TestPocketBaseSettingsRoutesAuditSuccessAndRejectedApplicationURL(t *testing.T) {
	h := newSettingsAuditHarness(t)
	secret := "smtp-password-must-not-appear-in-audit"
	response := h.request(http.MethodPatch, "/api/settings", h.adminToken, h.settingsBody(t, "  https://subflow.example.test/  ", secret))
	if response.Code != http.StatusOK {
		t.Fatalf("valid settings update status = %d, body = %s", response.Code, response.Body.String())
	}
	if got := h.app.Settings().Meta.AppURL; got != "https://subflow.example.test" {
		t.Fatalf("Application URL = %q, want normalized root URL", got)
	}
	entries := h.auditEntries(t, "pocketbase.settings.updated")
	if len(entries) != 1 || entries[0].Outcome != "success" || entries[0].ActorID != "" {
		t.Fatalf("settings update audit entries = %#v, want one successful event by the admin", entries)
	}
	var summary pocketBaseSettingsAuditSummary
	if err := json.Unmarshal([]byte(entries[0].Summary), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.ActorID != h.adminID || !summary.SecretChanged || strings.Join(summary.ChangedFields, ",") != "meta,smtp" || strings.Contains(entries[0].Summary, secret) {
		t.Fatalf("settings audit summary metadata = actor:%q secretChanged:%v fields:%v secretLeaked:%v", summary.ActorID, summary.SecretChanged, summary.ChangedFields, strings.Contains(entries[0].Summary, secret))
	}

	response = h.request(http.MethodPatch, "/api/settings", h.adminToken, h.settingsBody(t, "https://subflow.example.test/app", ""))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"meta"`) || !strings.Contains(response.Body.String(), `"appURL"`) {
		t.Fatalf("subpath Application URL response = %d %s, want native meta.appURL validation error", response.Code, response.Body.String())
	}
	if got := h.app.Settings().Meta.AppURL; got != "https://subflow.example.test" {
		t.Fatalf("invalid URL changed saved Application URL to %q", got)
	}
	entries = h.auditEntries(t, "pocketbase.settings.updated")
	if len(entries) != 2 {
		t.Fatalf("settings URL rejection audit entries = %#v, want one failure without a duplicate event", entries)
	}
	seenOutcomes := map[string]bool{}
	for _, entry := range entries {
		seenOutcomes[entry.Outcome] = true
		if entry.Outcome == "failure" {
			if err := json.Unmarshal([]byte(entry.Summary), &summary); err != nil {
				t.Fatal(err)
			}
			if summary.FailureClass != "application_url_invalid" || !containsString(summary.ChangedFields, "meta") || strings.Contains(entry.Summary, "/app") {
				t.Fatalf("invalid URL audit summary = %s", entry.Summary)
			}
		}
	}
	if !seenOutcomes["success"] || !seenOutcomes["failure"] {
		t.Fatalf("settings URL outcomes = %v", seenOutcomes)
	}
}

func TestPocketBaseSettingsRouteAuditsAuthenticationAndRequestFailures(t *testing.T) {
	h := newSettingsAuditHarness(t)
	user := createSettingsAuditAuthRecord(t, h.app, "users", "ordinary-user@example.test")
	userToken, err := user.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		token      string
		body       []byte
		wantStatus int
		wantClass  string
	}{
		{name: "unauthenticated", wantStatus: http.StatusUnauthorized, wantClass: "unauthorized"},
		{name: "ordinary user", token: userToken, body: h.settingsBody(t, "https://subflow.example.test", ""), wantStatus: http.StatusForbidden, wantClass: "forbidden"},
		{name: "invalid JSON", token: h.adminToken, body: []byte("{"), wantStatus: http.StatusBadRequest, wantClass: "request_invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := h.request(http.MethodPatch, "/api/settings", test.token, test.body)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, body = %s, want %d", response.Code, response.Body.String(), test.wantStatus)
			}
		})
	}
	entries := h.auditEntries(t, "pocketbase.settings.updated")
	if len(entries) != 3 {
		t.Fatalf("settings rejection audit count = %d, want 3: %#v", len(entries), entries)
	}
	wantClasses := map[string]bool{"unauthorized": false, "forbidden": false, "request_invalid": false}
	for _, entry := range entries {
		if entry.Outcome != "failure" {
			t.Fatalf("rejected settings request was recorded as %q", entry.Outcome)
		}
		var summary pocketBaseSettingsAuditSummary
		if err := json.Unmarshal([]byte(entry.Summary), &summary); err != nil {
			t.Fatal(err)
		}
		if _, exists := wantClasses[summary.FailureClass]; !exists {
			t.Fatalf("unexpected failure class %q in %s", summary.FailureClass, entry.Summary)
		}
		wantClasses[summary.FailureClass] = true
	}
	for class, seen := range wantClasses {
		if !seen {
			t.Fatalf("missing settings failure event class %q", class)
		}
	}
}

func TestPocketBaseSettingsSuccessAuditFailureRollsBackSettings(t *testing.T) {
	h := newSettingsAuditHarness(t)
	originalURL := h.app.Settings().Meta.AppURL
	h.app.OnRecordCreateExecute(pbadapter.CollectionAuditLogs).BindFunc(func(event *core.RecordEvent) error {
		if event.Record.GetString("action") == "pocketbase.settings.updated" && event.Record.GetString("outcome") == "success" {
			return errors.New("forced success audit failure")
		}
		return event.Next()
	})

	response := h.request(http.MethodPatch, "/api/settings", h.adminToken, h.settingsBody(t, "https://new-subflow.example.test", ""))
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "audit record could not be written") {
		t.Fatalf("audit failure response = %d %s, want explicit 500", response.Code, response.Body.String())
	}
	if got := h.app.Settings().Meta.AppURL; got != originalURL {
		t.Fatalf("settings changed to %q after success audit failure; want rollback to %q", got, originalURL)
	}
	entries := h.auditEntries(t, "pocketbase.settings.updated")
	if len(entries) != 1 || entries[0].Outcome != "failure" {
		t.Fatalf("audit failure rollback events = %#v, want one separately committed failure event", entries)
	}
	var summary pocketBaseSettingsAuditSummary
	if err := json.Unmarshal([]byte(entries[0].Summary), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.FailureClass != "audit_write_failed" {
		t.Fatalf("failure summary = %s, want audit_write_failed", entries[0].Summary)
	}
}

func TestPocketBaseSettingsFailureAuditFailureReturnsExplicitError(t *testing.T) {
	h := newSettingsAuditHarness(t)
	originalURL := h.app.Settings().Meta.AppURL
	h.app.OnRecordCreateExecute(pbadapter.CollectionAuditLogs).BindFunc(func(event *core.RecordEvent) error {
		if event.Record.GetString("action") == "pocketbase.settings.updated" {
			return errors.New("forced audit failure")
		}
		return event.Next()
	})

	response := h.request(http.MethodPatch, "/api/settings", h.adminToken, h.settingsBody(t, "https://new-subflow.example.test", ""))
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "Settings update audit could not be written") {
		t.Fatalf("double audit failure response = %d %s, want explicit 500", response.Code, response.Body.String())
	}
	if got := h.app.Settings().Meta.AppURL; got != originalURL {
		t.Fatalf("settings changed to %q after both audit writes failed; want %q", got, originalURL)
	}
	if entries := h.auditEntries(t, "pocketbase.settings.updated"); len(entries) != 0 {
		t.Fatalf("failed audit writes unexpectedly persisted %#v", entries)
	}
}

func TestPocketBaseTestEmailAuditsSMTPAndBuffersResponseUntilResultAudit(t *testing.T) {
	t.Run("acceptance", func(t *testing.T) {
		h := newSettingsAuditHarness(t)
		h.app.OnMailerSend().UnbindAll()
		host, port, messages, commands := startSettingsAuditSMTP(t, false)
		configureSettingsAuditSMTP(t, h.app, host, port)
		response := h.request(http.MethodPost, "/api/settings/test/email", h.adminToken, []byte(`{"template":"verification","email":"recipient@example.test"}`))
		if response.Code != http.StatusNoContent {
			failureClass := "unknown"
			for _, entry := range h.auditEntries(t, "pocketbase.settings.test_email") {
				if entry.Outcome == "failure" {
					var summary pocketBaseSettingsAuditSummary
					_ = json.Unmarshal([]byte(entry.Summary), &summary)
					failureClass = summary.FailureClass
				}
			}
			seenCommands := make([]string, 0)
			for {
				select {
				case command := <-commands:
					seenCommands = append(seenCommands, command)
				default:
					has := func(want string) bool { return containsString(seenCommands, want) }
					t.Fatalf("SMTP send failed: status=%d class=%s ehlo=%v mail=%v rcpt=%v data=%v starttls=%v", response.Code, failureClass, has("EHLO"), has("MAIL"), has("RCPT"), has("DATA"), has("STARTTLS"))
				}
			}
		}
		select {
		case message := <-messages:
			if !bytes.Contains(message, []byte("recipient@example.test")) {
				t.Fatal("local SMTP server received an unexpected test message")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("local SMTP server received no test email")
		}
		entries := h.auditEntries(t, "pocketbase.settings.test_email")
		if len(entries) != 2 {
			t.Fatalf("test email audit entries = %#v, want attempt and success for the admin", entries)
		}
		outcomes := map[string]bool{}
		for _, entry := range entries {
			outcomes[entry.Outcome] = true
			var summary pocketBaseSettingsAuditSummary
			if err := json.Unmarshal([]byte(entry.Summary), &summary); err != nil {
				t.Fatal(err)
			}
			if entry.ActorID != "" || summary.ActorID != h.adminID {
				t.Fatalf("test email event actor = (%q, %q), want PocketBase superuser identity in summary", entry.ActorID, summary.ActorID)
			}
			if strings.Contains(entry.Summary, "recipient@example.test") {
				t.Fatalf("test email audit contains a recipient: %s", entry.Summary)
			}
		}
		if !outcomes["attempt"] || !outcomes["success"] {
			t.Fatalf("test email outcomes = %v", outcomes)
		}
	})

	t.Run("SMTP rejection is sanitized", func(t *testing.T) {
		h := newSettingsAuditHarness(t)
		h.app.OnMailerSend().UnbindAll()
		host, port, _, _ := startSettingsAuditSMTP(t, true)
		configureSettingsAuditSMTP(t, h.app, host, port)
		response := h.request(http.MethodPost, "/api/settings/test/email", h.adminToken, []byte(`{"template":"verification","email":"recipient@example.test"}`))
		if response.Code != http.StatusBadRequest || strings.Contains(response.Body.String(), "private-smtp-diagnostic") {
			t.Fatalf("SMTP rejection response leaked server diagnostics: %d %s", response.Code, response.Body.String())
		}
		entries := h.auditEntries(t, "pocketbase.settings.test_email")
		if len(entries) != 2 {
			t.Fatalf("SMTP rejection audits = %#v", entries)
		}
		var summary pocketBaseSettingsAuditSummary
		failureSeen, attemptSeen := false, false
		for _, entry := range entries {
			if entry.Outcome == "failure" {
				failureSeen = true
				if err := json.Unmarshal([]byte(entry.Summary), &summary); err != nil || summary.FailureClass != "smtp_rejected" {
					t.Fatalf("SMTP rejection summary = %s, error = %v", entry.Summary, err)
				}
			}
			attemptSeen = attemptSeen || entry.Outcome == "attempt"
		}
		if !failureSeen || !attemptSeen {
			t.Fatalf("SMTP rejection outcomes = %#v", entries)
		}
	})

	t.Run("authentication and validation failures keep PocketBase responses", func(t *testing.T) {
		h := newSettingsAuditHarness(t)
		body := []byte(`{"template":"verification","email":"recipient@example.test"}`)
		unauthorized := h.request(http.MethodPost, "/api/settings/test/email", "", body)
		if unauthorized.Code != http.StatusUnauthorized {
			t.Fatalf("unauthorized test email status = %d %s", unauthorized.Code, unauthorized.Body.String())
		}
		invalid := h.request(http.MethodPost, "/api/settings/test/email", h.adminToken, []byte(`{}`))
		if invalid.Code != http.StatusBadRequest || !strings.Contains(invalid.Body.String(), "validation_required") {
			t.Fatalf("invalid test email status = %d %s", invalid.Code, invalid.Body.String())
		}
		if h.app.TestMailer.TotalSend() != 0 {
			t.Fatalf("authentication/validation failure sent %d messages", h.app.TestMailer.TotalSend())
		}
		entries := h.auditEntries(t, "pocketbase.settings.test_email")
		if len(entries) != 4 {
			t.Fatalf("authentication/validation events = %#v, want attempt and result per request", entries)
		}
		outcomes := map[string]int{}
		for _, entry := range entries {
			outcomes[entry.Outcome]++
		}
		if outcomes["attempt"] != 2 || outcomes["failure"] != 2 {
			t.Fatalf("authentication/validation outcomes = %v", outcomes)
		}
	})

	t.Run("SMTP connection failure is audited without diagnostics", func(t *testing.T) {
		h := newSettingsAuditHarness(t)
		h.app.OnMailerSend().UnbindAll()
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		if err := listener.Close(); err != nil {
			t.Fatal(err)
		}
		configureSettingsAuditSMTP(t, h.app, "127.0.0.1", port)
		response := h.request(http.MethodPost, "/api/settings/test/email", h.adminToken, []byte(`{"template":"verification","email":"recipient@example.test"}`))
		if response.Code != http.StatusBadRequest || strings.Contains(response.Body.String(), "127.0.0.1") {
			t.Fatalf("SMTP connection failure response = %d %s, want sanitized 400", response.Code, response.Body.String())
		}
		entries := h.auditEntries(t, "pocketbase.settings.test_email")
		if len(entries) != 2 {
			t.Fatalf("connection failure audits = %#v", entries)
		}
		failureSeen, attemptSeen := false, false
		for _, entry := range entries {
			if entry.Outcome == "failure" {
				failureSeen = true
				var summary pocketBaseSettingsAuditSummary
				if err := json.Unmarshal([]byte(entry.Summary), &summary); err != nil || summary.FailureClass != "smtp_connection_failed" {
					t.Fatalf("connection failure summary = %s, error = %v", entry.Summary, err)
				}
			}
			attemptSeen = attemptSeen || entry.Outcome == "attempt"
		}
		if !failureSeen || !attemptSeen {
			t.Fatalf("connection failure outcomes = %#v", entries)
		}
	})

	t.Run("attempt audit failure prevents send", func(t *testing.T) {
		h := newSettingsAuditHarness(t)
		h.app.OnMailerSend().UnbindAll()
		host, port, messages, _ := startSettingsAuditSMTP(t, false)
		configureSettingsAuditSMTP(t, h.app, host, port)
		h.app.OnRecordCreateExecute(pbadapter.CollectionAuditLogs).BindFunc(func(event *core.RecordEvent) error {
			if event.Record.GetString("action") == "pocketbase.settings.test_email" {
				return errors.New("forced attempt audit failure")
			}
			return event.Next()
		})
		response := h.request(http.MethodPost, "/api/settings/test/email", h.adminToken, []byte(`{"template":"verification","email":"recipient@example.test"}`))
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("attempt audit failure status = %d %s", response.Code, response.Body.String())
		}
		select {
		case message := <-messages:
			t.Fatalf("SMTP received mail despite failed attempt audit: %q", message)
		case <-time.After(100 * time.Millisecond):
		}
	})

	t.Run("result audit failure returns 500 after one send", func(t *testing.T) {
		h := newSettingsAuditHarness(t)
		h.app.OnMailerSend().UnbindAll()
		host, port, messages, _ := startSettingsAuditSMTP(t, false)
		configureSettingsAuditSMTP(t, h.app, host, port)
		h.app.OnRecordCreateExecute(pbadapter.CollectionAuditLogs).BindFunc(func(event *core.RecordEvent) error {
			if event.Record.GetString("action") == "pocketbase.settings.test_email" && event.Record.GetString("outcome") == "success" {
				return errors.New("forced result audit failure")
			}
			return event.Next()
		})
		response := h.request(http.MethodPost, "/api/settings/test/email", h.adminToken, []byte(`{"template":"verification","email":"recipient@example.test"}`))
		if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "郵件可能已寄出，稽核記錄失敗") {
			t.Fatalf("result audit failure response = %d %s", response.Code, response.Body.String())
		}
		select {
		case <-messages:
		case <-time.After(2 * time.Second):
			t.Fatal("SMTP accepted the test email request but did not receive its message")
		}
		select {
		case message := <-messages:
			t.Fatalf("test email was retried after result audit failure: %q", message)
		case <-time.After(100 * time.Millisecond):
		}
		entries := h.auditEntries(t, "pocketbase.settings.test_email")
		if len(entries) != 1 || entries[0].Outcome != "attempt" {
			t.Fatalf("result-audit failure must preserve only the attempt event, got %#v", entries)
		}
	})
}

func configureSettingsAuditSMTP(t *testing.T, app core.App, host string, port int) {
	t.Helper()
	settings, err := app.Settings().Clone()
	if err != nil {
		t.Fatal(err)
	}
	settings.SMTP.Enabled = true
	settings.SMTP.Host = host
	settings.SMTP.Port = port
	settings.SMTP.TLS = false
	settings.SMTP.Username = ""
	settings.SMTP.Password = ""
	settings.Meta.SenderAddress = "noreply@example.test"
	if err := app.Save(settings); err != nil {
		t.Fatal(err)
	}
}

func startSettingsAuditSMTP(t *testing.T, reject bool) (string, int, <-chan []byte, <-chan string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	messages := make(chan []byte, 2)
	commands := make(chan string, 20)
	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			go func() {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
				reader := textproto.NewReader(bufio.NewReader(conn))
				_, _ = fmt.Fprint(conn, "220 localhost ESMTP\r\n")
				for {
					line, readErr := reader.ReadLine()
					if readErr != nil {
						return
					}
					if fields := strings.Fields(line); len(fields) > 0 {
						commands <- strings.ToUpper(fields[0])
					}
					switch {
					case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
						_, _ = fmt.Fprint(conn, "250-localhost\r\n250 8BITMIME\r\n")
					case strings.HasPrefix(line, "RCPT") && reject:
						_, _ = fmt.Fprint(conn, "550 private-smtp-diagnostic\r\n")
					case line == "DATA":
						_, _ = fmt.Fprint(conn, "354 send message\r\n")
						message, bodyErr := reader.ReadDotBytes()
						if bodyErr != nil {
							return
						}
						messages <- message
						_, _ = fmt.Fprint(conn, "250 queued\r\n")
					case line == "QUIT":
						_, _ = fmt.Fprint(conn, "221 bye\r\n")
						return
					default:
						_, _ = fmt.Fprint(conn, "250 ok\r\n")
					}
				}
			}()
		}
	}()
	address := listener.Addr().(*net.TCPAddr)
	return "127.0.0.1", address.Port, messages, commands
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestChangedSettingNamesIgnoreMissingAndSortFields(t *testing.T) {
	fields, secretChanged := changedSettingNames(map[string]any{"meta": true}, map[string]any{"meta": false, "new": true})
	if strings.Join(fields, ",") != "meta,new" || secretChanged {
		t.Fatalf("changedSettingNames = (%v, %v), want sorted non-secret fields", fields, secretChanged)
	}
}

func TestPocketBaseTestEmailAuditSummaryNeverStoresRecipient(t *testing.T) {
	h := newSettingsAuditHarness(t)
	event := &core.RequestEvent{App: h.app, Request: httptest.NewRequest("POST", "/api/settings/test/email", nil)}
	ctx := context.Background()
	if err := recordPocketBaseTestEmailAudit(ctx, event, settingsAuditTestKey, "success"); err != nil {
		t.Fatal(err)
	}
	entries := h.auditEntries(t, "pocketbase.settings.test_email")
	if len(entries) != 1 || strings.Contains(entries[0].Summary, "@") || strings.Contains(entries[0].Summary, "recipient") {
		t.Fatalf("test email audit summary is not minimal: %#v", entries)
	}
}

func TestSettingsAuditFailureClassDoesNotExposeError(t *testing.T) {
	var apiError error = (&core.RequestEvent{}).BadRequestError("smtp secret", nil)
	if got := pocketBaseFailureClass(apiError, 0); got != "smtp_rejected" {
		t.Fatalf("failure class = %q, want smtp_rejected", got)
	}
}
