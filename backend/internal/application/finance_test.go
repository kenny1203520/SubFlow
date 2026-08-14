package application

import (
	"testing"
	"time"

	"subflow/internal/domain"
)

// Billing dates used to be walked from index 0 under a fixed iteration cap,
// so an hourly subscription older than roughly 500 days exhausted the cap
// before reaching the requested window and reported no billing dates at all —
// the subscription simply vanished from the dashboard. The window is now
// located by index first, making the cost independent of the subscription's
// age.
func TestBillingDatesBetweenReachesWindowForLongLivedHourlySubscription(t *testing.T) {
	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	from := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC) // 22632 hours in
	to := from.AddDate(0, 0, 1)

	dates, err := billingDatesBetween(start, domain.BillingEveryNHours, 1, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(dates) != 24 {
		t.Fatalf("expected 24 hourly billing dates across the day, got %d", len(dates))
	}
	if !dates[0].Equal(from) {
		t.Fatalf("expected the first date to be the window start %s, got %s", from, dates[0])
	}
	if !dates[23].Equal(to.Add(-time.Hour)) {
		t.Fatalf("expected the last date to be the final hour before %s, got %s", to, dates[23])
	}
}

func TestBillingDatesBetweenStartsAtStartsOnWhenWindowPrecedesIt(t *testing.T) {
	start := time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC)
	from := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)

	dates, err := billingDatesBetween(start, domain.BillingMonthly, 1, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(dates) != 1 || !dates[0].Equal(start) {
		t.Fatalf("expected the single billing date to be StartsOn, got %#v", dates)
	}
}

func TestSubscriptionExpensesBetweenUsesRevisionAndBillingDate(t *testing.T) {
	location := time.UTC
	billingAt := time.Date(2026, time.August, 10, 0, 0, 0, 0, location)
	subscription := domain.Subscription{
		ID:              "sub-1",
		Name:            "Streaming",
		Category:        "entertainment",
		AmountMinor:     1000,
		Currency:        domain.CurrencyTWD,
		BaseCurrency:    domain.CurrencyTWD,
		BaseAmountMinor: 1000,
		BillingCycle:    domain.BillingMonthly,
		BillingInterval: 1,
		StartsOn:        time.Date(2025, time.August, 10, 0, 0, 0, 0, location),
		PaidBy:          "alice",
		SplitMode:       domain.SplitAmount,
		Splits:          []domain.ExpenseSplit{{UserID: "alice", AmountMinor: 1000, BaseAmountMinor: 1000}},
		Revisions: []domain.SubscriptionRevision{{
			ID:                 "rev-1",
			SubscriptionID:     "sub-1",
			Scope:              "one_off",
			EffectiveBillingAt: billingAt,
			Name:               "Streaming",
			Category:           "entertainment",
			AmountMinor:        1500,
			Currency:           domain.CurrencyTWD,
			BaseCurrency:       domain.CurrencyTWD,
			BaseAmountMinor:    1500,
			PaidBy:             "alice",
			SplitMode:          domain.SplitAmount,
			Splits:             []domain.ExpenseSplit{{UserID: "alice", AmountMinor: 1500, BaseAmountMinor: 1500}},
		}},
	}

	values, err := subscriptionExpensesBetween(subscription, time.Date(2026, time.August, 1, 0, 0, 0, 0, location), time.Date(2026, time.September, 1, 0, 0, 0, 0, location), location)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 synthetic expense, got %d", len(values))
	}
	if values[0].AmountMinor != 1500 || !values[0].IncurredOn.Equal(billingAt) || values[0].SubscriptionID != "sub-1" {
		t.Fatalf("unexpected synthetic expense: %#v", values[0])
	}
}

