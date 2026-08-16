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

type memberTransferFixture struct {
	service                           *application.Service
	stores                            adapters.Stores
	groupID                           string
	ownerID, fromID, toID, outsiderID string
}

func newMemberTransferFixture(t *testing.T) memberTransferFixture {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	if err = pocketbase.EnsureSchema(app); err != nil {
		t.Fatal(err)
	}
	stores, err := adapters.New("pocketbase", app)
	if err != nil {
		t.Fatal(err)
	}
	service := application.New(stores)
	ctx := context.Background()

	createUser := func(email, name string) string {
		t.Helper()
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			t.Fatal(err)
		}
		record := core.NewRecord(users)
		record.Set("email", email)
		record.Set("name", name)
		record.SetPassword("correct-horse-battery-staple")
		if err = app.Save(record); err != nil {
			t.Fatal(err)
		}
		return record.Id
	}

	ownerID := createUser("owner@example.com", "Owner")
	fromID := createUser("from@example.com", "From")
	toID := createUser("to@example.com", "To")
	outsiderID := createUser("outsider@example.com", "Outsider")

	group := &domain.Group{Name: "Member Transfer Group", Currency: domain.CurrencyTWD, Color: "#7057e8", OwnerID: ownerID, Timezone: "UTC"}
	if err = stores.Groups.Create(ctx, group); err != nil {
		t.Fatal(err)
	}
	if err = stores.Memberships.Create(ctx, &domain.Membership{GroupID: group.ID, UserID: ownerID, Role: domain.RoleOwner}); err != nil {
		t.Fatal(err)
	}
	if err = stores.Memberships.Create(ctx, &domain.Membership{GroupID: group.ID, UserID: fromID, Role: domain.RoleMember}); err != nil {
		t.Fatal(err)
	}
	if err = stores.Memberships.Create(ctx, &domain.Membership{GroupID: group.ID, UserID: toID, Role: domain.RoleMember}); err != nil {
		t.Fatal(err)
	}

	return memberTransferFixture{service: service, stores: stores, groupID: group.ID, ownerID: ownerID, fromID: fromID, toID: toID, outsiderID: outsiderID}
}

func TestCreateMemberTransferRequiresMembersManagePermission(t *testing.T) {
	f := newMemberTransferFixture(t)
	ctx := context.Background()
	if _, err := f.service.CreateMemberTransfer(ctx, f.fromID, f.groupID, f.fromID, f.toID); err != domain.ErrForbidden {
		t.Fatalf("expected a plain member without group.members.manage to be forbidden, got %v", err)
	}
}

func TestCreateMemberTransferRejectsOwnerAsFromUser(t *testing.T) {
	f := newMemberTransferFixture(t)
	ctx := context.Background()
	if _, err := f.service.CreateMemberTransfer(ctx, f.ownerID, f.groupID, f.ownerID, f.toID); err != domain.ErrForbidden {
		t.Fatalf("expected transferring the owner's own identity away to be forbidden (use ownership transfer instead), got %v", err)
	}
}

func TestCreateMemberTransferRejectsSelfAndNonMembers(t *testing.T) {
	f := newMemberTransferFixture(t)
	ctx := context.Background()
	if _, err := f.service.CreateMemberTransfer(ctx, f.ownerID, f.groupID, f.fromID, f.fromID); err != domain.ErrInvalid {
		t.Fatalf("expected a self-transfer to be rejected, got %v", err)
	}
	if _, err := f.service.CreateMemberTransfer(ctx, f.ownerID, f.groupID, f.fromID, f.outsiderID); err != domain.ErrInvalid {
		t.Fatalf("expected a non-member target to be rejected, got %v", err)
	}
	if _, err := f.service.CreateMemberTransfer(ctx, f.ownerID, f.groupID, f.outsiderID, f.toID); err != domain.ErrInvalid {
		t.Fatalf("expected a non-member source to be rejected, got %v", err)
	}
}

