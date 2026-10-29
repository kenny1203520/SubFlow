package application_test

import (
	"context"
	"testing"
	"time"

	"subflow/internal/domain"
)

func TestHomeDashboardUsesEachAccountingTimezoneAndCountsGroupCashflowOnce(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	group, err := f.stores.Groups.Get(ctx, f.groupID)
	if err != nil {
		t.Fatal(err)
	}
	group.Timezone = "America/Los_Angeles"
	if err = f.stores.Groups.Update(ctx, group); err != nil {
		t.Fatal(err)
	}
	f.service.Now = func() time.Time { return time.Date(2026, time.October, 1, 2, 0, 0, 0, time.UTC) }
	monthDate := time.Date(2026, time.September, 15, 12, 0, 0, 0, time.UTC)
	if _, err = f.service.CreateExpense(ctx, f.ownerID, domain.Expense{
		GroupID: f.groupID, Title: "Shared expense", AmountMinor: 10000, Currency: domain.CurrencyTWD,
		PaidBy: f.ownerID, IncurredOn: monthDate, SplitMode: domain.SplitEqual,
		Splits: []*domain.ExpenseSplit{{BaseSplit: domain.BaseSplit{UserID: f.ownerID}}, {BaseSplit: domain.BaseSplit{UserID: f.memberID}}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.CreateGroupIncome(ctx, f.ownerID, domain.Income{
		GroupID: f.groupID, Title: "Shared income", AmountMinor: 8000, Currency: domain.CurrencyTWD,
		EarnedBy: f.ownerID, ReceivedOn: monthDate.AddDate(0, 0, 1), SplitMode: domain.SplitEqual,
		Splits: []*domain.IncomeSplit{{BaseSplit: domain.BaseSplit{UserID: f.ownerID}}, {BaseSplit: domain.BaseSplit{UserID: f.memberID}}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{
		GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.ownerID, AmountMinor: 500,
		SettledOn: monthDate.AddDate(0, 0, 2),
	}); err != nil {
		t.Fatal(err)
	}

	got, err := f.service.HomeDashboard(ctx, f.ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Month != "2026-10" {
		t.Fatalf("personal month should use the account timezone, got %q", got.Month)
	}
	if len(got.Monthly) != 1 {
		t.Fatalf("expected one group/currency monthly row, got %#v", got.Monthly)
	}
	row := got.Monthly[0]
	if row.GroupID != f.groupID || row.Month != "2026-09" || row.Timezone != "America/Los_Angeles" {
		t.Fatalf("group monthly row should use group timezone and identify its source, got %#v", row)
	}
	if row.ExpensePaidMinor != 10000 || row.PersonalExpenseShareMinor != 5000 {
		t.Fatalf("group expense totals should be counted once in group currency: %#v", row)
	}
	if row.IncomeReceivedMinor != 8000 || row.BeneficialIncomeMinor != 4000 {
		t.Fatalf("group income totals should separate cash received from personal entitlement: %#v", row)
	}
	if row.SettlementReceivedMinor != 500 || row.SettlementPaidMinor != 0 {
		t.Fatalf("effective settlement cashflow should be attributed to the receiving member: %#v", row)
	}
}

func TestHomeDashboardShowsManualIncomeActionAndUpcomingRecurringIncome(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	now := time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	f.service.Now = func() time.Time { return now }
	template := domain.Income{Title: "Salary", AmountMinor: 120000, Currency: domain.CurrencyTWD, EarnedBy: f.ownerID}

	manual, err := f.service.CreateRecurringIncome(ctx, f.ownerID, domain.RecurringIncome{
		Template: template, Timezone: "UTC", Cycle: domain.BillingWeekly, Interval: 1,
		StartsOn: time.Date(2026, time.October, 19, 9, 0, 0, 0, time.UTC),
		NextOn:   time.Date(2026, time.October, 19, 9, 0, 0, 0, time.UTC), PostingMode: "manual",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.service.CreateRecurringIncome(ctx, f.ownerID, domain.RecurringIncome{
		Template: template, Timezone: "UTC", Cycle: domain.BillingWeekly, Interval: 1,
		StartsOn: time.Date(2026, time.October, 26, 9, 0, 0, 0, time.UTC),
		NextOn:   time.Date(2026, time.October, 26, 9, 0, 0, 0, time.UTC), PostingMode: "automatic",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.PostDueRecurringIncomes(ctx); err != nil {
		t.Fatal(err)
	}

	got, err := f.service.HomeDashboard(ctx, f.ownerID)
	if err != nil {
		t.Fatal(err)
	}
	var manualAction bool
	for _, action := range got.Actions {
		if action.Type == "recurring_income_manual_pending" && action.ResourceID == manual.ID {
			manualAction = action.ScheduledOn.Equal(manual.StartsOn)
		}
	}
	if !manualAction {
		t.Fatalf("expected the due manual income to appear as an action, got %#v", got.Actions)
	}
	var futureIncome bool
	for _, item := range got.Upcoming {
		if item.Kind == "income" && item.Name == "Salary" && item.NextBilling.Equal(time.Date(2026, time.October, 26, 9, 0, 0, 0, time.UTC)) {
			futureIncome = true
		}
	}
	if !futureIncome {
		t.Fatalf("expected the next recurring income forecast, got %#v", got.Upcoming)
	}
}
