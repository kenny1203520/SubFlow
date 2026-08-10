package application

import (
	"testing"
	"time"

	"subflow/internal/domain"
)

func TestSubscriptionExpensesBetweenUsesRevisionAndBillingDate(t *testing.T) {
	location := time.UTC
	billingAt := time.Date(2026, time.August, 10, 0, 0, 0, 0, location)
	subscription := domain.Subscription{
		ID:               "sub-1",
		Name:             "Streaming",
		Category:         "entertainment",
		AmountMinor:      1000,
		Currency:         domain.CurrencyTWD,
		BaseCurrency:     domain.CurrencyTWD,
		BaseAmountMinor:  1000,
		BillingCycle:     domain.BillingMonthly,
		BillingInterval:  1,
		StartsOn:         time.Date(2025, time.August, 10, 0, 0, 0, 0, location),
		PaidBy:           "alice",
		SplitMode:        domain.SplitAmount,
		Splits:           []domain.ExpenseSplit{{UserID: "alice", AmountMinor: 1000, BaseAmountMinor: 1000}},
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
		Occurrences: []domain.SubscriptionOccurrence{{BillingAt: time.Date(2026, time.August, 10, 0, 0, 0, 0, location), ExpenseID: "expense-1"}},
	}

	values, err := subscriptionExpensesBetween(subscription, time.Date(2026, time.August, 1, 0, 0, 0, 0, location), time.Date(2026, time.September, 1, 0, 0, 0, 0, location), location)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 0 {
		t.Fatalf("expected posted occurrence to be skipped, got %#v", values)
	}
}