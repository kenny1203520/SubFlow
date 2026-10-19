package application_test

import (
	"context"
	"subflow/internal/domain"
	"subflow/internal/ports"
	"testing"
	"time"
)

func TestRecurringIncomeAutomaticIsIdempotentAndManualConfirmationPostsOnce(t *testing.T) {
	for _, mode := range []string{"automatic", "manual"} {
		t.Run(mode, func(t *testing.T) {
			f := newMemberTransferFixture(t)
			ctx := context.Background()
			now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
			f.service.Now = func() time.Time { return now }
			starts := time.Date(2026, 8, 19, 9, 0, 0, 0, time.UTC)
			template := domain.Income{Title: "Salary", AmountMinor: 10000, Currency: domain.CurrencyTWD, EarnedBy: f.ownerID, SplitMode: domain.SplitEqual, Splits: []*domain.IncomeSplit{{BaseSplit: domain.BaseSplit{UserID: f.ownerID}}, {BaseSplit: domain.BaseSplit{UserID: f.fromID}}}}
			series, err := f.service.CreateRecurringIncome(ctx, f.ownerID, domain.RecurringIncome{OwnerID: f.ownerID, Template: template, Timezone: "UTC", Cycle: domain.BillingMonthly, Interval: 1, StartsOn: starts, NextOn: starts, PostingMode: mode})
			if err != nil {
				t.Fatal(err)
			}
			if err = f.service.PostDueRecurringIncomes(ctx); err != nil {
				t.Fatal(err)
			}
			page, err := f.stores.RecurringIncomes.ListOccurrences(ctx, series.ID, ports.PageRequest{Page: 1, PerPage: 50})
			if err != nil || len(page.Items) != 3 {
				t.Fatalf("occurrences: %d %v", len(page.Items), err)
			}
			if mode == "automatic" {
				incomes, err := f.service.ListPersonalIncomes(ctx, f.ownerID, ports.PageRequest{Page: 1, PerPage: 50})
				if err != nil || incomes.TotalItems != 3 {
					t.Fatalf("posted incomes: %d %v", incomes.TotalItems, err)
				}
				if err = f.service.PostDueRecurringIncomes(ctx); err != nil {
					t.Fatal(err)
				}
				incomes, err = f.service.ListPersonalIncomes(ctx, f.ownerID, ports.PageRequest{Page: 1, PerPage: 50})
				if err != nil || incomes.TotalItems != 3 {
					t.Fatalf("duplicate: %d %v", incomes.TotalItems, err)
				}
			} else {
				incomes, err := f.service.ListPersonalIncomes(ctx, f.ownerID, ports.PageRequest{Page: 1, PerPage: 50})
				if err != nil || incomes.TotalItems != 0 {
					t.Fatalf("pending counted as income: %d %v", incomes.TotalItems, err)
				}
				actual := domain.Income{Title: "Salary received", AmountMinor: 9000, Currency: domain.CurrencyTWD, EarnedBy: f.ownerID, ReceivedOn: page.Items[0].ScheduledOn, SplitMode: domain.SplitEqual, Splits: []*domain.IncomeSplit{{BaseSplit: domain.BaseSplit{UserID: f.ownerID}}, {BaseSplit: domain.BaseSplit{UserID: f.fromID}}}}
				if _, err = f.service.ConfirmIncomeOccurrence(ctx, f.ownerID, series.ID, page.Items[0].ScheduledOn, actual); err != nil {
					t.Fatal(err)
				}
				if _, err = f.service.ConfirmIncomeOccurrence(ctx, f.ownerID, series.ID, page.Items[0].ScheduledOn, actual); err == nil {
					t.Fatal("period confirmed twice")
				}
				incomes, err = f.service.ListPersonalIncomes(ctx, f.ownerID, ports.PageRequest{Page: 1, PerPage: 50})
				if err != nil || incomes.TotalItems != 1 || incomes.Items[0].AmountMinor != 9000 {
					t.Fatalf("confirmed income: %#v %v", incomes, err)
				}
			}
		})
	}
}

func TestRecurringIncomePauseResumeSkipAndPermission(t *testing.T) {
	f := newMemberTransferFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	f.service.Now = func() time.Time { return now }
	start := now.AddDate(0, -2, 0)
	v, err := f.service.CreateRecurringIncome(ctx, f.ownerID, domain.RecurringIncome{Template: domain.Income{Title: "Grant", AmountMinor: 300, Currency: domain.CurrencyTWD, EarnedBy: f.ownerID}, Timezone: "UTC", Cycle: domain.BillingMonthly, Interval: 1, StartsOn: start, NextOn: start, PostingMode: "manual"})
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.SetRecurringIncomeStatus(ctx, f.ownerID, v.ID, "paused", nil); err != nil {
		t.Fatal(err)
	}
	if err = f.service.PostDueRecurringIncomes(ctx); err != nil {
		t.Fatal(err)
	}
	page, err := f.stores.RecurringIncomes.ListOccurrences(ctx, v.ID, ports.PageRequest{Page: 1, PerPage: 10})
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("paused series caught up: %#v %v", page, err)
	}
	if err = f.service.SetRecurringIncomeStatus(ctx, f.ownerID, v.ID, "active", nil); err != nil {
		t.Fatal(err)
	}
	if err = f.service.PostDueRecurringIncomes(ctx); err != nil {
		t.Fatal(err)
	}
	page, err = f.stores.RecurringIncomes.ListOccurrences(ctx, v.ID, ports.PageRequest{Page: 1, PerPage: 10})
	if err != nil || len(page.Items) > 1 {
		t.Fatalf("resume catchup: %#v %v", page, err)
	}
	if _, err = f.service.UpdateRecurringIncome(ctx, f.outsiderID, v.ID, *v); err == nil {
		t.Fatal("outsider changed recurring income")
	}
}

func TestRecurringIncomeScheduleUsesSavedTimezoneAndEndDate(t *testing.T) {
	f := newMemberTransferFixture(t)
	ctx := context.Background()
	now := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	f.service.Now = func() time.Time { return now }
	loc, _ := time.LoadLocation("America/New_York")
	start := time.Date(2024, 1, 31, 9, 0, 0, 0, loc)
	end := time.Date(2024, 2, 29, 9, 0, 0, 0, loc)
	v, err := f.service.CreateRecurringIncome(ctx, f.ownerID, domain.RecurringIncome{Template: domain.Income{Title: "Rent", AmountMinor: 100, Currency: domain.CurrencyTWD, EarnedBy: f.ownerID}, Timezone: loc.String(), Cycle: domain.BillingMonthly, Interval: 1, StartsOn: start, NextOn: start, EndsOn: &end, PostingMode: "automatic"})
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.PostDueRecurringIncomes(ctx); err != nil {
		t.Fatal(err)
	}
	page, err := f.stores.RecurringIncomes.ListOccurrences(ctx, v.ID, ports.PageRequest{Page: 1, PerPage: 20})
	if err != nil || len(page.Items) != 2 {
		t.Fatalf("end-of-month dates: %#v %v", page, err)
	}
	got := map[string]bool{}
	for _, occurrence := range page.Items {
		got[occurrence.ScheduledOn.In(loc).Format("2006-01-02 15:04")] = true
	}
	if !got["2024-01-31 09:00"] || !got["2024-02-29 09:00"] {
		t.Fatalf("local billing time drifted: %#v", got)
	}
}
