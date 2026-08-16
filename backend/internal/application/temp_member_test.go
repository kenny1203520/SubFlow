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
	"subflow/internal/ports"
)

func newTempMemberFixture(t *testing.T) (*application.CollaborationService, adapters.Stores, string, string) {
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
	record.Set("email", "temp-member-owner@example.com")
	record.Set("name", "Owner")
	record.Set("timezone", "UTC")
	record.SetPassword("correct-horse-battery-staple")
	if err = app.Save(record); err != nil {
		t.Fatal(err)
	}
	ownerID := record.Id

	stores, err := adapters.New("pocketbase", app)
	if err != nil {
		t.Fatal(err)
	}
	base := application.New(stores)
	collab := &application.CollaborationService{Base: base, Environment: "development", AppURL: "http://localhost:8080"}

	ctx := context.Background()
	group := &domain.Group{Name: "Temp Member Group", Currency: domain.CurrencyTWD, Color: "#7057e8", OwnerID: ownerID, Timezone: "UTC"}
	if err = stores.Groups.Create(ctx, group); err != nil {
		t.Fatal(err)
	}
	if err = stores.Memberships.Create(ctx, &domain.Membership{GroupID: group.ID, UserID: ownerID, Role: domain.RoleOwner}); err != nil {
		t.Fatal(err)
	}

	return collab, stores, ownerID, group.ID
}

func newRealUser(t *testing.T, stores adapters.Stores, email string) string {
	t.Helper()
	created, err := stores.Users.Create(context.Background(), domain.SetupInput{Email: email, Password: "correct-horse-battery-staple", AdminName: "Real Person", DefaultTimezone: "UTC", DefaultCurrency: domain.CurrencyTWD})
	if err != nil {
		t.Fatal(err)
	}
	return created.ID
}

// A placeholder must be a fully functional participant the moment it's
// created — that's the whole point of letting splitting start before
// everyone has actually joined.
func TestCreateTempMemberCanBePaidByAndSplitParticipant(t *testing.T) {
	collab, _, ownerID, groupID := newTempMemberFixture(t)
	ctx := context.Background()

	membership, err := collab.Base.CreateTempMember(ctx, ownerID, groupID, "小明")
	if err != nil {
		t.Fatal(err)
	}
	if !membership.User.Placeholder {
		t.Fatalf("expected the created user to be flagged as a placeholder, got %#v", membership.User)
	}

	expense, err := collab.Base.CreateExpense(ctx, ownerID, domain.Expense{
		GroupID: groupID, Title: "Groceries", AmountMinor: 10000, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		PaidBy: membership.UserID, IncurredOn: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
		SplitMode: domain.SplitEqual, Splits: []domain.ExpenseSplit{{UserID: ownerID}, {UserID: membership.UserID}},
	})
	if err != nil {
		t.Fatalf("expected a placeholder to be usable as PaidBy and a split participant, got %v", err)
	}
	if len(expense.Splits) != 2 {
		t.Fatalf("expected 2 canonical splits, got %#v", expense.Splits)
	}
}

