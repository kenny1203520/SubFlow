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
	"subflow/internal/ports"
)

func newExportFixtureTZ(t *testing.T, timezone string) (*application.Service, adapters.Stores, string, string) {
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
	record.Set("timezone", timezone)
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

	group := &domain.Group{Name: "Export Group", Currency: domain.CurrencyTWD, Color: "#7057e8", OwnerID: userID, Timezone: timezone}
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

	return service, stores, userID, group.ID
}

func newExportFixture(t *testing.T) (*application.Service, string, string) {
	t.Helper()
	service, _, userID, groupID := newExportFixtureTZ(t, "UTC")
	return service, userID, groupID
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

// A posted occurrence already has a matching Expense row (created by
// postSubscriptionOccurrence); the export must not also emit a subscription
// row for it, or the same charge would be counted twice. A failed occurrence
// never gets an Expense, so it's the only period-level detail worth
// surfacing as its own row.
func TestExportLedgerSkipsPostedOccurrenceButIncludesFailed(t *testing.T) {
	service, stores, userID, groupID := newExportFixtureTZ(t, "UTC")
	ctx := context.Background()

	subs, err := service.ListSubscriptions(ctx, userID, groupID, ports.PageRequest{Page: 1, PerPage: 100})
	if err != nil {
		t.Fatal(err)
	}
	var subscriptionID string
	for _, v := range subs.Items {
		if v.Name == "YouTube" {
			subscriptionID = v.ID
		}
	}
	if subscriptionID == "" {
		t.Fatal("expected the YouTube subscription fixture to exist")
	}
	revisions, err := stores.Subscriptions.ListRevisions(ctx, subscriptionID)
	if err != nil || len(revisions) == 0 {
		t.Fatalf("expected an initial revision, got %v (err=%v)", revisions, err)
	}
	revisionID := revisions[0].ID

	postedExpense := domain.Expense{
		GroupID: groupID, SubscriptionID: subscriptionID, Title: "YouTube", AmountMinor: 30000,
		Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD, PaidBy: userID,
		IncurredOn: time.Date(2025, time.September, 10, 0, 0, 0, 0, time.UTC),
	}
	if err = stores.Expenses.Create(ctx, &postedExpense); err != nil {
		t.Fatal(err)
	}
	postedOccurrence := domain.SubscriptionOccurrence{SubscriptionID: subscriptionID, RevisionID: revisionID, ExpenseID: postedExpense.ID, BillingAt: postedExpense.IncurredOn, Status: "posted"}
	if err = stores.Subscriptions.CreateOccurrence(ctx, &postedOccurrence); err != nil {
		t.Fatal(err)
	}
	failedOccurrence := domain.SubscriptionOccurrence{SubscriptionID: subscriptionID, RevisionID: revisionID, BillingAt: time.Date(2025, time.October, 10, 0, 0, 0, 0, time.UTC), Status: "failed", Error: "subscription_split_invalid"}
	if err = stores.Subscriptions.CreateOccurrence(ctx, &failedOccurrence); err != nil {
		t.Fatal(err)
	}

	data, _, err := service.ExportLedger(ctx, userID, groupID)
	if err != nil {
		t.Fatal(err)
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\xEF\xBB\xBF")))
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	expenseRows, subscriptionRows := 0, 0
	sawFailedRow := false
	for _, row := range rows[1:] {
		switch row[0] {
		case "支出":
			expenseRows++
		case "訂閱":
			subscriptionRows++
			if row[7] == "failed" {
				sawFailedRow = true
			}
		}
	}
	if expenseRows != 2 { // Team Lunch + the posted YouTube occurrence's expense
		t.Fatalf("expected 2 expense rows, got %d: %#v", expenseRows, rows)
	}
	if subscriptionRows != 1 {
		t.Fatalf("expected exactly 1 subscription row (the failed occurrence, not a duplicate of the posted one), got %d: %#v", subscriptionRows, rows)
	}
	if !sawFailedRow {
		t.Fatalf("expected the failed occurrence to appear with status=failed, got %#v", rows)
	}
}

// The export must fully reflect how an expense was divided among members,
// not just its total, or a group can't reconcile who owes what from the CSV.
func TestExportLedgerIncludesExpenseSplitDetail(t *testing.T) {
	service, stores, ownerID, groupID := newExportFixtureTZ(t, "UTC")
	ctx := context.Background()

	created, err := stores.Users.Create(ctx, domain.SetupInput{AdminName: "Other Member", Email: "export-other@example.com", Password: "correct-horse-battery-staple"})
	if err != nil {
		t.Fatal(err)
	}
	if err = stores.Memberships.Create(ctx, &domain.Membership{GroupID: groupID, UserID: created.ID, Role: domain.RoleMember}); err != nil {
		t.Fatal(err)
	}

	if _, err = service.CreateExpense(ctx, ownerID, domain.Expense{
		GroupID: groupID, Title: "Split Dinner", AmountMinor: 10000, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		PaidBy: ownerID, IncurredOn: time.Date(2026, time.August, 3, 0, 0, 0, 0, time.UTC),
		SplitMode: domain.SplitEqual, Splits: []domain.ExpenseSplit{{UserID: ownerID}, {UserID: created.ID}},
	}); err != nil {
		t.Fatal(err)
	}

	data, _, err := service.ExportLedger(ctx, ownerID, groupID)
	if err != nil {
		t.Fatal(err)
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\xEF\xBB\xBF")))
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	header := rows[0]
	if header[8] != "分帳明細" {
		t.Fatalf("expected the split-detail header at index 8, got %#v", header)
	}
	found := false
	for _, row := range rows[1:] {
		if row[2] != "Split Dinner" {
			continue
		}
		found = true
		detail := row[8]
		if !strings.Contains(detail, "Export Owner:50.00") || !strings.Contains(detail, "Other Member:50.00") {
			t.Fatalf("expected both participants' 50/50 shares in split detail, got %q", detail)
		}
	}
	if !found {
		t.Fatalf("expected to find the Split Dinner row, got %#v", rows)
	}
}

// Reproduces the same UTC-vs-local-day bug class already fixed once in the
// date-input forms, this time in the export's date formatting: a record
// incurred late at night UTC is a different calendar day in Asia/Taipei, and
// the exported CSV must reflect the group's timezone, not UTC.
func TestExportLedgerUsesGroupTimezoneForDates(t *testing.T) {
	service, stores, userID, groupID := newExportFixtureTZ(t, "Asia/Taipei")
	ctx := context.Background()

	lateUTC := time.Date(2026, time.August, 9, 16, 0, 0, 0, time.UTC) // 2026-08-10 00:00 in Asia/Taipei
	expense := domain.Expense{
		GroupID: groupID, Title: "Late Night Snack", AmountMinor: 1000, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		PaidBy: userID, IncurredOn: lateUTC,
	}
	if err := stores.Expenses.Create(ctx, &expense); err != nil {
		t.Fatal(err)
	}

	data, _, err := service.ExportLedger(ctx, userID, groupID)
	if err != nil {
		t.Fatal(err)
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\xEF\xBB\xBF")))
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range rows[1:] {
		if row[2] == "Late Night Snack" {
			found = true
			if row[1] != "2026-08-10" {
				t.Fatalf("expected the Taipei-local date 2026-08-10 (not the UTC date 2026-08-09), got %q", row[1])
			}
		}
	}
	if !found {
		t.Fatalf("expected to find the late-night expense row, got %#v", rows)
	}
}
