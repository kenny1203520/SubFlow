package application_test

import (
	"context"
	"errors"
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

type historicalFixture struct {
	service      *application.Service
	stores       adapters.Stores
	group        *domain.Group
	subscription *domain.Subscription
	owner        string
	member       string
	ids          []string
	pastBilling  time.Time
}

// Builds a monthly group subscription that started well in the past, so its
// earlier billing dates are already behind the current NextBilling.
func newHistoricalFixture(t *testing.T) *historicalFixture {
	t.Helper()
	return newHistoricalFixtureTZ(t, "UTC")
}

// newHistoricalFixtureTZ is identical to newHistoricalFixture but seeds the
// group with the given IANA timezone, for cases where the group's non-UTC
// offset matters (e.g. reproducing a client that formats/parses dates in the
// group's local calendar day instead of preserving the exact instant).
func newHistoricalFixtureTZ(t *testing.T, timezone string) *historicalFixture {
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
	ids := make([]string, 0, 2)
	for _, email := range []string{"owner@example.com", "member@example.com"} {
		record := core.NewRecord(users)
		record.Set("email", email)
		record.Set("name", email)
		record.Set("timezone", timezone)
		record.SetPassword("correct-horse-battery-staple")
		if err = app.Save(record); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, record.Id)
	}
	stores, err := adapters.New("pocketbase", app)
	if err != nil {
		t.Fatal(err)
	}
	service := application.New(stores)
	service.Now = func() time.Time { return time.Date(2026, time.August, 11, 0, 0, 0, 0, time.UTC) }

	ctx := context.Background()
	group := &domain.Group{Name: "Test Group", Currency: domain.CurrencyTWD, Color: "#7057e8", OwnerID: ids[0], Timezone: timezone}
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
	created, err := service.CreateSubscription(ctx, ids[0], domain.Subscription{
		GroupID:      group.ID,
		Name:         "YouTube",
		AmountMinor:  30000,
		Currency:     domain.CurrencyTWD,
		BaseCurrency: domain.CurrencyTWD,
		BillingCycle: domain.BillingMonthly,
		StartsOn:     time.Date(2024, time.September, 11, 0, 0, 0, 0, time.UTC),
		Status:       domain.SubscriptionActive,
		PaidBy:       ids[0],
		SplitMode:    domain.SplitEqual,
		Splits:       splits,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &historicalFixture{
		service:      service,
		stores:       stores,
		group:        group,
		subscription: created,
		owner:        ids[0],
		member:       ids[1],
		ids:          ids,
		pastBilling:  time.Date(2024, time.September, 11, 0, 0, 0, 0, time.UTC),
	}
}

func TestUpdateSubscriptionRejectsHistoricalEditWithoutPermission(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()

	// The seeded member role does not carry ledger.records.historical_write.
	edit := *f.subscription
	edit.RevisionScope = "one_off"
	edit.EffectiveBillingAt = f.pastBilling
	edit.SplitMode = domain.SplitAmount
	edit.Splits = []domain.ExpenseSplit{
		{UserID: f.owner, AmountMinor: 30000},
		{UserID: f.member, AmountMinor: 0},
	}
	if _, err := f.service.UpdateSubscription(ctx, f.member, edit); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for a member editing a past period, got %v", err)
	}
}

func TestUpdateSubscriptionAllowsHistoricalEditWithPermission(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()

	edit := *f.subscription
	edit.RevisionScope = "one_off"
	edit.EffectiveBillingAt = f.pastBilling
	edit.SplitMode = domain.SplitAmount
	edit.Splits = []domain.ExpenseSplit{
		{UserID: f.owner, AmountMinor: 30000},
		{UserID: f.member, AmountMinor: 0},
	}
	// The owner seed role holds ledger.records.historical_write.
	if _, err := f.service.UpdateSubscription(ctx, f.owner, edit); err != nil {
		t.Fatalf("owner should be allowed to revise a past period: %v", err)
	}

	// The revised split must apply to that month and leave later ones alone.
	past, err := f.service.WorkspaceDashboard(ctx, f.member, application.DashboardQuery{Scope: "group", GroupID: f.group.ID, Month: "2024-09"})
	if err != nil {
		t.Fatal(err)
	}
	if len(past.Currencies) != 1 {
		t.Fatalf("expected one currency bucket for 2024-09, got %#v", past.Currencies)
	}
	if got := past.Currencies[0].PersonalShareMinor; got != 0 {
		t.Fatalf("2024-09 share for the member: want 0 after the revision, got %d", got)
	}

	later, err := f.service.WorkspaceDashboard(ctx, f.member, application.DashboardQuery{Scope: "group", GroupID: f.group.ID, Month: "2024-10"})
	if err != nil {
		t.Fatal(err)
	}
	if len(later.Currencies) != 1 {
		t.Fatalf("expected one currency bucket for 2024-10, got %#v", later.Currencies)
	}
	if got := later.Currencies[0].PersonalShareMinor; got != 15000 {
		t.Fatalf("2024-10 share for the member: want the untouched 15000, got %d", got)
	}
}

func TestUpdateSubscriptionRejectsPastDateOffSchedule(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()

	edit := *f.subscription
	edit.RevisionScope = "one_off"
	edit.EffectiveBillingAt = f.pastBilling.AddDate(0, 0, 3) // not a billing date
	if _, err := f.service.UpdateSubscription(ctx, f.owner, edit); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for a past date off the schedule, got %v", err)
	}
}

