package application_test

import (
	"context"
	"testing"
	"time"

	"subflow/internal/application"
	"subflow/internal/domain"
)

// The personal dashboard's subscription list intentionally also includes
// group subscriptions this user happens to be the designated payer for (the
// same "owner OR paid_by" query ExportLedger's personal ledger relies on --
// see TestExportLedgerPersonalIncludesOwnedAndPaidRecords). That's correct:
// MonthlySubscriptionMinor mirrors CashOutflowMinor's "full amount when I'm
// the one actually paying" semantics, while PersonalMonthlySubscriptionMinor
// mirrors PersonalShareMinor's "my fair share" semantics -- the same
// gross-vs-share split already established for expenses. This asserts that
// split stays intact for subscriptions: a personal subscription and a
// group subscription split with another member must combine correctly in
// both figures, not just get the group one's full amount double-applied to
// both.
func TestPersonalDashboardDistinguishesGrossFromPersonalShare(t *testing.T) {
	f := newHistoricalFixture(t) // seeds a 30000/mo group subscription, split evenly between f.owner and f.member, paid by f.owner
	ctx := context.Background()

	personal, err := f.service.CreateSubscription(ctx, f.owner, domain.Subscription{
		Name: "Truly Personal Sub", AmountMinor: 16900, Currency: domain.CurrencyTWD,
		BillingCycle: domain.BillingMonthly, Status: domain.SubscriptionActive,
		StartsOn: time.Date(2024, time.March, 3, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	// service.Now() is fixed to 2026-08-11 by the fixture; both schedules
	// (day 3 and day 11) have already billed once this month.
	dashboard, err := f.service.WorkspaceDashboard(ctx, f.owner, application.DashboardQuery{Scope: "personal", Month: "2026-08"})
	if err != nil {
		t.Fatal(err)
	}
	if len(dashboard.Currencies) != 1 {
		t.Fatalf("expected one currency bucket, got %#v", dashboard.Currencies)
	}
	bucket := dashboard.Currencies[0]

	wantGross := personal.AmountMinor + f.subscription.AmountMinor // 16900 + 30000: full YouTube amount, since f.owner is its payer
	if bucket.MonthlySubscriptionMinor != wantGross {
		t.Fatalf("expected the gross monthly total (personal + full group amount paid) to be %d, got %d", wantGross, bucket.MonthlySubscriptionMinor)
	}
	wantShare := personal.AmountMinor + f.subscription.AmountMinor/2 // 16900 + 15000: only f.owner's even split of YouTube
	if bucket.PersonalMonthlySubscriptionMinor != wantShare {
		t.Fatalf("expected the personal-share monthly total to be %d (personal amount + fair share of the group one), got %d", wantShare, bucket.PersonalMonthlySubscriptionMinor)
	}
	if bucket.PersonalMonthlySubscriptionMinor == bucket.MonthlySubscriptionMinor {
		t.Fatal("the gross and personal-share totals must diverge here -- a group subscription split with another member is not 100% this user's own cost")
	}
}