// Binding must eagerly repoint every historical expense_split to the real
// account and remove the placeholder entirely, rather than leaving it around
// as a permanent alias target (see Service.repointUserReferences).
func TestBindingPlaceholderRewritesHistoricalData(t *testing.T) {
	collab, stores, ownerID, groupID := newTempMemberFixture(t)
	ctx := context.Background()

	membership, err := collab.Base.CreateTempMember(ctx, ownerID, groupID, "小明")
	if err != nil {
		t.Fatal(err)
	}
	expense, err := collab.Base.CreateExpense(ctx, ownerID, domain.Expense{
		GroupID: groupID, Title: "Groceries", AmountMinor: 10000, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		PaidBy: ownerID, IncurredOn: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
		SplitMode: domain.SplitEqual, Splits: []domain.ExpenseSplit{{UserID: ownerID}, {UserID: membership.UserID}},
	})
	if err != nil {
		t.Fatal(err)
	}

	realUserID := newRealUser(t, stores, "real-person@example.com")
	inv, err := collab.CreateInvitationBinding(ctx, ownerID, groupID, "real-person@example.com", membership.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if inv.TargetPlaceholderID != membership.UserID {
		t.Fatalf("expected the invitation to carry the target placeholder id, got %#v", inv)
	}
	if _, err = collab.AcceptInvitationByID(ctx, realUserID, inv.ID); err != nil {
		t.Fatal(err)
	}

	if _, err = stores.Users.Get(ctx, membership.UserID); err == nil {
		t.Fatal("expected the placeholder's users row to be deleted after binding")
	}

	splits, err := stores.Expenses.ListSplits(ctx, expense.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, split := range splits {
		if split.UserID == realUserID {
			found = true
		}
		if split.UserID == membership.UserID {
			t.Fatalf("binding must rewrite historical splits, but found one still pointing at the deleted placeholder: %#v", splits)
		}
	}
	if !found {
		t.Fatalf("expected the historical split to now reference the real user, got %#v", splits)
	}
}

// If the real member being bound to already has a split on the same expense
// as the placeholder (e.g. they were both separately involved before their
// identities merged), the two rows must merge into one rather than violate
// the (expense, user) unique index with a blind rewrite.
func TestBindingPlaceholderMergesConflictingExpenseSplits(t *testing.T) {
	collab, stores, ownerID, groupID := newTempMemberFixture(t)
	ctx := context.Background()

	membership, err := collab.Base.CreateTempMember(ctx, ownerID, groupID, "小明")
	if err != nil {
		t.Fatal(err)
	}
	realUserID := newRealUser(t, stores, "real-person@example.com")
	// Already a member ahead of binding (e.g. joined separately before being
	// bound to this placeholder), so CanonicalSplits accepts them as a
	// third split participant below.
	if err = stores.Memberships.Create(ctx, &domain.Membership{GroupID: groupID, UserID: realUserID, Role: domain.RoleMember}); err != nil {
		t.Fatal(err)
	}

	expense, err := collab.Base.CreateExpense(ctx, ownerID, domain.Expense{
		GroupID: groupID, Title: "Groceries", AmountMinor: 30000, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		PaidBy: ownerID, IncurredOn: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
		SplitMode: domain.SplitAmount,
		Splits: []domain.ExpenseSplit{
			{UserID: ownerID, AmountMinor: 10000},
			{UserID: membership.UserID, AmountMinor: 10000},
			{UserID: realUserID, AmountMinor: 10000},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	inv, err := collab.CreateInvitationBinding(ctx, ownerID, groupID, "real-person@example.com", membership.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = collab.AcceptInvitationByID(ctx, realUserID, inv.ID); err != nil {
		t.Fatal(err)
	}

	splits, err := stores.Expenses.ListSplits(ctx, expense.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(splits) != 2 {
		t.Fatalf("expected the placeholder and real-user splits to merge into one row, got %d splits: %#v", len(splits), splits)
	}
	var merged *domain.ExpenseSplit
	for i := range splits {
		if splits[i].UserID == realUserID {
			merged = &splits[i]
		}
	}
	if merged == nil {
		t.Fatalf("expected a split for the real user, got %#v", splits)
	}
	if merged.AmountMinor != 20000 {
		t.Fatalf("expected the merged split to sum both amounts (20000), got %d", merged.AmountMinor)
	}
}

// A subscription's revisions each carry their own independent paid_by/splits
// snapshot — binding must walk every one of them, not just the subscription's
// current fields, since the placeholder can appear differently across
// revisions (as PaidBy in one, only inside Splits in another).
func TestBindingPlaceholderRewritesSubscriptionAcrossRevisions(t *testing.T) {
	collab, stores, ownerID, groupID := newTempMemberFixture(t)
	ctx := context.Background()

	membership, err := collab.Base.CreateTempMember(ctx, ownerID, groupID, "小明")
	if err != nil {
		t.Fatal(err)
	}
	sub, err := collab.Base.CreateSubscription(ctx, ownerID, domain.Subscription{
		GroupID: groupID, Name: "Shared Netflix", AmountMinor: 39900, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		BillingCycle: domain.BillingMonthly, StartsOn: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
		Status: domain.SubscriptionActive, PaidBy: ownerID, SplitMode: domain.SplitEqual,
		Splits: []domain.ExpenseSplit{{UserID: ownerID}, {UserID: membership.UserID}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// A second, directly-constructed revision where the placeholder is the
	// payer but not a split participant — the inverse shape of the initial
	// revision CreateSubscription made above.
	secondRevision := domain.SubscriptionRevision{
		SubscriptionID: sub.ID, Scope: "one_off", EffectiveBillingAt: sub.NextBilling.AddDate(0, 1, 0),
		Name: sub.Name, AmountMinor: sub.AmountMinor, Currency: sub.Currency, BaseCurrency: sub.BaseCurrency,
		RateMode: domain.RateAutomatic, PaidBy: membership.UserID, SplitMode: domain.SplitAmount,
		Splits: []domain.ExpenseSplit{{UserID: ownerID, AmountMinor: sub.AmountMinor}},
	}
	if err = stores.Subscriptions.CreateRevision(ctx, &secondRevision); err != nil {
		t.Fatal(err)
	}

	realUserID := newRealUser(t, stores, "real-person@example.com")
	inv, err := collab.CreateInvitationBinding(ctx, ownerID, groupID, "real-person@example.com", membership.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = collab.AcceptInvitationByID(ctx, realUserID, inv.ID); err != nil {
		t.Fatal(err)
	}

	revisions, err := stores.Subscriptions.ListRevisions(ctx, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(revisions) != 2 {
		t.Fatalf("expected both revisions to survive, got %d", len(revisions))
	}
	for _, revision := range revisions {
		if revision.PaidBy == membership.UserID {
			t.Fatalf("expected no revision to still list the deleted placeholder as PaidBy, got %#v", revision)
		}
		for _, split := range revision.Splits {
			if split.UserID == membership.UserID {
				t.Fatalf("expected no revision split to still reference the deleted placeholder, got %#v", revision)
			}
		}
	}
	firstFound, secondFound := false, false
	for _, revision := range revisions {
		for _, split := range revision.Splits {
			if split.UserID == realUserID {
				firstFound = true
			}
		}
		if revision.ID == secondRevision.ID && revision.PaidBy == realUserID {
			secondFound = true
		}
	}
	if !firstFound {
		t.Fatalf("expected the initial revision's split to now reference the real user, got %#v", revisions)
	}
	if !secondFound {
		t.Fatalf("expected the second revision's PaidBy to now reference the real user, got %#v", revisions)
	}
}

// Binding must rewrite settlement from_user/to_user/created_by, and must
// delete rather than leave self-referential a settlement that existed
// directly between the placeholder and the real user being bound to.
func TestBindingPlaceholderRewritesSettlementsAndDropsDegenerateOnes(t *testing.T) {
	collab, stores, ownerID, groupID := newTempMemberFixture(t)
	ctx := context.Background()

	membership, err := collab.Base.CreateTempMember(ctx, ownerID, groupID, "小明")
	if err != nil {
		t.Fatal(err)
	}
	realUserID := newRealUser(t, stores, "real-person@example.com")
	settledOn := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)

	ordinary := domain.Settlement{GroupID: groupID, FromUserID: ownerID, ToUserID: membership.UserID, CreatedBy: ownerID, AmountMinor: 500, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD, SettledOn: settledOn}
	if err = stores.Settlements.Create(ctx, &ordinary); err != nil {
		t.Fatal(err)
	}
	degenerate := domain.Settlement{GroupID: groupID, FromUserID: membership.UserID, ToUserID: realUserID, CreatedBy: ownerID, AmountMinor: 700, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD, SettledOn: settledOn}
	if err = stores.Settlements.Create(ctx, &degenerate); err != nil {
		t.Fatal(err)
	}

	inv, err := collab.CreateInvitationBinding(ctx, ownerID, groupID, "real-person@example.com", membership.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = collab.AcceptInvitationByID(ctx, realUserID, inv.ID); err != nil {
		t.Fatal(err)
	}

	remaining, err := stores.Settlements.List(ctx, groupID, ports.PageRequest{Page: 1, PerPage: 100})
	if err != nil {
		t.Fatal(err)
	}
	if remaining.TotalItems != 1 {
		t.Fatalf("expected the degenerate settlement to be dropped, leaving 1, got %d: %#v", remaining.TotalItems, remaining.Items)
	}
	kept := remaining.Items[0]
	if kept.ID != ordinary.ID {
		t.Fatalf("expected the ordinary settlement to survive, got %#v", kept)
	}
	if kept.ToUserID != realUserID {
		t.Fatalf("expected the ordinary settlement's to_user to now reference the real user, got %#v", kept)
	}
}

// A balance accrued while someone was still a placeholder must count toward
// their real account once bound, even though the stored split still
// references the placeholder's own id.
func TestWorkspaceDashboardFoldsBoundPlaceholderBalanceIntoRealUser(t *testing.T) {
	collab, stores, ownerID, groupID := newTempMemberFixture(t)
	ctx := context.Background()
	base := collab.Base
	now := time.Date(2026, time.August, 15, 0, 0, 0, 0, time.UTC)
	base.Now = func() time.Time { return now }

	membership, err := collab.Base.CreateTempMember(ctx, ownerID, groupID, "小明")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = base.CreateExpense(ctx, ownerID, domain.Expense{
		GroupID: groupID, Title: "Groceries", AmountMinor: 10000, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		PaidBy: ownerID, IncurredOn: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
		SplitMode: domain.SplitEqual, Splits: []domain.ExpenseSplit{{UserID: ownerID}, {UserID: membership.UserID}},
	}); err != nil {
		t.Fatal(err)
	}

	before, err := base.WorkspaceDashboard(ctx, ownerID, application.DashboardQuery{Scope: "group", GroupID: groupID, Month: "2026-08"})
	if err != nil {
		t.Fatal(err)
	}
	if balanceFor(before.Balances, membership.UserID) == 0 {
		t.Fatalf("expected the placeholder to owe a balance before binding, got %#v", before.Balances)
	}

	realUserID := newRealUser(t, stores, "real-person-2@example.com")
	inv, err := collab.CreateInvitationBinding(ctx, ownerID, groupID, "real-person-2@example.com", membership.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = collab.AcceptInvitationByID(ctx, realUserID, inv.ID); err != nil {
		t.Fatal(err)
	}

	after, err := base.WorkspaceDashboard(ctx, ownerID, application.DashboardQuery{Scope: "group", GroupID: groupID, Month: "2026-08"})
	if err != nil {
		t.Fatal(err)
	}
	if balanceFor(after.Balances, membership.UserID) != 0 {
		t.Fatalf("expected the placeholder's own balance bucket to disappear after binding, got %#v", after.Balances)
	}
	if balanceFor(after.Balances, realUserID) == 0 {
		t.Fatalf("expected the real user's balance to include what was accrued while still a placeholder, got %#v", after.Balances)
	}
}

func balanceFor(balances []domain.MemberBalance, userID string) int64 {
	for _, b := range balances {
		if b.UserID == userID {
			return b.AmountMinor
		}
	}
	return 0
}

// Removing a still-unbound placeholder cleans up its users row as before.
// The bound-placeholder guard in RemoveMember is now legacy-data-only (a
// binding done through the normal flow never leaves the placeholder around
// to reach it), so this simulates that legacy state directly through the
// store rather than through CollaborationService.accept.
func TestRemoveMemberGuardsBoundPlaceholderButCleansUpUnbound(t *testing.T) {
	collab, stores, ownerID, groupID := newTempMemberFixture(t)
	ctx := context.Background()
	base := collab.Base

	unbound, err := base.CreateTempMember(ctx, ownerID, groupID, "未綁定")
	if err != nil {
		t.Fatal(err)
	}
	if err = base.RemoveMember(ctx, ownerID, groupID, unbound.UserID); err != nil {
		t.Fatalf("expected removing an unbound placeholder to succeed, got %v", err)
	}
	if _, err = stores.Users.Get(ctx, unbound.UserID); err == nil {
		t.Fatal("expected the unbound placeholder's users row to be cleaned up")
	}

	bound, err := base.CreateTempMember(ctx, ownerID, groupID, "已綁定")
	if err != nil {
		t.Fatal(err)
	}
	realUserID := newRealUser(t, stores, "real-person-3@example.com")
	// Simulates a placeholder bound under the pre-repointUserReferences
	// design: LinkedUserID set directly, bypassing accept()'s eager rewrite.
	if err = stores.Users.LinkPlaceholder(ctx, bound.UserID, realUserID); err != nil {
		t.Fatal(err)
	}
	if err = base.RemoveMember(ctx, ownerID, groupID, bound.UserID); err == nil {
		t.Fatal("expected removing a legacy bound placeholder to be refused")
	} else if err != domain.ErrConflict {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}
