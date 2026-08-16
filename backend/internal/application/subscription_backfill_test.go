package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

// A subscription backdated to migrate an existing ledger into SubFlow never
// gets real records for the periods between StartsOn and "now": StartsOn
// creation always sets NextBilling to the first date on/after now (see
// Service.CreateSubscription), and the due-date cron only ever posts at
// NextBilling. Those historical periods are otherwise only ever synthesized
// on the fly for display (dashboard, SubscriptionPeriods) — invisible to the
// real expense list and the ledger export. BackfillSubscriptionPeriods closes
// that gap.
func TestBackfillSubscriptionPeriodsCreatesHistoricalExpenses(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()

	created, err := f.service.BackfillSubscriptionPeriods(ctx, f.owner, f.subscription.ID)
	if err != nil {
		t.Fatal(err)
	}
	// StartsOn 2024-09-11 monthly through "now" 2026-08-11 (exclusive of
	// NextBilling, which lands exactly on "now") is 23 periods.
	if created != 23 {
		t.Fatalf("expected 23 backfilled periods, got %d", created)
	}

	occurrences, err := f.stores.Subscriptions.ListOccurrences(ctx, f.subscription.ID)
	if err != nil {
		t.Fatal(err)
	}
	posted := 0
	for _, occurrence := range occurrences {
		if occurrence.Status == "posted" && occurrence.ExpenseID != "" {
			posted++
		}
	}
	if posted != 23 {
		t.Fatalf("expected 23 posted occurrences, got %d (of %d total)", posted, len(occurrences))
	}

	// The backfilled periods must be real, listable expenses — not just
	// synthesized display data.
	expenses, err := f.stores.Expenses.List(ctx, f.group.ID, ports.PageRequest{Page: 1, PerPage: 100})
	if err != nil {
		t.Fatal(err)
	}
	if expenses.TotalItems != 23 {
		t.Fatalf("expected 23 real expenses from the backfill, got %d", expenses.TotalItems)
	}

	// Re-running must be a no-op: every period already has an occurrence.
	again, err := f.service.BackfillSubscriptionPeriods(ctx, f.owner, f.subscription.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again != 0 {
		t.Fatalf("expected re-running the backfill to create nothing, got %d", again)
	}
	expensesAfter, err := f.stores.Expenses.List(ctx, f.group.ID, ports.PageRequest{Page: 1, PerPage: 100})
	if err != nil {
		t.Fatal(err)
	}
	if expensesAfter.TotalItems != 23 {
		t.Fatalf("expected the re-run to leave the expense count at 23, got %d", expensesAfter.TotalItems)
	}
}

// A narrow one_off revision sitting exactly on StartsOn (e.g. a past
// correction to just that first period's split) used to fool
// ensureBaseRevisionCovers into thinking the whole backfill range was
// covered, since it only ever checked StartsOn itself: a one_off matches
// only its own exact date, so every later period was still uncovered, and
// postOccurrenceAt hard-failed on the second date with ErrInvalid instead
// of the backfill completing. Every date must be checked, not just the
// first.
func TestBackfillSubscriptionPeriodsCoversGapAfterLeadingOneOffRevision(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()

	oneOff := domain.SubscriptionRevision{
		SubscriptionID: f.subscription.ID, Scope: "one_off", EffectiveBillingAt: f.pastBilling,
		Name: f.subscription.Name, AmountMinor: f.subscription.AmountMinor, Currency: f.subscription.Currency,
		BaseCurrency: f.subscription.BaseCurrency, BaseAmountMinor: f.subscription.AmountMinor, RateMode: domain.RateAutomatic,
		PaidBy: f.owner, SplitMode: domain.SplitAmount,
		Splits: []domain.ExpenseSplit{{UserID: f.owner, AmountMinor: 30000}, {UserID: f.member, AmountMinor: 0}},
	}
	if err := f.stores.Subscriptions.CreateRevision(ctx, &oneOff); err != nil {
		t.Fatal(err)
	}

	created, err := f.service.BackfillSubscriptionPeriods(ctx, f.owner, f.subscription.ID)
	if err != nil {
		t.Fatalf("expected the backfill to succeed despite the leading one_off revision, got %v", err)
	}
	if created != 23 {
		t.Fatalf("expected all 23 periods to backfill, got %d", created)
	}

	occurrences, err := f.stores.Subscriptions.ListOccurrences(ctx, f.subscription.ID)
	if err != nil {
		t.Fatal(err)
	}
	failed := 0
	for _, occurrence := range occurrences {
		if occurrence.Status == "failed" {
			failed++
		}
	}
	if failed != 0 {
		t.Fatalf("expected no failed occurrences, got %d (of %d total)", failed, len(occurrences))
	}
}

func TestBackfillSubscriptionPeriodsRequiresHistoricalWritePermission(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	if _, err := f.service.BackfillSubscriptionPeriods(ctx, f.member, f.subscription.ID); err != domain.ErrForbidden {
		t.Fatalf("expected a member without ledger.records.historical_write to be forbidden, got %v", err)
	}
	occurrences, err := f.stores.Subscriptions.ListOccurrences(ctx, f.subscription.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(occurrences) != 0 {
		t.Fatalf("expected no occurrences to be created by a rejected backfill, got %d", len(occurrences))
	}
}

// Personal subscriptions post real occurrences the same way group ones do
// (see Service.postSubscriptionOccurrence), so a migrated personal
// subscription with a backdated StartsOn must be able to backfill its gap
// too -- otherwise it can never accumulate more than the single synthetic
// placeholder row the export/expense list would otherwise show for it
// forever, no matter how long it's actually been running.
func TestBackfillSubscriptionPeriodsWorksForPersonalSubscription(t *testing.T) {
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

	created, err := f.service.BackfillSubscriptionPeriods(ctx, f.owner, personal.ID)
	if err != nil {
		t.Fatalf("expected the personal subscription's backfill to succeed, got %v", err)
	}
	if created != 23 {
		t.Fatalf("expected 23 backfilled periods, got %d", created)
	}

	expenses, err := f.service.ListPersonalExpenses(ctx, f.owner, ports.PageRequest{Page: 1, PerPage: 100})
	if err != nil {
		t.Fatal(err)
	}
	posted := 0
	for _, expense := range expenses.Items {
		if expense.SubscriptionID == personal.ID {
			posted++
			if expense.OwnerID != f.owner {
				t.Fatalf("expected the posted expense to be owned by %s, got %q", f.owner, expense.OwnerID)
			}
		}
	}
	if posted != 23 {
		t.Fatalf("expected 23 real personal expenses from the backfill, got %d", posted)
	}
}

func TestBackfillSubscriptionPeriodsRejectsNonOwnerForPersonalSubscription(t *testing.T) {
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
	if _, err = f.service.BackfillSubscriptionPeriods(ctx, f.member, personal.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected a non-owner to be forbidden from backfilling someone else's personal subscription, got %v", err)
	}
}

// An hourly cadence backdated far enough would generate tens of thousands of
// real expense rows in one request; BackfillSubscriptionPeriods must refuse
// rather than silently doing that.
func TestBackfillSubscriptionPeriodsRejectsTooManyPeriods(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	hourly, err := f.service.CreateSubscription(ctx, f.owner, domain.Subscription{
		GroupID: f.group.ID, Name: "Hourly parking", AmountMinor: 100, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		BillingCycle: domain.BillingEveryNHours, BillingInterval: 1,
		StartsOn: time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC), // ~71 days before "now" = 1700+ hourly periods
		Status:   domain.SubscriptionActive, PaidBy: f.owner,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.BackfillSubscriptionPeriods(ctx, f.owner, hourly.ID); err != domain.ErrInvalid {
		t.Fatalf("expected too many periods to be rejected, got %v", err)
	}
	occurrences, err := f.stores.Subscriptions.ListOccurrences(ctx, hourly.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(occurrences) != 0 {
		t.Fatalf("expected no partial backfill when the limit is exceeded, got %d occurrences", len(occurrences))
	}
}
