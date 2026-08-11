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

// Binding must not touch any historical expense_splits/settlements rows —
// they keep pointing at the placeholder's own ID, resolved only at read time
// (see resolvePlaceholderAliases).
func TestBindingPlaceholderLeavesHistoricalDataUntouched(t *testing.T) {
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

	placeholder, err := stores.Users.Get(ctx, membership.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if placeholder.LinkedUserID != realUserID {
		t.Fatalf("expected the placeholder to be linked to the real user, got %#v", placeholder)
	}

	splits, err := stores.Expenses.ListSplits(ctx, expense.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, split := range splits {
		if split.UserID == membership.UserID {
			found = true
		}
		if split.UserID == realUserID {
			t.Fatalf("binding must not rewrite historical splits, but found one already pointing at the real user: %#v", splits)
		}
	}
	if !found {
		t.Fatalf("expected the historical split to still reference the placeholder id, got %#v", splits)
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
	inv, err := collab.CreateInvitationBinding(ctx, ownerID, groupID, "real-person-3@example.com", bound.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = collab.AcceptInvitationByID(ctx, realUserID, inv.ID); err != nil {
		t.Fatal(err)
	}
	if err = base.RemoveMember(ctx, ownerID, groupID, bound.UserID); err == nil {
		t.Fatal("expected removing a bound placeholder to be refused")
	} else if err != domain.ErrConflict {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}
