package application_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

// decodedAuditSummary mirrors application.auditSummary's JSON shape without
// needing access to that unexported type from this external test package.
type decodedAuditSummary struct {
	Details map[string]any `json:"details"`
	Changes []struct {
		Field  string `json:"field"`
		Before any    `json:"before"`
		After  any    `json:"after"`
	} `json:"changes"`
}

func decodeSummary(t *testing.T, raw string) decodedAuditSummary {
	t.Helper()
	var summary decodedAuditSummary
	if err := json.Unmarshal([]byte(raw), &summary); err != nil {
		t.Fatalf("failed to decode audit summary %q: %v", raw, err)
	}
	return summary
}

func latestAuditLog(t *testing.T, f *historicalFixture, action string) domain.AuditLog {
	t.Helper()
	logs, err := f.stores.Audits.List(context.Background(), f.group.ID, ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 50}, Action: action})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs.Items) == 0 {
		t.Fatalf("expected at least one %q audit entry, got none", action)
	}
	return logs.Items[0] // Audits.List orders newest first (see other tests in this package).
}

// Every audit summary that carries amount_minor must also carry the
// currency it's denominated in -- a bare integer minor-unit amount is
// ambiguous without it (see auditsummary.go's design note on this).
func TestSettlementAuditIncludesCurrency(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()

	created, err := f.service.CreateSettlement(ctx, f.owner, domain.Settlement{GroupID: f.group.ID, FromUserID: f.owner, ToUserID: f.member, AmountMinor: 5500, SettledOn: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	createdLog := latestAuditLog(t, f, "settlement.created")
	createdSummary := decodeSummary(t, createdLog.Summary)
	if createdSummary.Details["currency"] != string(domain.CurrencyTWD) {
		t.Fatalf("expected settlement.created details to carry currency=TWD, got %#v", createdSummary.Details)
	}

	if err = f.service.DeleteSettlement(ctx, f.owner, created.ID); err != nil {
		t.Fatal(err)
	}
	deletedLog := latestAuditLog(t, f, "settlement.deleted")
	deletedSummary := decodeSummary(t, deletedLog.Summary)
	if deletedSummary.Details["currency"] != string(domain.CurrencyTWD) {
		t.Fatalf("expected settlement.deleted details to carry currency=TWD, got %#v", deletedSummary.Details)
	}
}

func TestSubscriptionCreatedAndDeletedAuditIncludeCurrency(t *testing.T) {
	f := newHistoricalFixture(t) // creation already happened during fixture setup
	ctx := context.Background()

	createdSummary := decodeSummary(t, latestAuditLog(t, f, "subscription.created").Summary)
	if createdSummary.Details["currency"] != string(domain.CurrencyTWD) {
		t.Fatalf("expected subscription.created details to carry currency=TWD, got %#v", createdSummary.Details)
	}

	if err := f.service.DeleteSubscription(ctx, f.owner, f.subscription.ID); err != nil {
		t.Fatal(err)
	}
	deletedSummary := decodeSummary(t, latestAuditLog(t, f, "subscription.deleted").Summary)
	if deletedSummary.Details["currency"] != string(domain.CurrencyTWD) {
		t.Fatalf("expected subscription.deleted details to carry currency=TWD, got %#v", deletedSummary.Details)
	}
}

func TestExpenseCreatedAuditIncludesCurrency(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()

	if _, err := f.service.CreateExpense(ctx, f.owner, domain.Expense{GroupID: f.group.ID, Title: "Snacks", AmountMinor: 200, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD, PaidBy: f.owner, IncurredOn: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatal(err)
	}
	summary := decodeSummary(t, latestAuditLog(t, f, "expense.created").Summary)
	if summary.Details["currency"] != string(domain.CurrencyTWD) {
		t.Fatalf("expected expense.created details to carry currency=TWD, got %#v", summary.Details)
	}
}

// The subscription.version_created entry used to only carry scope and
// effective_billing_at, so a viewer had to correlate it with a separate
// subscription.updated entry to know what actually changed for that date.
// It should be self-descriptive.
func TestSubscriptionVersionCreatedIncludesRevisionSnapshot(t *testing.T) {
	f := newHistoricalFixture(t) // fixture setup itself triggers CreateSubscription

	summary := decodeSummary(t, latestAuditLog(t, f, "subscription.version_created").Summary)
	if summary.Details["name"] != f.subscription.Name {
		t.Fatalf("expected version_created details to carry the subscription name, got %#v", summary.Details)
	}
	if summary.Details["currency"] != string(domain.CurrencyTWD) {
		t.Fatalf("expected version_created details to carry currency=TWD, got %#v", summary.Details)
	}
	if summary.Details["paid_by"] != f.owner {
		t.Fatalf("expected version_created details to carry paid_by, got %#v", summary.Details)
	}
	amount, ok := summary.Details["amount_minor"].(float64) // JSON numbers decode as float64
	if !ok || int64(amount) != f.subscription.AmountMinor {
		t.Fatalf("expected version_created details to carry amount_minor=%d, got %#v", f.subscription.AmountMinor, summary.Details["amount_minor"])
	}
}

func TestSubscriptionVersionCreatedIncludesEndBillingAtWhenBounded(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	start := f.pastBilling.AddDate(0, 1, 0)
	end := f.pastBilling.AddDate(0, 2, 0)

	edit := *f.subscription
	edit.RevisionScope = "future"
	edit.EffectiveBillingAt = start
	edit.EndBillingAt = &end
	if _, err := f.service.UpdateSubscription(ctx, f.owner, edit); err != nil {
		t.Fatal(err)
	}

	summary := decodeSummary(t, latestAuditLog(t, f, "subscription.version_created").Summary)
	want := end.Format("2006-01-02")
	if summary.Details["end_billing_at"] != want {
		t.Fatalf("expected version_created details to carry end_billing_at=%q, got %#v", want, summary.Details)
	}
}

// PostDueSubscriptions is the actual production posting cron; its audit
// entry must be self-descriptive about the amount it posted.
func TestOccurrencePostedAuditIncludesCurrency(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	if err := f.service.PostDueSubscriptions(ctx); err != nil {
		t.Fatal(err)
	}
	summary := decodeSummary(t, latestAuditLog(t, f, "subscription.occurrence_posted").Summary)
	if summary.Details["currency"] != string(domain.CurrencyTWD) {
		t.Fatalf("expected occurrence_posted details to carry currency=TWD, got %#v", summary.Details)
	}
}

// A retroactive edit that leaves currency unchanged used to leave the
// occurrence_regenerated entry's amount_minor without any currency context
// at all, since changeSet.addString only records a field when it differs.
// currency must now always be present in details regardless.
func TestOccurrenceRegeneratedDetailsIncludeCurrencyEvenWhenUnchanged(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	if err := f.service.PostDueSubscriptions(ctx); err != nil {
		t.Fatal(err)
	}

	edit := *f.subscription
	edit.RevisionScope = "one_off"
	edit.EffectiveBillingAt = f.subscription.NextBilling
	edit.AmountMinor = f.subscription.AmountMinor + 1000 // change amount, not currency
	edit.SplitMode = domain.SplitAmount
	// Both sides stay non-zero: PocketBase's NumberField treats a Required
	// field's zero value as blank (see historical_subscription_test.go's
	// note on the same quirk) -- unrelated to what this test is exercising.
	edit.Splits = []domain.ExpenseSplit{
		{UserID: f.owner, AmountMinor: edit.AmountMinor - 1000},
		{UserID: f.member, AmountMinor: 1000},
	}
	if _, err := f.service.UpdateSubscription(ctx, f.owner, edit); err != nil {
		t.Fatal(err)
	}

	summary := decodeSummary(t, latestAuditLog(t, f, "subscription.occurrence_regenerated").Summary)
	if summary.Details["currency"] != string(domain.CurrencyTWD) {
		t.Fatalf("expected occurrence_regenerated details to carry currency=TWD even though it didn't change, got %#v", summary.Details)
	}
}