func TestSubscriptionExpensesBetweenSkipsPostedOccurrences(t *testing.T) {
	location := time.UTC
	subscription := domain.Subscription{
		ID:              "sub-2",
		Name:            "Streaming",
		AmountMinor:     1000,
		Currency:        domain.CurrencyTWD,
		BaseCurrency:    domain.CurrencyTWD,
		BaseAmountMinor: 1000,
		BillingCycle:    domain.BillingMonthly,
		BillingInterval: 1,
		StartsOn:        time.Date(2025, time.August, 10, 0, 0, 0, 0, location),
		PaidBy:          "alice",
		Occurrences:     []domain.SubscriptionOccurrence{{BillingAt: time.Date(2026, time.August, 10, 0, 0, 0, 0, location), ExpenseID: "expense-1"}},
	}

	values, err := subscriptionExpensesBetween(subscription, time.Date(2026, time.August, 1, 0, 0, 0, 0, location), time.Date(2026, time.September, 1, 0, 0, 0, 0, location), location)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 0 {
		t.Fatalf("expected posted occurrence to be skipped, got %#v", values)
	}
}

func TestSubscriptionExpensesBetweenPreservesSplitsForBalances(t *testing.T) {
	location := time.UTC
	billingAt := time.Date(2026, time.August, 10, 0, 0, 0, 0, location)
	subscription := domain.Subscription{
		ID:              "sub-3",
		Name:            "Streaming",
		AmountMinor:     1000,
		Currency:        domain.CurrencyTWD,
		BaseCurrency:    domain.CurrencyTWD,
		BaseAmountMinor: 1000,
		BillingCycle:    domain.BillingMonthly,
		BillingInterval: 1,
		StartsOn:        time.Date(2025, time.August, 10, 0, 0, 0, 0, location),
		PaidBy:          "alice",
		SplitMode:       domain.SplitAmount,
		Splits: []domain.ExpenseSplit{
			{UserID: "alice", AmountMinor: 500, BaseAmountMinor: 500},
			{UserID: "bob", AmountMinor: 500, BaseAmountMinor: 500},
		},
	}

	values, err := subscriptionExpensesBetween(subscription, time.Date(2026, time.August, 1, 0, 0, 0, 0, location), time.Date(2026, time.September, 1, 0, 0, 0, 0, location), location)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 synthetic expense, got %d", len(values))
	}
	if !values[0].IncurredOn.Equal(billingAt) {
		t.Fatalf("unexpected billing date: %#v", values[0].IncurredOn)
	}
	balances := domain.MemberBalances(values, nil)
	got := map[string]int64{}
	for _, balance := range balances {
		got[balance.UserID] = balance.AmountMinor
	}
	if got["alice"] != 500 || got["bob"] != -500 {
		t.Fatalf("unexpected balances for split subscription: %#v", got)
	}
}

func TestSubscriptionUserShare(t *testing.T) {
	shared := domain.Subscription{
		ID:      "sub-shared",
		GroupID: "group-1",
		PaidBy:  "alice",
		Splits: []domain.ExpenseSplit{
			{UserID: "alice", AmountMinor: 300, BaseAmountMinor: 300},
			{UserID: "bob", AmountMinor: 300, BaseAmountMinor: 300},
			{UserID: "carol", AmountMinor: 300, BaseAmountMinor: 300},
		},
	}
	personalOnly := domain.Subscription{ID: "sub-personal", PaidBy: "alice"}

	cases := []struct {
		name     string
		sub      domain.Subscription
		userID   string
		scope    string
		amount   int64
		expected int64
	}{
		{"group scope reduces to viewer split", shared, "alice", "group", 900, 300},
		{"personal scope reduces to viewer split", shared, "alice", "personal", 900, 300},
		{"all scope reduces to viewer split", shared, "bob", "all", 900, 300},
		{"non-participant gets zero share in group scope", shared, "dave", "group", 900, 0},
		{"non-participant gets zero share", shared, "dave", "personal", 900, 0},
		{"unshared personal subscription keeps full amount", personalOnly, "alice", "personal", 500, 500},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := subscriptionUserShare(testCase.sub.Splits, testCase.sub.GroupID, testCase.userID, testCase.scope, testCase.amount)
			if got != testCase.expected {
				t.Fatalf("expected %d, got %d", testCase.expected, got)
			}
		})
	}
}
