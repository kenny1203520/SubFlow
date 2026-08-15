package application_test

import (
	"context"
	"testing"
	"time"

	"subflow/internal/ports"
)

// newHistoricalFixture's subscription starts 2024-09-11 monthly with
// service.Now() fixed at 2026-08-11 — its NextBilling lands exactly on
// "now", making it due for PostDueSubscriptions without any extra setup.
func TestPostDueSubscriptionsPostsAndAdvancesNextBilling(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	dueDate := f.subscription.NextBilling

	if err := f.service.PostDueSubscriptions(ctx); err != nil {
		t.Fatal(err)
	}

	occurrences, err := f.stores.Subscriptions.ListOccurrences(ctx, f.subscription.ID)
	if err != nil {
		t.Fatal(err)
	}
	posted := 0
	for _, occurrence := range occurrences {
		if occurrence.Status == "posted" && occurrence.BillingAt.Equal(dueDate) {
			posted++
		}
	}
	if posted != 1 {
		t.Fatalf("expected exactly 1 posted occurrence for %s, got %d (of %d total): %#v", dueDate.Format("2006-01-02"), posted, len(occurrences), occurrences)
	}

	expenses, err := f.stores.Expenses.List(ctx, f.group.ID, ports.PageRequest{Page: 1, PerPage: 100})
	if err != nil {
		t.Fatal(err)
	}
	if expenses.TotalItems != 1 {
		t.Fatalf("expected the due period to produce exactly 1 real expense, got %d", expenses.TotalItems)
	}

	updated, err := f.stores.Subscriptions.Get(ctx, f.subscription.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := dueDate.AddDate(0, 1, 0); !updated.NextBilling.Equal(want) {
		t.Fatalf("expected NextBilling to advance to %s, got %s", want.Format("2006-01-02"), updated.NextBilling.Format("2006-01-02"))
	}
}

// Re-running the scheduler against the same billing date (e.g. a duplicate
// cron tick before NextBilling has advanced in a caller's cached copy) must
// not create a second expense for it -- postOccurrenceAt's own
// (subscription, billing_at) existence check inside its transaction is what
// guarantees this.
func TestPostDueSubscriptionsIsIdempotentForTheSameBillingDate(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	dueDate := f.subscription.NextBilling

	if err := f.service.PostDueSubscriptions(ctx); err != nil {
		t.Fatal(err)
	}

	// Roll NextBilling back to the date that was just posted, simulating a
	// second scheduler tick that hasn't observed the advance yet.
	rewound, err := f.stores.Subscriptions.Get(ctx, f.subscription.ID)
	if err != nil {
		t.Fatal(err)
	}
	rewound.NextBilling = dueDate
	if err = f.stores.Subscriptions.Update(ctx, rewound); err != nil {
		t.Fatal(err)
	}

	if err = f.service.PostDueSubscriptions(ctx); err != nil {
		t.Fatal(err)
	}

	expenses, err := f.stores.Expenses.List(ctx, f.group.ID, ports.PageRequest{Page: 1, PerPage: 100})
	if err != nil {
		t.Fatal(err)
	}
	if expenses.TotalItems != 1 {
		t.Fatalf("expected the re-run to still leave exactly 1 expense for the same billing date, got %d", expenses.TotalItems)
	}
}

func TestPostDueSubscriptionsSkipsNotYetDue(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	future, err := f.stores.Subscriptions.Get(ctx, f.subscription.ID)
	if err != nil {
		t.Fatal(err)
	}
	future.NextBilling = f.subscription.NextBilling.AddDate(0, 1, 0)
	if err = f.stores.Subscriptions.Update(ctx, future); err != nil {
		t.Fatal(err)
	}

	if err = f.service.PostDueSubscriptions(ctx); err != nil {
		t.Fatal(err)
	}

	expenses, err := f.stores.Expenses.List(ctx, f.group.ID, ports.PageRequest{Page: 1, PerPage: 100})
	if err != nil {
		t.Fatal(err)
	}
	if expenses.TotalItems != 0 {
		t.Fatalf("expected nothing to post before the subscription is due, got %d expenses", expenses.TotalItems)
	}
}

// RefreshAutomaticSubscriptions only ever touches group subscriptions on
// automatic rate mode (see the rate_mode='automatic' filter in
// ListAutomaticSubscriptions) -- this checks it runs cleanly end to end and
// keeps the base-currency conversion fields populated for the identity
// (same-currency) case, without erroring.
func TestRefreshAutomaticSubscriptionsUpdatesConversionFields(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()

	if err := f.service.RefreshAutomaticSubscriptions(ctx); err != nil {
		t.Fatal(err)
	}

	updated, err := f.stores.Subscriptions.Get(ctx, f.subscription.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.BaseAmountMinor != updated.AmountMinor {
		t.Fatalf("expected the same-currency identity conversion to leave BaseAmountMinor equal to AmountMinor, got %d vs %d", updated.BaseAmountMinor, updated.AmountMinor)
	}
	if updated.ExchangeRateDate.IsZero() {
		t.Fatal("expected ExchangeRateDate to be populated after a refresh")
	}
}

func TestRefreshAutomaticSubscriptionsSkipsManualRateMode(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	manual, err := f.stores.Subscriptions.Get(ctx, f.subscription.ID)
	if err != nil {
		t.Fatal(err)
	}
	manual.RateMode = "manual"
	staleDate := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	manual.ExchangeRateDate = staleDate
	if err = f.stores.Subscriptions.Update(ctx, manual); err != nil {
		t.Fatal(err)
	}

	if err = f.service.RefreshAutomaticSubscriptions(ctx); err != nil {
		t.Fatal(err)
	}

	untouched, err := f.stores.Subscriptions.Get(ctx, f.subscription.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !untouched.ExchangeRateDate.Equal(staleDate) {
		t.Fatalf("expected a manual-rate subscription to be left untouched, but ExchangeRateDate changed to %s", untouched.ExchangeRateDate.Format("2006-01-02"))
	}
}
