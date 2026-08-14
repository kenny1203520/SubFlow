package application_test

import (
	"context"
	"testing"

	"subflow/internal/application"
	"subflow/internal/domain"
)

// The dashboard used to price its "monthly average" cards off the
// subscription's current (NextBilling) settings no matter which month was
// being viewed, while the cash-flow cards beside them were resolved per
// period. Looking at an earlier month therefore showed two contradictory sets
// of numbers in the same currency panel. For a monthly cadence the normalized
// average must simply equal that month's actual charge, which is the
// invariant asserted below.
func TestDashboardMonthlyFiguresUseTheViewedMonthsPrice(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	rangeStart := f.pastBilling.AddDate(0, 1, 0) // 2024-10-11
	rangeEnd := f.pastBilling.AddDate(0, 2, 0)   // 2024-11-11

	// Double the price for two past periods only.
	edit := *f.subscription
	edit.RevisionScope = "future"
	edit.EffectiveBillingAt = rangeStart
	edit.EndBillingAt = &rangeEnd
	edit.AmountMinor = 60000
	edit.SplitMode = domain.SplitEqual
	edit.Splits = []domain.ExpenseSplit{{UserID: f.owner}, {UserID: f.member}}
	if _, err := f.service.UpdateSubscription(ctx, f.owner, edit); err != nil {
		t.Fatalf("owner should be allowed to revise a bounded past range: %v", err)
	}

	cases := map[string]struct{ monthly, personal int64 }{
		"2024-09": {30000, 15000}, // before the range: original price
		"2024-10": {60000, 30000}, // in range
		"2024-11": {60000, 30000}, // in range (inclusive end)
		"2024-12": {30000, 15000}, // after the range: back to the original
	}
	for month, want := range cases {
		dashboard, err := f.service.WorkspaceDashboard(ctx, f.member, application.DashboardQuery{Scope: "group", GroupID: f.group.ID, Month: month})
		if err != nil {
			t.Fatal(err)
		}
		if len(dashboard.Currencies) != 1 {
			t.Fatalf("%s: expected one currency bucket, got %#v", month, dashboard.Currencies)
		}
		got := dashboard.Currencies[0]
		if got.MonthlySubscriptionMinor != want.monthly {
			t.Fatalf("%s: monthly average want %d, got %d", month, want.monthly, got.MonthlySubscriptionMinor)
		}
		if got.PersonalMonthlySubscriptionMinor != want.personal {
			t.Fatalf("%s: personal monthly average want %d, got %d", month, want.personal, got.PersonalMonthlySubscriptionMinor)
		}
		if got.MonthlySubscriptionMinor != got.CashOutflowMinor {
			t.Fatalf("%s: monthly average (%d) must agree with the same panel's cash outflow (%d)", month, got.MonthlySubscriptionMinor, got.CashOutflowMinor)
		}
		if got.PersonalMonthlySubscriptionMinor != got.PersonalShareMinor {
			t.Fatalf("%s: personal monthly average (%d) must agree with the same panel's personal share (%d)", month, got.PersonalMonthlySubscriptionMinor, got.PersonalShareMinor)
		}
	}
}

// A bounded range that lies entirely in the future is a temporary price for
// later periods, so it must not be written onto the subscription row, which
// represents what the subscription bills right now. Reads that don't hydrate
// revisions would otherwise show — and post — a price that isn't in effect.
func TestUpdateSubscriptionFutureOnlyRangeLeavesLiveDefaults(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	rangeStart := f.subscription.NextBilling.AddDate(0, 1, 0)
	rangeEnd := f.subscription.NextBilling.AddDate(0, 2, 0)

	edit := *f.subscription
	edit.RevisionScope = "future"
	edit.EffectiveBillingAt = rangeStart
	edit.EndBillingAt = &rangeEnd
	edit.AmountMinor = 99000
	if _, err := f.service.UpdateSubscription(ctx, f.owner, edit); err != nil {
		t.Fatalf("a future bounded range should be accepted: %v", err)
	}

	stored, err := f.stores.Subscriptions.Get(ctx, f.subscription.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.AmountMinor != 30000 {
		t.Fatalf("expected the live subscription row to keep its current price 30000, got %d", stored.AmountMinor)
	}
}

// The complement of the case above: an unbounded "this period onward" edit
// starting at NextBilling does govern what bills next, so it must update the
// live row.
func TestUpdateSubscriptionOnwardRangeUpdatesLiveDefaults(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()

	edit := *f.subscription
	edit.RevisionScope = "future"
	edit.EffectiveBillingAt = f.subscription.NextBilling
	edit.AmountMinor = 45000
	if _, err := f.service.UpdateSubscription(ctx, f.owner, edit); err != nil {
		t.Fatal(err)
	}

	stored, err := f.stores.Subscriptions.Get(ctx, f.subscription.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.AmountMinor != 45000 {
		t.Fatalf("expected the live subscription row to take the new price 45000, got %d", stored.AmountMinor)
	}
}
