package application_test

import (
	"context"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"subflow/internal/adapters"
	"subflow/internal/adapters/pocketbase"
	"subflow/internal/application"
	"subflow/internal/domain"
)

func newExportFixture(t *testing.T) (*application.Service, string, string) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	if err = pocketbase.EnsureSchema(app); err != nil {
		t.Fatal(err)
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(users)
	record.Set("email", "export-owner@example.com")
	record.Set("name", "Export Owner")
	record.Set("timezone", "UTC")
	record.SetPassword("correct-horse-battery-staple")
	if err = app.Save(record); err != nil {
		t.Fatal(err)
	}
	userID := record.Id

	stores, err := adapters.New("pocketbase", app)
	if err != nil {
		t.Fatal(err)
	}
	service := application.New(stores)
	ctx := context.Background()

	group := &domain.Group{Name: "Export Group", Currency: domain.CurrencyTWD, Color: "#7057e8", OwnerID: userID, Timezone: "UTC"}
	if err = stores.Groups.Create(ctx, group); err != nil {
		t.Fatal(err)
	}
	if err = stores.Memberships.Create(ctx, &domain.Membership{GroupID: group.ID, UserID: userID, Role: domain.RoleOwner}); err != nil {
		t.Fatal(err)
	}

	if _, err = service.CreateExpense(ctx, userID, domain.Expense{
		GroupID: group.ID, Title: "Team Lunch", AmountMinor: 12000, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		PaidBy: userID, IncurredOn: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.CreateSubscription(ctx, userID, domain.Subscription{
		GroupID: group.ID, Name: "YouTube", AmountMinor: 30000, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		BillingCycle: domain.BillingMonthly, StartsOn: time.Date(2025, time.August, 10, 0, 0, 0, 0, time.UTC),
		Status: domain.SubscriptionActive, PaidBy: userID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.CreateExpense(ctx, userID, domain.Expense{
		Title: "Personal Coffee", AmountMinor: 5000, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		PaidBy: userID, IncurredOn: time.Date(2026, time.August, 2, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}

	return service, userID, group.ID
}

func TestExportLedgerGroupIncludesExpenseAndSubscription(t *testing.T) {
	service, userID, groupID := newExportFixture(t)
	data, filename, err := service.ExportLedger(context.Background(), userID, groupID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(filename, ".csv") {
		t.Fatalf("expected a .csv filename, got %q", filename)
	}
	text := string(data)
	if !strings.HasPrefix(text, "\xEF\xBB\xBF") {
		t.Fatalf("expected UTF-8 BOM prefix for Excel compatibility")
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(text, "\xEF\xBB\xBF")))
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 { // header + expense + subscription
		t.Fatalf("expected 3 rows (header + 2 records), got %d: %#v", len(rows), rows)
	}
	foundExpense, foundSubscription := false, false
	for _, row := range rows[1:] {
		switch row[0] {
		case "支出":
			foundExpense = true
			if row[2] != "Team Lunch" {
				t.Fatalf("expected expense title, got %#v", row)
			}
		case "訂閱":
			foundSubscription = true
			if row[2] != "YouTube" {
				t.Fatalf("expected subscription name, got %#v", row)
			}
		}
	}
	if !foundExpense || !foundSubscription {
		t.Fatalf("expected both an expense and a subscription row, got %#v", rows)
	}
}

// The personal-scope export reuses the same "owner=me || paid_by=me" query as
// ListPersonalExpenses/ListPersonalSubscriptions (the same records the app's
// personal ledger view already shows), so a group record the user paid for
// appears here too alongside their true personal (groupless) records.
func TestExportLedgerPersonalIncludesOwnedAndPaidRecords(t *testing.T) {
	service, userID, _ := newExportFixture(t)
	data, _, err := service.ExportLedger(context.Background(), userID, "")
	if err != nil {
		t.Fatal(err)
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\xEF\xBB\xBF")))
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 { // header + Personal Coffee + Team Lunch (paid by user) + YouTube (paid by user)
		t.Fatalf("expected 4 rows (header + 3 records), got %d: %#v", len(rows), rows)
	}
	titles := map[string]bool{}
	for _, row := range rows[1:] {
		titles[row[2]] = true
	}
	for _, want := range []string{"Personal Coffee", "Team Lunch", "YouTube"} {
		if !titles[want] {
			t.Fatalf("expected %q among personal-scope rows, got %#v", want, rows)
		}
	}
}
