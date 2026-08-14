package application_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"subflow/internal/application"
	"subflow/internal/domain"
)

func groupBalances(t *testing.T, summary domain.DashboardSummary) map[string]int64 {
	t.Helper()
	balances := map[string]int64{}
	for _, balance := range summary.Balances {
		balances[balance.UserID] = balance.AmountMinor
	}
	return balances
}

// The dashboard used to read a single 100-row page for every input, so a group
// past 100 expenses silently reported understated totals and — because the
// same truncated slice feeds domain.MemberBalances — wrong amounts owed
// between members. Seeding well past the old page size and measuring the
// delta against a baseline isolates the truncation from the fixture's own
// records.
func TestDashboardAggregatesBeyondOnePageOfExpenses(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	const count = 130
	const amount = int64(1000)
	const month = "2026-08"
	incurred := time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC)
	query := application.DashboardQuery{Scope: "group", GroupID: f.group.ID, Month: month}

	before, err := f.service.WorkspaceDashboard(ctx, f.member, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Currencies) != 1 {
		t.Fatalf("expected one currency bucket, got %#v", before.Currencies)
	}
	baselineShare := before.Currencies[0].PersonalShareMinor
	baselineBalances := groupBalances(t, before)

	// Every expense is paid by the owner and owed entirely by the member, so
	// each one moves the member's share and the pair's balances by `amount`.
	for index := 0; index < count; index++ {
		if _, err = f.service.CreateExpense(ctx, f.owner, domain.Expense{
			GroupID: f.group.ID, Title: fmt.Sprintf("Snack %d", index), AmountMinor: amount,
			Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD, PaidBy: f.owner,
			IncurredOn: incurred, SplitMode: domain.SplitAmount,
			Splits: []domain.ExpenseSplit{{UserID: f.member, AmountMinor: amount}},
		}); err != nil {
			t.Fatalf("seeding expense %d: %v", index, err)
		}
	}

	after, err := f.service.WorkspaceDashboard(ctx, f.member, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Currencies) != 1 {
		t.Fatalf("expected one currency bucket, got %#v", after.Currencies)
	}

	wantDelta := amount * count
	if got := after.Currencies[0].PersonalShareMinor - baselineShare; got != wantDelta {
		t.Fatalf("personal share should rise by %d across %d expenses, rose by %d", wantDelta, count, got)
	}
	afterBalances := groupBalances(t, after)
	if got := afterBalances[f.owner] - baselineBalances[f.owner]; got != wantDelta {
		t.Fatalf("owner should be owed %d more, got %d", wantDelta, got)
	}
	if got := afterBalances[f.member] - baselineBalances[f.member]; got != -wantDelta {
		t.Fatalf("member should owe %d more, got %d", wantDelta, -got)
	}
}