// Reproduces the frontend timezone bug directly against the backend: a date
// input that round-trips a UTC instant through a local calendar day (as the
// buggy client did) can land 24h off the stored StartsOn, which used to trip
// the one_off immutability guard and silently drop the whole edit. The fix
// is on the frontend (it now omits an untouched date field entirely), but
// this asserts the backend contract that makes that fix work: an omitted
// StartsOn always falls back to the stored value, in any group timezone.
func TestUpdateSubscriptionHistoricalEditSurvivesNonUTCTimezone(t *testing.T) {
	f := newHistoricalFixtureTZ(t, "Asia/Taipei")
	ctx := context.Background()

	// A client that reconstructs StartsOn from a same-day-shifted local date
	// (the old, buggy behaviour) sends an instant that no longer matches the
	// stored one, and must still be rejected by the immutability guard.
	shifted := *f.subscription
	shifted.RevisionScope = "one_off"
	shifted.EffectiveBillingAt = f.pastBilling
	shifted.StartsOn = f.subscription.StartsOn.Add(-24 * time.Hour)
	if _, err := f.service.UpdateSubscription(ctx, f.owner, shifted); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for a shifted StartsOn, got %v", err)
	}

	// The corrected client omits StartsOn instead of reconstructing it; the
	// backend's IsZero fallback must keep the exact original instant and let
	// the historical edit through.
	omitted := *f.subscription
	omitted.StartsOn = time.Time{}
	omitted.RevisionScope = "one_off"
	omitted.EffectiveBillingAt = f.pastBilling
	omitted.SplitMode = domain.SplitAmount
	omitted.Splits = []domain.ExpenseSplit{
		{UserID: f.owner, AmountMinor: 30000},
		{UserID: f.member, AmountMinor: 0},
	}
	if _, err := f.service.UpdateSubscription(ctx, f.owner, omitted); err != nil {
		t.Fatalf("expected the historical edit to succeed with StartsOn omitted, got %v", err)
	}

	dashboard, err := f.service.WorkspaceDashboard(ctx, f.member, application.DashboardQuery{Scope: "group", GroupID: f.group.ID, Month: "2024-09"})
	if err != nil {
		t.Fatal(err)
	}
	if len(dashboard.Currencies) != 1 || dashboard.Currencies[0].PersonalShareMinor != 0 {
		t.Fatalf("expected the revision to apply, got %#v", dashboard.Currencies)
	}
}