// Unlike ownership transfer, a group can have several member transfers
// pending at once for different source members -- only a second transfer for
// the *same* source member should conflict.
func TestCreateMemberTransferConflictsOnlyPerFromUser(t *testing.T) {
	f := newMemberTransferFixture(t)
	ctx := context.Background()
	if _, err := f.service.CreateMemberTransfer(ctx, f.ownerID, f.groupID, f.fromID, f.toID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CreateMemberTransfer(ctx, f.ownerID, f.groupID, f.fromID, f.ownerID); err != domain.ErrConflict {
		t.Fatalf("expected a second pending transfer for the same source member to conflict, got %v", err)
	}
	third := newRealUser(t, f.stores, "third@example.com")
	if err := f.stores.Memberships.Create(ctx, &domain.Membership{GroupID: f.groupID, UserID: third, Role: domain.RoleMember}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CreateMemberTransfer(ctx, f.ownerID, f.groupID, third, f.toID); err != nil {
		t.Fatalf("expected a transfer for a different source member to succeed alongside the first, got %v", err)
	}
}

func TestMemberTransferOnlyTargetCanRespond(t *testing.T) {
	f := newMemberTransferFixture(t)
	ctx := context.Background()
	transfer, err := f.service.CreateMemberTransfer(ctx, f.ownerID, f.groupID, f.fromID, f.toID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.RespondMemberTransfer(ctx, f.fromID, transfer.ID, true); err != domain.ErrForbidden {
		t.Fatalf("expected the source member responding to be forbidden, got %v", err)
	}
	if _, err = f.service.RespondMemberTransfer(ctx, f.ownerID, transfer.ID, true); err != domain.ErrForbidden {
		t.Fatalf("expected the initiator responding to be forbidden, got %v", err)
	}
}

// Accepting must repoint every reference the source member had in the group
// to the target, and remove the source from the group entirely -- a full
// handover, exercising the same repointUserReferences helper the
// temp-member bind path uses (see temp_member_test.go).
func TestMemberTransferAcceptRepointsDataAndRemovesFromGroup(t *testing.T) {
	f := newMemberTransferFixture(t)
	ctx := context.Background()

	expense, err := f.service.CreateExpense(ctx, f.ownerID, domain.Expense{
		GroupID: f.groupID, Title: "Groceries", AmountMinor: 10000, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		PaidBy: f.fromID, IncurredOn: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
		SplitMode: domain.SplitEqual, Splits: []domain.ExpenseSplit{{UserID: f.ownerID}, {UserID: f.fromID}},
	})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := f.service.CreateSubscription(ctx, f.ownerID, domain.Subscription{
		GroupID: f.groupID, Name: "Shared Netflix", AmountMinor: 39900, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		BillingCycle: domain.BillingMonthly, StartsOn: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
		Status: domain.SubscriptionActive, PaidBy: f.fromID, SplitMode: domain.SplitEqual,
		Splits: []domain.ExpenseSplit{{UserID: f.ownerID}, {UserID: f.fromID}},
	})
	if err != nil {
		t.Fatal(err)
	}

	transfer, err := f.service.CreateMemberTransfer(ctx, f.ownerID, f.groupID, f.fromID, f.toID)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := f.service.RespondMemberTransfer(ctx, f.toID, transfer.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != domain.MemberTransferAccepted {
		t.Fatalf("expected status accepted, got %v", accepted.Status)
	}

	if _, err = f.stores.Memberships.GetRole(ctx, f.groupID, f.fromID); err == nil {
		t.Fatal("expected the source member to be removed from the group after the transfer")
	}

	updatedExpense, err := f.stores.Expenses.Get(ctx, expense.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updatedExpense.PaidBy != f.toID {
		t.Fatalf("expected the expense's paid_by to be repointed to the target, got %q", updatedExpense.PaidBy)
	}
	splits, err := f.stores.Expenses.ListSplits(ctx, expense.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, split := range splits {
		if split.UserID == f.fromID {
			t.Fatalf("expected no split to still reference the transferred-away member, got %#v", splits)
		}
	}

	updatedSub, err := f.stores.Subscriptions.Get(ctx, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updatedSub.PaidBy != f.toID {
		t.Fatalf("expected the subscription's paid_by to be repointed to the target, got %q", updatedSub.PaidBy)
	}
}

func TestMemberTransferDeclineLeavesDataUnchanged(t *testing.T) {
	f := newMemberTransferFixture(t)
	ctx := context.Background()
	transfer, err := f.service.CreateMemberTransfer(ctx, f.ownerID, f.groupID, f.fromID, f.toID)
	if err != nil {
		t.Fatal(err)
	}
	declined, err := f.service.RespondMemberTransfer(ctx, f.toID, transfer.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if declined.Status != domain.MemberTransferDeclined {
		t.Fatalf("expected status declined, got %v", declined.Status)
	}
	if _, err = f.stores.Memberships.GetRole(ctx, f.groupID, f.fromID); err != nil {
		t.Fatalf("expected the source member to remain in the group after a decline, got %v", err)
	}
}

// Unlike CancelOwnershipTransfer (initiator-only), cancelling a member
// transfer is gated on group.members.manage -- any authorized manager may
// withdraw it, not only whoever happened to propose it.
func TestMemberTransferCancelRequiresMembersManagePermission(t *testing.T) {
	f := newMemberTransferFixture(t)
	ctx := context.Background()
	transfer, err := f.service.CreateMemberTransfer(ctx, f.ownerID, f.groupID, f.fromID, f.toID)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.CancelMemberTransfer(ctx, f.toID, transfer.ID); err != domain.ErrForbidden {
		t.Fatalf("expected a member without group.members.manage to be forbidden from cancelling, got %v", err)
	}
	if err = f.service.CancelMemberTransfer(ctx, f.ownerID, transfer.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.RespondMemberTransfer(ctx, f.toID, transfer.ID, true); err != domain.ErrConflict {
		t.Fatalf("expected responding to a cancelled transfer to conflict, got %v", err)
	}
}
