package application_test

import (
	"context"
	"errors"
	"subflow/internal/application"
	"subflow/internal/domain"
	"testing"
	"time"
)

func TestIncomeCurrencyRepairAndAuditRollback(t *testing.T) {
	f := newMemberTransferFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	f.service.Now = func() time.Time { return now }
	var first *domain.Income
	for i := 0; i < 101; i++ {
		income, err := f.service.CreateGroupIncome(ctx, f.ownerID, domain.Income{GroupID: f.groupID, Title: "Receipt", AmountMinor: 101, Currency: domain.CurrencyTWD, ReceivedOn: now, EarnedBy: f.ownerID, SplitMode: domain.SplitEqual, Splits: []*domain.IncomeSplit{{BaseSplit: domain.BaseSplit{UserID: f.ownerID}}, {BaseSplit: domain.BaseSplit{UserID: f.fromID}}}})
		if err != nil {
			t.Fatal(err)
		}
		if first == nil {
			first = income
		}
	}
	// Simulate a previous currency change that did not rebase income records.
	first.BaseCurrency = domain.CurrencyUSD
	if err := f.stores.Incomes.Update(ctx, first); err != nil {
		t.Fatal(err)
	}
	summary, err := f.service.WorkspaceDashboard(ctx, f.ownerID, application.DashboardQuery{Scope: "group", GroupID: f.groupID, Month: "2026-10"})
	if err != nil || len(summary.IncomeCurrencyRepairIDs) != 1 || len(summary.Balances) != 0 {
		t.Fatalf("unsafe balances: %#v %v", summary, err)
	}
	preview, err := f.service.PreviewGroupCurrency(ctx, f.ownerID, f.groupID, domain.CurrencyTWD)
	if err != nil || preview.Affected != 101 || len(preview.IncomeCurrencyRepairIDs) != 1 {
		t.Fatalf("preview: %#v %v", preview, err)
	}
	f.service.Stores.Audits = unavailableIncomeAudit{f.stores.Audits}
	if _, err = f.service.ChangeGroupCurrency(ctx, f.ownerID, f.groupID, domain.CurrencyTWD); !errors.Is(err, domain.ErrAuditUnavailable) {
		t.Fatalf("audit: %v", err)
	}
	stored, err := f.stores.Incomes.Get(ctx, first.ID)
	if err != nil || stored.BaseCurrency != domain.CurrencyUSD {
		t.Fatalf("rebase escaped rollback: %#v %v", stored, err)
	}
	f.service.Stores.Audits = f.stores.Audits
	if _, err = f.service.ChangeGroupCurrency(ctx, f.ownerID, f.groupID, domain.CurrencyTWD); err != nil {
		t.Fatal(err)
	}
	stored, err = f.stores.Incomes.Get(ctx, first.ID)
	if err != nil || stored.BaseCurrency != domain.CurrencyTWD || stored.Splits[0].BaseAmountMinor+stored.Splits[1].BaseAmountMinor != 101 {
		t.Fatalf("repair: %#v %v", stored, err)
	}
	preview, err = f.service.PreviewGroupCurrency(ctx, f.ownerID, f.groupID, domain.CurrencyUSD)
	if err != nil || len(preview.Missing) != 101 {
		t.Fatalf("missing rates: %#v %v", preview, err)
	}
	if _, err = f.service.ChangeGroupCurrency(ctx, f.ownerID, f.groupID, domain.CurrencyUSD); !errors.Is(err, domain.ErrRateUnavailable) {
		t.Fatalf("missing rates change: %v", err)
	}
}