// scope=future with a historical EffectiveBillingAt and no EndBillingAt
// means "this period onward, indefinitely" — every month at or after the
// edited period should reflect it, not just the one exact period.
func TestUpdateSubscriptionFutureScopeHistoricalEditAppliesOnwardIndefinitely(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	secondBilling := f.pastBilling.AddDate(0, 1, 0) // 2024-10-11

	edit := *f.subscription
	edit.RevisionScope = "future"
	edit.EffectiveBillingAt = secondBilling
	edit.SplitMode = domain.SplitAmount
	edit.Splits = []domain.ExpenseSplit{
		{UserID: f.owner, AmountMinor: 30000},
		{UserID: f.member, AmountMinor: 0},
	}
	if _, err := f.service.UpdateSubscription(ctx, f.owner, edit); err != nil {
		t.Fatalf("owner should be allowed to revise a period onward: %v", err)
	}

	before, err := f.service.WorkspaceDashboard(ctx, f.member, application.DashboardQuery{Scope: "group", GroupID: f.group.ID, Month: "2024-09"})
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Currencies) != 1 || before.Currencies[0].PersonalShareMinor != 15000 {
		t.Fatalf("expected the period before the edit to keep the original 15000 share, got %#v", before.Currencies)
	}

	for _, month := range []string{"2024-10", "2024-11", "2024-12"} {
		dashboard, err := f.service.WorkspaceDashboard(ctx, f.member, application.DashboardQuery{Scope: "group", GroupID: f.group.ID, Month: month})
		if err != nil {
			t.Fatal(err)
		}
		if len(dashboard.Currencies) != 1 || dashboard.Currencies[0].PersonalShareMinor != 0 {
			t.Fatalf("expected %s to reflect the onward edit (share 0), got %#v", month, dashboard.Currencies)
		}
	}
}

// scope=future with both EffectiveBillingAt and EndBillingAt bounds the edit
// to a closed A-through-B range; periods outside the range must revert to
// whatever would otherwise apply (here, the subscription's original split).
func TestUpdateSubscriptionBoundedRangeAppliesOnlyWithinRange(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	start := f.pastBilling.AddDate(0, 1, 0) // 2024-10-11
	end := f.pastBilling.AddDate(0, 2, 0)   // 2024-11-11

	edit := *f.subscription
	edit.RevisionScope = "future"
	edit.EffectiveBillingAt = start
	edit.EndBillingAt = &end
	edit.SplitMode = domain.SplitAmount
	edit.Splits = []domain.ExpenseSplit{
		{UserID: f.owner, AmountMinor: 30000},
		{UserID: f.member, AmountMinor: 0},
	}
	if _, err := f.service.UpdateSubscription(ctx, f.owner, edit); err != nil {
		t.Fatalf("owner should be allowed to revise a bounded range: %v", err)
	}

	cases := map[string]int64{
		"2024-09": 15000, // before the range: untouched
		"2024-10": 0,     // in range
		"2024-11": 0,     // in range (inclusive end)
		"2024-12": 15000, // after the range: reverts to the original split
	}
	for month, want := range cases {
		dashboard, err := f.service.WorkspaceDashboard(ctx, f.member, application.DashboardQuery{Scope: "group", GroupID: f.group.ID, Month: month})
		if err != nil {
			t.Fatal(err)
		}
		if len(dashboard.Currencies) != 1 || dashboard.Currencies[0].PersonalShareMinor != want {
			t.Fatalf("%s: want share %d, got %#v", month, want, dashboard.Currencies)
		}
	}
}

func TestUpdateSubscriptionEndBillingBeforeEffectiveRejected(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	end := f.pastBilling.AddDate(0, 0, 0)
	start := f.pastBilling.AddDate(0, 1, 0)

	edit := *f.subscription
	edit.RevisionScope = "future"
	edit.EffectiveBillingAt = start
	edit.EndBillingAt = &end
	if _, err := f.service.UpdateSubscription(ctx, f.owner, edit); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid when end_billing_at precedes effective_billing_at, got %v", err)
	}
}

