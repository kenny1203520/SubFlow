package application_test

import (
	"context"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"subflow/internal/adapters"
	"subflow/internal/adapters/pocketbase"
	"subflow/internal/application"
	"subflow/internal/domain"
	"subflow/internal/ports"
)

// End-to-end: a context carrying AuditRequestMeta (as the HTTP middleware in
// backend/internal/transport/httpapi/api.go attaches on every request) must
// result in an audit_logs row with IP and UserAgent populated, not the
// perpetual N/A this was fixed from.
func TestMutationRecordsCallerIPAndUserAgent(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	if err = pocketbase.EnsureSchema(app); err != nil {
		t.Fatal(err)
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(users)
	record.Set("email", "owner@example.com")
	record.Set("name", "owner@example.com")
	record.Set("timezone", "UTC")
	record.SetPassword("correct-horse-battery-staple")
	if err = app.Save(record); err != nil {
		t.Fatal(err)
	}
	userID := record.Id

	stores, err := adapters.New("pocketbase", app)
	if err != nil {
		t.Fatal(err)
	}
	service := application.New(stores)

	ctx := application.WithAuditRequestMeta(context.Background(), application.AuditRequestMeta{IP: "203.0.113.7", UserAgent: "integration-test/1.0"})
	group, err := service.CreateGroup(ctx, userID, domain.Group{Name: "Test Group", Currency: domain.CurrencyTWD})
	if err != nil {
		t.Fatal(err)
	}

	logs, err := stores.Audits.List(context.Background(), group.ID, ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 10}})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range logs.Items {
		if entry.Action != "group.created" {
			continue
		}
		found = true
		if entry.IP != "203.0.113.7" {
			t.Fatalf("expected IP to be recorded, got %q", entry.IP)
		}
		if entry.UserAgent != "integration-test/1.0" {
			t.Fatalf("expected UserAgent to be recorded, got %q", entry.UserAgent)
		}
	}
	if !found {
		t.Fatalf("expected a group.created audit row, got %#v", logs.Items)
	}

	// A background job with no HTTP-attached meta must not leave the columns
	// blank and indistinguishable from a bug — it should say "system".
	bgCtx := context.Background()
	deleteErr := service.DeleteGroup(bgCtx, userID, group.ID)
	if deleteErr != nil {
		t.Fatal(deleteErr)
	}
	logs, err = stores.Audits.List(context.Background(), group.ID, ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 10}})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range logs.Items {
		if entry.Action != "group.deleted" {
			continue
		}
		if entry.IP != "system" || entry.UserAgent != "system" {
			t.Fatalf("expected system placeholders for a context with no HTTP-attached meta, got ip=%q ua=%q", entry.IP, entry.UserAgent)
		}
	}
}
