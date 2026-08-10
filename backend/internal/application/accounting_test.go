package application

import (
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

	if !subscriptionBillingDateAllowed(subscription, startsOn.AddDate(0, 0, 14), location) {
		t.Fatal("expected a future weekly billing date to be accepted")
	}
	if subscriptionBillingDateAllowed(subscription, startsOn.AddDate(0, 0, 15), location) {
		t.Fatal("expected a non-billing date to be rejected")
	}
	if subscriptionBillingDateAllowed(subscription, startsOn, location) {
		t.Fatal("expected a billing date before next billing to be rejected")
	}
}