// A historical edit must retroactively rewrite an already-posted period's
// real Expense (and its splits), not just create a revision that only
// governs still-unposted future occurrences.
func TestUpdateSubscriptionRetroactivelyRegeneratesPostedOccurrence(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	secondBilling := f.pastBilling.AddDate(0, 1, 0) // 2024-10-11

	revisions, err := f.stores.Subscriptions.ListRevisions(ctx, f.subscription.ID)
	if err != nil || len(revisions) == 0 {
		t.Fatalf("expected an initial revision, got %v (err=%v)", revisions, err)
	}
	originalRevisionID := revisions[0].ID

	postedExpense := domain.Expense{
		GroupID: f.group.ID, SubscriptionID: f.subscription.ID, Title: "YouTube", AmountMinor: 30000,
		Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD, PaidBy: f.owner, IncurredOn: secondBilling,
		SplitMode: domain.SplitEqual, Splits: []domain.ExpenseSplit{{UserID: f.owner, AmountMinor: 15000}, {UserID: f.member, AmountMinor: 15000}},
	}
	if err = f.stores.Expenses.Create(ctx, &postedExpense); err != nil {
		t.Fatal(err)
	}
	if err = f.stores.Expenses.ReplaceSplits(ctx, postedExpense.ID, postedExpense.Splits); err != nil {
		t.Fatal(err)
	}
	occurrence := domain.SubscriptionOccurrence{SubscriptionID: f.subscription.ID, RevisionID: originalRevisionID, ExpenseID: postedExpense.ID, BillingAt: secondBilling, Status: "posted"}
	if err = f.stores.Subscriptions.CreateOccurrence(ctx, &occurrence); err != nil {
		t.Fatal(err)
	}

	// AmountMinor deliberately stays non-zero for both sides: PocketBase's
	// NumberField treats a Required field's zero value as blank, which is a
	// pre-existing, unrelated quirk of the expense_splits schema (also
	// latent in the ordinary, non-historical posting path) — not something
	// this test is exercising.
	edit := *f.subscription
	edit.RevisionScope = "one_off"
	edit.EffectiveBillingAt = secondBilling
	edit.SplitMode = domain.SplitAmount
	edit.Splits = []domain.ExpenseSplit{
		{UserID: f.owner, AmountMinor: 29000},
		{UserID: f.member, AmountMinor: 1000},
	}
	if _, err = f.service.UpdateSubscription(ctx, f.owner, edit); err != nil {
		t.Fatalf("owner should be allowed to retroactively revise the posted period: %v", err)
	}

	updated, err := f.stores.Expenses.Get(ctx, postedExpense.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.AmountMinor != 30000 {
		t.Fatalf("expected the total amount to stay 30000, got %d", updated.AmountMinor)
	}
	updatedSplits, err := f.stores.Expenses.ListSplits(ctx, postedExpense.ID)
	if err != nil {
		t.Fatal(err)
	}
	memberShare := int64(-1)
	for _, split := range updatedSplits {
		if split.UserID == f.member {
			memberShare = split.AmountMinor
		}
	}
	if memberShare != 1000 {
		t.Fatalf("expected the posted expense's member split to be retroactively rewritten to 1000, got %d (splits=%#v)", memberShare, updated.Splits)
	}

	logs, err := f.stores.Audits.List(ctx, f.group.ID, ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 50}, Action: "subscription.occurrence_regenerated"})
	if err != nil {
		t.Fatal(err)
	}
	if logs.TotalItems != 1 {
		t.Fatalf("expected exactly 1 occurrence_regenerated audit entry, got %d: %#v", logs.TotalItems, logs.Items)
	}
	if logs.Items[0].ResourceID != postedExpense.ID {
		t.Fatalf("expected the audit entry to reference the regenerated expense, got %#v", logs.Items[0])
	}
}

func TestBillingDatesIncludePastRequiresPermission(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()

	owned, err := f.service.BillingDates(ctx, f.owner, f.subscription.ID, "", 12, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(owned.Dates) == 0 || !owned.Dates[0].Equal(f.pastBilling) {
		t.Fatalf("owner should see past billing dates starting at %s, got %#v", f.pastBilling.Format("2006-01-02"), owned.Dates)
	}

	// Without the permission the flag is ignored rather than failing, so the
	// picker still works and simply offers upcoming dates.
	limited, err := f.service.BillingDates(ctx, f.member, f.subscription.ID, "", 12, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(limited.Dates) == 0 {
		t.Fatal("expected upcoming billing dates for a member")
	}
	if limited.Dates[0].Before(f.subscription.NextBilling) {
		t.Fatalf("member must not receive past billing dates, got %s", limited.Dates[0].Format("2006-01-02"))
	}
}
