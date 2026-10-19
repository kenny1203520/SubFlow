package application_test

import (
	"context"
	"subflow/internal/application"
	"subflow/internal/domain"
	"testing"
	"time"
)

func TestIncomeMetricsSeparateBenefitsFromReceipts(t *testing.T) {
	f := newMemberTransferFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	f.service.Now = func() time.Time { return now }
	// More than one API page; allocation rounds 101 into 51 + 50.
	for i := 0; i < 101; i++ {
		_, err := f.service.CreateGroupIncome(ctx, f.ownerID, domain.Income{GroupID: f.groupID, Title: "Shared receipt", AmountMinor: 101, Currency: domain.CurrencyTWD, EarnedBy: f.ownerID, ReceivedOn: now, SplitMode: domain.SplitAmount, Splits: []*domain.IncomeSplit{{BaseSplit: domain.BaseSplit{UserID: f.ownerID, AmountMinor: 51}}, {BaseSplit: domain.BaseSplit{UserID: f.fromID, AmountMinor: 50}}}})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		user        string
		share, cash int64
	}{{f.ownerID, 5151, 10201}, {f.fromID, 5050, 0}} {
		result, err := f.service.WorkspaceDashboard(ctx, tc.user, application.DashboardQuery{Scope: "all", Month: "2026-10"})
		if err != nil {
			t.Fatal(err)
		}
		if result.MonthIncomeMinor != tc.share || result.MonthReceivedIncomeMinor != tc.cash || len(result.Currencies) != 1 || result.Currencies[0].BeneficialIncomeMinor != tc.share || result.Currencies[0].CashInflowMinor != tc.cash {
			t.Fatalf("dashboard for %s: %#v", tc.user, result)
		}
		ledger, err := f.service.PersonalLedger(ctx, tc.user, "2026-10-19", true)
		if err != nil {
			t.Fatal(err)
		}
		if len(ledger.Items) != 101 || len(ledger.Summaries) != 1 || ledger.Summaries[0].IncomeMinor != tc.share || ledger.Summaries[0].ReceivedIncomeMinor != tc.cash || ledger.Items[0].AmountMinor != 101 {
			t.Fatalf("ledger: %#v", ledger)
		}
	}
}
