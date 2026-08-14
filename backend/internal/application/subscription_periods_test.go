package application_test

import (
	"context"
	"testing"

	"subflow/internal/domain"
)

// Nothing in the UI could previously answer "which period is this price for",
// because a subscription only ever reports the price at its next billing date.
// SubscriptionPeriods resolves each period's own governing revision, so a
// range-scoped price change is visible period by period.
func TestSubscriptionPeriodsReportPerPeriodPrices(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	rangeStart := f.pastBilling.AddDate(0, 1, 0) // 2024-10-11
	rangeEnd := f.pastBilling.AddDate(0, 2, 0)   // 2024-11-11

	edit := *f.subscription
	edit.RevisionScope = "future"
	edit.EffectiveBillingAt = rangeStart
	edit.EndBillingAt = &rangeEnd
	edit.AmountMinor = 60000
	edit.SplitMode = domain.SplitEqual
	edit.Splits = []domain.ExpenseSplit{{UserID: f.owner}, {UserID: f.member}}
	if _, err := f.service.UpdateSubscription(ctx, f.owner, edit); err != nil {
		t.Fatal(err)
	}

	page, err := f.service.SubscriptionPeriods(ctx, f.member, f.subscription.ID, "", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Periods) != 5 {
		t.Fatalf("expected 5 periods, got %d", len(page.Periods))
	}
	if !page.Periods[0].BillingAt.Equal(f.pastBilling) {
		t.Fatalf("expected the first period at StartsOn %s, got %s", f.pastBilling, page.Periods[0].BillingAt)
	}
	want := []int64{30000, 60000, 60000, 30000, 30000}
	for index, amount := range want {
		if page.Periods[index].AmountMinor != amount {
			t.Fatalf("period %d (%s): want %d, got %d", index, page.Periods[index].BillingAt.Format("2006-01-02"), amount, page.Periods[index].AmountMinor)
		}
		if page.Periods[index].Status != "pending" {
			t.Fatalf("period %d: nothing has posted, want pending, got %q", index, page.Periods[index].Status)
		}
	}
	if page.NextCursor == "" {
		t.Fatal("expected a cursor when a full page was returned")
	}

	next, err := f.service.SubscriptionPeriods(ctx, f.member, f.subscription.ID, page.NextCursor, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Periods) == 0 || !next.Periods[0].BillingAt.After(page.Periods[4].BillingAt) {
		t.Fatalf("expected the cursor to continue past %s, got %#v", page.Periods[4].BillingAt, next.Periods)
	}
}

// A failed occurrence never becomes an expense, so without this endpoint it is
// entirely invisible to the user.
func TestSubscriptionPeriodsSurfacePostedAndFailedOccurrences(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	postedAt := f.pastBilling
	failedAt := f.pastBilling.AddDate(0, 1, 0)

	revisions, err := f.stores.Subscriptions.ListRevisions(ctx, f.subscription.ID)
	if err != nil || len(revisions) == 0 {
		t.Fatalf("expected an initial revision, got %v (err=%v)", revisions, err)
	}

	postedExpense := domain.Expense{
		GroupID: f.group.ID, SubscriptionID: f.subscription.ID, Title: "YouTube", AmountMinor: 12345,
		Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD, PaidBy: f.owner, IncurredOn: postedAt,
		SplitMode: domain.SplitAmount, Splits: []domain.ExpenseSplit{{UserID: f.owner, AmountMinor: 12345}},
	}
	if err = f.stores.Expenses.Create(ctx, &postedExpense); err != nil {
		t.Fatal(err)
	}
	if err = f.stores.Expenses.ReplaceSplits(ctx, postedExpense.ID, postedExpense.Splits); err != nil {
		t.Fatal(err)
	}
	if err = f.stores.Subscriptions.CreateOccurrence(ctx, &domain.SubscriptionOccurrence{
		SubscriptionID: f.subscription.ID, RevisionID: revisions[0].ID, ExpenseID: postedExpense.ID, BillingAt: postedAt, Status: "posted",
	}); err != nil {
		t.Fatal(err)
	}
	if err = f.stores.Subscriptions.CreateOccurrence(ctx, &domain.SubscriptionOccurrence{
		SubscriptionID: f.subscription.ID, RevisionID: revisions[0].ID, BillingAt: failedAt, Status: "failed", Error: "subscription_split_invalid",
	}); err != nil {
		t.Fatal(err)
	}

	page, err := f.service.SubscriptionPeriods(ctx, f.member, f.subscription.ID, "", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Periods) < 2 {
		t.Fatalf("expected at least 2 periods, got %d", len(page.Periods))
	}
	posted := page.Periods[0]
	if posted.Status != "posted" || posted.ExpenseID != postedExpense.ID {
		t.Fatalf("expected the first period to report its posted expense, got %#v", posted)
	}
	// The posted period must report what was actually billed, not the
	// revision's price, so a directly edited charge isn't misreported.
	if posted.AmountMinor != 12345 {
		t.Fatalf("expected the posted period to report the real expense amount 12345, got %d", posted.AmountMinor)
	}
	if len(posted.Splits) != 1 || posted.Splits[0].AmountMinor != 12345 {
		t.Fatalf("expected the posted period to carry the expense's hydrated splits, got %#v", posted.Splits)
	}
	failed := page.Periods[1]
	if failed.Status != "failed" || failed.Error != "subscription_split_invalid" {
		t.Fatalf("expected the second period to surface the failure, got %#v", failed)
	}
}

func TestSubscriptionPeriodsRejectsNonMember(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	outsider, err := f.stores.Users.Create(ctx, domain.SetupInput{AdminName: "Outsider", Email: "outsider@example.com", Password: "correct-horse-battery-staple"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.SubscriptionPeriods(ctx, outsider.ID, f.subscription.ID, "", 5); err == nil {
		t.Fatal("expected a non-member to be refused")
	}
}
