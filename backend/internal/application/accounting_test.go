package application

import (
	"encoding/json"
	"testing"
	"time"

	"subflow/internal/domain"
)

func TestSubscriptionRevisionAtPrefersOneOffForMatchingBillingPeriod(t *testing.T) {
	billingAt := time.Date(2026, time.August, 10, 8, 0, 0, 0, time.UTC)
	futureAt := billingAt.AddDate(0, -1, 0)
	values := []domain.SubscriptionRevision{
		{ID: "future", Scope: "future", EffectiveBillingAt: futureAt, AmountMinor: 100},
		{ID: "one-off", Scope: "one_off", EffectiveBillingAt: billingAt, AmountMinor: 250},
	}

	actual, ok := subscriptionRevisionAt(values, billingAt)
	if !ok {
		t.Fatal("expected a revision")
	}
	if actual.ID != "one-off" || actual.AmountMinor != 250 {
		t.Fatalf("expected matching one-off revision, got %#v", actual)
	}
}

func TestSubscriptionRevisionAtUsesLatestFutureAndIgnoresExpiredOneOff(t *testing.T) {
	billingAt := time.Date(2026, time.August, 10, 8, 0, 0, 0, time.UTC)
	values := []domain.SubscriptionRevision{
		{ID: "old-future", Scope: "future", EffectiveBillingAt: billingAt.AddDate(0, -2, 0), AmountMinor: 100},
		{ID: "new-future", Scope: "future", EffectiveBillingAt: billingAt.AddDate(0, -1, 0), AmountMinor: 150},
		{ID: "past-one-off", Scope: "one_off", EffectiveBillingAt: billingAt.AddDate(0, 0, -1), AmountMinor: 99},
	}

	actual, ok := subscriptionRevisionAt(values, billingAt)
	if !ok {
		t.Fatal("expected a revision")
	}
	if actual.ID != "new-future" || actual.AmountMinor != 150 {
		t.Fatalf("expected latest future revision, got %#v", actual)
	}
}

func TestSubscriptionBillingDateAllowed(t *testing.T) {
	location := time.FixedZone("UTC+08", 8*60*60)
	startsOn := time.Date(2026, time.August, 3, 9, 0, 0, 0, location)
	subscription := domain.Subscription{
		StartsOn:        startsOn,
		NextBilling:     startsOn.AddDate(0, 0, 7),
		BillingCycle:    domain.BillingWeekly,
		BillingInterval: 1,
	}

	if !subscriptionBillingDateAllowed(subscription, startsOn.AddDate(0, 0, 14), location, false) {
		t.Fatal("expected a future weekly billing date to be accepted")
	}
	if subscriptionBillingDateAllowed(subscription, startsOn.AddDate(0, 0, 15), location, false) {
		t.Fatal("expected a non-billing date to be rejected")
	}
	if subscriptionBillingDateAllowed(subscription, startsOn, location, false) {
		t.Fatal("expected a billing date before next billing to be rejected")
	}

	// A caller holding ledger.records.historical_write may reach back to a past
	// billing date, but the date still has to land on the schedule.
	if !subscriptionBillingDateAllowed(subscription, startsOn, location, true) {
		t.Fatal("expected a past billing date to be accepted for a historical edit")
	}
	if subscriptionBillingDateAllowed(subscription, startsOn.AddDate(0, 0, 3), location, true) {
		t.Fatal("expected a past non-billing date to stay rejected for a historical edit")
	}
	if subscriptionBillingDateAllowed(subscription, startsOn.AddDate(0, 0, -7), location, true) {
		t.Fatal("expected a date before the subscription started to be rejected")
	}
}

func TestHistoricalExpenseChangeSummaryIncludesDetails(t *testing.T) {
	before := &domain.Expense{Title: "Lunch", IncurredOn: time.Date(2026, time.July, 10, 0, 0, 0, 0, time.UTC), AmountMinor: 1200, PaidBy: "alice", CategoryID: "food-old"}
	after := &domain.Expense{Title: "Team lunch", IncurredOn: time.Date(2026, time.July, 12, 0, 0, 0, 0, time.UTC), AmountMinor: 1500, PaidBy: "bob", CategoryID: "food-new"}
	summary := historicalExpenseChangeSummary(before, after)

	var decoded auditSummary
	if err := json.Unmarshal([]byte(summary), &decoded); err != nil {
		t.Fatalf("summary %q is not valid JSON: %v", summary, err)
	}
	if decoded.Details["title"] != "Team lunch" || decoded.Details["incurred_on"] != "2026-07-12" {
		t.Fatalf("unexpected details: %#v", decoded.Details)
	}
	got := map[string]auditChange{}
	for _, change := range decoded.Changes {
		got[change.Field] = change
	}
	for _, field := range []string{"title", "incurred_on", "amount_minor", "paid_by", "category_id"} {
		if _, ok := got[field]; !ok {
			t.Fatalf("summary %q missing change for %q", summary, field)
		}
	}
	if got["title"].Before != "Lunch" || got["title"].After != "Team lunch" {
		t.Fatalf("unexpected title change: %#v", got["title"])
	}
	if got["amount_minor"].Before != float64(1200) || got["amount_minor"].After != float64(1500) {
		t.Fatalf("unexpected amount_minor change: %#v", got["amount_minor"])
	}
}
