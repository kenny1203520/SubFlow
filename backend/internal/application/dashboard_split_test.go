package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"subflow/internal/adapters"
	"subflow/internal/adapters/pocketbase"
	"subflow/internal/application"
	"subflow/internal/domain"
)

// An equally split group subscription must be reported as the viewer's own
// share, not the full group amount. This exercises the whole stack (service ->
// PocketBase adapter -> dashboard aggregation) because the defect that made
// every participant look like they owed the full amount lived in the adapter's
// JSON read path, which the in-memory finance tests never reach.
func TestWorkspaceDashboardReportsSplitSubscriptionShare(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	if err = pocketbase.EnsureSchema(app); err != nil {
		t.Fatal(err)
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, 3)
	for _, email := range []string{"admin@example.com", "kenny@example.com", "it@example.com"} {
		record := core.NewRecord(users)
		record.Set("email", email)
		record.Set("name", email)
		record.Set("timezone", "UTC")
		record.SetPassword("correct-horse-battery-staple")
		if err = app.Save(record); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, record.Id)
	}
	payer := ids[0]

	stores, err := adapters.New("pocketbase", app)
	if err != nil {
		t.Fatal(err)
	}
	service := application.New(stores)
	now := time.Date(2026, time.August, 11, 0, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }

	ctx := context.Background()
	group := &domain.Group{Name: "Test Group", Currency: domain.CurrencyTWD, Color: "#7057e8", OwnerID: payer, Timezone: "UTC"}
	if err = stores.Groups.Create(ctx, group); err != nil {
		t.Fatal(err)
	}
	for i, id := range ids {
		role := domain.RoleMember
		if i == 0 {
			role = domain.RoleOwner
		}
		if err = stores.Memberships.Create(ctx, &domain.Membership{GroupID: group.ID, UserID: id, Role: role}); err != nil {
			t.Fatal(err)
		}
	}

	splits := make([]domain.ExpenseSplit, 0, len(ids))
	for _, id := range ids {
		splits = append(splits, domain.ExpenseSplit{UserID: id})
	}
	created, err := service.CreateSubscription(ctx, payer, domain.Subscription{
		GroupID:      group.ID,
		Name:         "YouTube",
		AmountMinor:  30000,
		Currency:     domain.CurrencyTWD,
		BaseCurrency: domain.CurrencyTWD,
		BillingCycle: domain.BillingMonthly,
		StartsOn:     time.Date(2025, time.August, 10, 0, 0, 0, 0, time.UTC),
		Status:       domain.SubscriptionActive,
		PaidBy:       payer,
		SplitMode:    domain.SplitEqual,
		Splits:       splits,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Splits) != 3 {
		t.Fatalf("expected 3 canonical splits, got %#v", created.Splits)
	}

	summary, err := service.WorkspaceDashboard(ctx, payer, application.DashboardQuery{Scope: "group", GroupID: group.ID, Month: "2026-08"})
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Currencies) != 1 {
		t.Fatalf("expected one currency bucket, got %#v", summary.Currencies)
	}
	item := summary.Currencies[0]
	// The group as a whole still spends the full 300; the viewer owes 100 and
	// is owed the other 200 by the two other participants.
	for _, check := range []struct {
		label string
		got   int64
		want  int64
	}{
		{"cashOutflowMinor", item.CashOutflowMinor, 30000},
		{"personalShareMinor", item.PersonalShareMinor, 10000},
		{"reimbursableMinor", item.ReimbursableMinor, 20000},
		{"monthlySubscriptionMinor", item.MonthlySubscriptionMinor, 30000},
		{"personalMonthlySubscriptionMinor", item.PersonalMonthlySubscriptionMinor, 10000},
	} {
		if check.got != check.want {
			t.Errorf("%s: want %d, got %d", check.label, check.want, check.got)
		}
	}
}
