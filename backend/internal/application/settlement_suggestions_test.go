package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

func TestSettlementSuggestionsAreStableAndReservePendingSettlementAmounts(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	now := time.Date(2026, time.October, 25, 12, 0, 0, 0, time.UTC)
	f.service.Now = func() time.Time { return now }
	if _, err := f.service.CreateExpense(ctx, f.ownerID, domain.Expense{
		GroupID: f.groupID, Title: "Shared expense", AmountMinor: 900, Currency: domain.CurrencyTWD,
		PaidBy: f.ownerID, IncurredOn: now.Add(-time.Hour), SplitMode: domain.SplitEqual,
		Splits: []*domain.ExpenseSplit{
			{BaseSplit: domain.BaseSplit{UserID: f.ownerID}},
			{BaseSplit: domain.BaseSplit{UserID: f.memberID}},
			{BaseSplit: domain.BaseSplit{UserID: f.otherID}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	initial, err := f.service.SettlementSuggestions(ctx, f.ownerID, f.groupID)
	if err != nil {
		t.Fatal(err)
	}
	if len(initial) != 2 {
		t.Fatalf("expected one suggestion per debtor, got %#v", initial)
	}
	if initial[0].FromUserID >= initial[1].FromUserID {
		t.Fatalf("equal balances must use stable member-ID ordering, got %#v", initial)
	}
	for _, item := range initial {
		if item.ToUserID != f.ownerID || item.AmountMinor != 300 || item.Currency != domain.CurrencyTWD || item.Label != "simplified" || item.Version == "" {
			t.Fatalf("unexpected simplified settlement suggestion: %#v", item)
		}
	}

	group, err := f.stores.Groups.Get(ctx, f.groupID)
	if err != nil {
		t.Fatal(err)
	}
	group.SettlementApprovalEnabled = true
	if err = f.stores.Groups.Update(ctx, group); err != nil {
		t.Fatal(err)
	}
	pending, err := f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{
		GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.ownerID, AmountMinor: 100, SettledOn: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if pending.Status != "pending" || domain.SettlementIsEffective(*pending) {
		t.Fatalf("a review-required settlement must reserve a suggestion without changing the effective balance: %#v", pending)
	}

	reserved, err := f.service.SettlementSuggestions(ctx, f.ownerID, f.groupID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reserved) != 2 || reserved[0].FromUserID != f.otherID || reserved[0].AmountMinor != 300 || reserved[1].FromUserID != f.memberID || reserved[1].AmountMinor != 200 {
		t.Fatalf("pending settlement should reduce duplicate suggestions while preserving the other debtor, got %#v", reserved)
	}
}

func TestSettlementSuggestionVersionRejectsStaleSubmission(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	now := time.Date(2026, time.October, 25, 12, 0, 0, 0, time.UTC)
	f.service.Now = func() time.Time { return now }
	createExpense := func(payer string, amount int64, participants ...string) {
		t.Helper()
		splits := make([]*domain.ExpenseSplit, 0, len(participants))
		for _, userID := range participants {
			splits = append(splits, &domain.ExpenseSplit{BaseSplit: domain.BaseSplit{UserID: userID}})
		}
		if _, err := f.service.CreateExpense(ctx, payer, domain.Expense{
			GroupID: f.groupID, Title: "Versioned expense", AmountMinor: amount, Currency: domain.CurrencyTWD,
			PaidBy: payer, IncurredOn: now.Add(-time.Hour), SplitMode: domain.SplitEqual, Splits: splits,
		}); err != nil {
			t.Fatal(err)
		}
	}
	createExpense(f.ownerID, 900, f.ownerID, f.memberID, f.otherID)
	initial, err := f.service.SettlementSuggestions(ctx, f.ownerID, f.groupID)
	if err != nil || len(initial) == 0 {
		t.Fatalf("initial suggestion: %#v %v", initial, err)
	}
	stale := initial[0]
	createExpense(f.ownerID, 90, f.ownerID, stale.FromUserID)
	_, err = f.service.CreateSettlement(ctx, stale.FromUserID, domain.Settlement{
		GroupID: f.groupID, FromUserID: stale.FromUserID, ToUserID: stale.ToUserID,
		AmountMinor: stale.AmountMinor, Currency: stale.Currency, SettledOn: now, SuggestionVersion: stale.Version,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("stale suggestion should require a fresh review, got %v", err)
	}
	settlements, err := f.service.ListSettlements(ctx, f.ownerID, f.groupID, ports.SettlementQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 20}})
	if err != nil {
		t.Fatal(err)
	}
	if settlements.TotalItems != 0 {
		t.Fatalf("stale suggestion must not create a settlement, got %#v", settlements.Items)
	}
}
