package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"subflow/internal/domain"
)

// Stopping at or after NextBilling is the ordinary "schedule a future stop"
// case and must keep working unchanged.
func TestStopSubscriptionFutureDateStillWorks(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()

	updated, err := f.service.StopSubscription(ctx, f.owner, f.subscription.ID, f.subscription.NextBilling.Format("2006-01-02"))
	if err != nil {
		t.Fatalf("expected the future stop date to be accepted, got %v", err)
	}
	if updated.EndsOn == nil || !updated.EndsOn.Equal(f.subscription.NextBilling) {
		t.Fatalf("expected EndsOn to be set to NextBilling, got %v", updated.EndsOn)
	}
}

// A stop date before NextBilling is a historical correction (e.g. "this
// subscription actually already ended back in month X") and must be gated
// behind ledger.records.historical_write for a group subscription, the same
// permission historical edits require.
func TestStopSubscriptionHistoricalDateRequiresPermission(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()

	if _, err := f.service.StopSubscription(ctx, f.member, f.subscription.ID, f.pastBilling.Format("2006-01-02")); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for a member choosing a past stop date, got %v", err)
	}
}

func TestStopSubscriptionHistoricalDateAllowedWithPermission(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()

	updated, err := f.service.StopSubscription(ctx, f.owner, f.subscription.ID, f.pastBilling.Format("2006-01-02"))
	if err != nil {
		t.Fatalf("owner should be allowed to set a historical stop date: %v", err)
	}
	if updated.EndsOn == nil || !updated.EndsOn.Equal(f.pastBilling) {
		t.Fatalf("expected EndsOn to be set to the chosen past date %s, got %v", f.pastBilling.Format("2006-01-02"), updated.EndsOn)
	}
	if updated.LifecycleStatus != "ended" {
		t.Fatalf("expected a subscription with a past EndsOn to report lifecycle 'ended', got %q", updated.LifecycleStatus)
	}
}

// A personal (groupless) subscription has no group permission system to
// gate against; the owner check alone is the authorization boundary, so a
// historical stop date must work for the owner without any extra permission.
func TestStopSubscriptionHistoricalDateAllowedForPersonalOwner(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()

	personal, err := f.service.CreateSubscription(ctx, f.owner, domain.Subscription{
		Name: "Personal Netflix", AmountMinor: 1000, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		BillingCycle: domain.BillingMonthly, StartsOn: time.Date(2024, time.September, 11, 0, 0, 0, 0, time.UTC),
		Status: domain.SubscriptionActive, PaidBy: f.owner,
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := f.service.StopSubscription(ctx, f.owner, personal.ID, f.pastBilling.Format("2006-01-02"))
	if err != nil {
		t.Fatalf("owner should be allowed to set a historical stop date on their own personal subscription: %v", err)
	}
	if updated.EndsOn == nil || !updated.EndsOn.Equal(f.pastBilling) {
		t.Fatalf("expected EndsOn to be set to the chosen past date, got %v", updated.EndsOn)
	}
}
