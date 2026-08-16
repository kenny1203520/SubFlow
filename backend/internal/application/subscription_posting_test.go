package application_test

import (
	"context"
	"testing"
	"time"

	"subflow/internal/domain"
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

// A personal subscription's due date now posts a real Expense too, exactly
// like a group subscription's -- see Service.postSubscriptionOccurrence.
// Without this, a personal subscription could never accumulate more than a
// single synthetic placeholder in the export/expense list, no matter how
// long it had actually been running for.
func TestPostDueSubscriptionsPostsForPersonalSubscription(t *testing.T) {
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
	dueDate := personal.NextBilling // "now" (2026-08-11), same anchor day as f.subscription

	if err = f.service.PostDueSubscriptions(ctx); err != nil {
		t.Fatal(err)
	}

	occurrences, err := f.stores.Subscriptions.ListOccurrences(ctx, personal.ID)
	if err != nil {
		t.Fatal(err)
	}
	posted := 0
	for _, occurrence := range occurrences {
		if occurrence.Status == "posted" && occurrence.BillingAt.Equal(dueDate) && occurrence.ExpenseID != "" {
			posted++
		}
	}
	if posted != 1 {
		t.Fatalf("expected exactly 1 posted occurrence for the personal subscription, got %d (of %d total): %#v", posted, len(occurrences), occurrences)
	}

	expenses, err := f.service.ListPersonalExpenses(ctx, f.owner, ports.PageRequest{Page: 1, PerPage: 100})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, expense := range expenses.Items {
		if expense.SubscriptionID == personal.ID {
			found = true
			if expense.OwnerID != f.owner {
				t.Fatalf("expected the posted expense to be owned by %s, got %q", f.owner, expense.OwnerID)
			}
		}
	}
	if !found {
		t.Fatal("expected the personal subscription's due period to produce a real, listable personal expense")
	}
}

// A subscription created before every subscription got an initial revision
// (see CreateSubscription) has none at all, which used to make
// postOccurrenceAt hard-fail with "subscription_version_missing" on its
// very first cron tick. postSubscriptionOccurrence now self-heals this via
// ensureBaseRevisionCovers before attempting to post.
func TestPostDueSubscriptionsSelfHealsSubscriptionWithNoRevisions(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	now := time.Date(2026, time.August, 11, 0, 0, 0, 0, time.UTC)

	legacy := &domain.Subscription{
		OwnerID: f.owner, PaidBy: f.owner, Name: "Legacy Personal Sub",
		AmountMinor: 1000, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		BaseAmountMinor: 1000, RateScaled: domain.ExchangeRateScale, ExchangeRate: "1", ExchangeRateDate: now,
		RateMode: domain.RateAutomatic, BillingCycle: domain.BillingMonthly, BillingInterval: 1,
		StartsOn: time.Date(2024, time.March, 3, 0, 0, 0, 0, time.UTC), NextBilling: now,
		Status: domain.SubscriptionActive, SplitMode: domain.SplitAmount,
		Splits: []domain.ExpenseSplit{{UserID: f.owner, AmountMinor: 1000, BaseAmountMinor: 1000}},
	}
	// Bypasses Service.CreateSubscription deliberately, to simulate a
	// subscription that predates it always creating an initial revision.
	if err := f.stores.Subscriptions.Create(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	if revisions, err := f.stores.Subscriptions.ListRevisions(ctx, legacy.ID); err != nil || len(revisions) != 0 {
		t.Fatalf("expected the legacy subscription to start with zero revisions, got %d (err=%v)", len(revisions), err)
	}

	if err := f.service.PostDueSubscriptions(ctx); err != nil {
		t.Fatal(err)
	}

	occurrences, err := f.stores.Subscriptions.ListOccurrences(ctx, legacy.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(occurrences) != 1 || occurrences[0].Status != "posted" {
		t.Fatalf("expected the self-healed subscription to post successfully, got %#v", occurrences)
	}
}

// RefreshAutomaticSubscriptions now also touches personal subscriptions on
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
