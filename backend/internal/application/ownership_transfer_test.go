package application_test

import (
	"context"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"subflow/internal/adapters"
	"subflow/internal/adapters/pocketbase"
	"subflow/internal/application"
	"subflow/internal/domain"
)

type ownershipFixture struct {
	service                    *application.Service
	stores                     adapters.Stores
	groupID                    string
	ownerID, memberID, otherID string
}

func newOwnershipFixture(t *testing.T) ownershipFixture {
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
	memberID := createUser("member@example.com", "Member")
	otherID := createUser("other@example.com", "Other")

	group := &domain.Group{Name: "Ownership Group", Currency: domain.CurrencyTWD, Color: "#7057e8", OwnerID: ownerID, Timezone: "UTC"}
	if err = stores.Groups.Create(ctx, group); err != nil {
		t.Fatal(err)
	}
	if err = stores.Memberships.Create(ctx, &domain.Membership{GroupID: group.ID, UserID: ownerID, Role: domain.RoleOwner}); err != nil {
		t.Fatal(err)
	}
	if err = stores.Memberships.Create(ctx, &domain.Membership{GroupID: group.ID, UserID: memberID, Role: domain.RoleMember}); err != nil {
		t.Fatal(err)
	}
	if err = stores.Memberships.Create(ctx, &domain.Membership{GroupID: group.ID, UserID: otherID, Role: domain.RoleMember}); err != nil {
		t.Fatal(err)
	}

	return ownershipFixture{service: service, stores: stores, groupID: group.ID, ownerID: ownerID, memberID: memberID, otherID: otherID}
}

func (f ownershipFixture) roleIDByKey(t *testing.T, key string) string {
	t.Helper()
	roles, err := f.stores.Roles.List(context.Background(), "group", f.groupID)
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range roles {
		if role.Key == key {
			return role.ID
		}
	}
	t.Fatalf("expected a %q role to exist", key)
	return ""
}

func TestAssignGroupRoleCannotGrantOwnerRoleToSecondMember(t *testing.T) {
	f := newOwnershipFixture(t)
	ctx := context.Background()
	ownerRoleID := f.roleIDByKey(t, "owner")
	if err := f.service.AssignGroupRole(ctx, f.ownerID, f.groupID, f.memberID, ownerRoleID); err != domain.ErrForbidden {
		t.Fatalf("expected assigning the owner role via AssignGroupRole to be forbidden, got %v", err)
	}
	role, err := f.stores.Memberships.GetRole(ctx, f.groupID, f.memberID)
	if err != nil {
		t.Fatal(err)
	}
	if role != domain.RoleMember {
		t.Fatalf("expected the member's role to be unchanged, got %v", role)
	}
}

func TestOwnershipTransferOnlyCurrentOwnerCanCreate(t *testing.T) {
	f := newOwnershipFixture(t)
	ctx := context.Background()
	if _, err := f.service.CreateOwnershipTransfer(ctx, f.memberID, f.groupID, f.otherID); err != domain.ErrForbidden {
		t.Fatalf("expected a non-owner creating a transfer to be forbidden, got %v", err)
	}
}

func TestOwnershipTransferOnlyOnePendingAtATime(t *testing.T) {
	f := newOwnershipFixture(t)
	ctx := context.Background()
	if _, err := f.service.CreateOwnershipTransfer(ctx, f.ownerID, f.groupID, f.memberID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CreateOwnershipTransfer(ctx, f.ownerID, f.groupID, f.otherID); err != domain.ErrConflict {
		t.Fatalf("expected a second pending transfer to be rejected as a conflict, got %v", err)
	}
}

func TestOwnershipTransferOnlyTargetCanRespond(t *testing.T) {
	f := newOwnershipFixture(t)
	ctx := context.Background()
	transfer, err := f.service.CreateOwnershipTransfer(ctx, f.ownerID, f.groupID, f.memberID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.RespondOwnershipTransfer(ctx, f.otherID, transfer.ID, true); err != domain.ErrForbidden {
		t.Fatalf("expected a non-target responding to be forbidden, got %v", err)
	}
	if _, err = f.service.RespondOwnershipTransfer(ctx, f.ownerID, transfer.ID, true); err != domain.ErrForbidden {
		t.Fatalf("expected the initiator responding to their own transfer to be forbidden, got %v", err)
	}
}

func TestOwnershipTransferAcceptMovesOwnershipAndSwapsRoles(t *testing.T) {
	f := newOwnershipFixture(t)
	ctx := context.Background()
	ownerRoleID := f.roleIDByKey(t, "owner")

	transfer, err := f.service.CreateOwnershipTransfer(ctx, f.ownerID, f.groupID, f.memberID)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := f.service.RespondOwnershipTransfer(ctx, f.memberID, transfer.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != domain.OwnershipTransferAccepted {
		t.Fatalf("expected status accepted, got %v", accepted.Status)
	}

	group, err := f.stores.Groups.Get(ctx, f.groupID)
	if err != nil {
		t.Fatal(err)
	}
	if group.OwnerID != f.memberID {
		t.Fatalf("expected the group owner to be the accepting member, got %q", group.OwnerID)
	}

	newOwnerRole, err := f.stores.Memberships.GetRole(ctx, f.groupID, f.memberID)
	if err != nil {
		t.Fatal(err)
	}
	if newOwnerRole != domain.RoleOwner {
		t.Fatalf("expected the new owner's enum role to be owner, got %v", newOwnerRole)
	}
	oldOwnerRole, err := f.stores.Memberships.GetRole(ctx, f.groupID, f.ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if oldOwnerRole != domain.RoleMember {
		t.Fatalf("expected the old owner's enum role to be demoted to member, got %v", oldOwnerRole)
	}

	newOwnerPermissions, err := f.service.GroupPermissions(ctx, f.memberID, f.groupID)
	if err != nil {
		t.Fatal(err)
	}
	if !containsPermission(newOwnerPermissions, "group.roles.manage") {
		t.Fatalf("expected the new owner to hold group.roles.manage, got %v", newOwnerPermissions)
	}
	oldOwnerPermissions, err := f.service.GroupPermissions(ctx, f.ownerID, f.groupID)
	if err != nil {
		t.Fatal(err)
	}
	if containsPermission(oldOwnerPermissions, "group.roles.manage") {
		t.Fatalf("expected the demoted old owner to lose group.roles.manage, got %v", oldOwnerPermissions)
	}

	// The old owner can no longer grant the owner role away (they aren't
	// even a manager anymore), and the new owner can't either, since only
	// the transfer flow itself is allowed to move it.
	if err = f.service.AssignGroupRole(ctx, f.memberID, f.groupID, f.otherID, ownerRoleID); err != domain.ErrForbidden {
		t.Fatalf("expected the new owner to be unable to grant the owner role via AssignGroupRole, got %v", err)
	}
}

func TestOwnershipTransferDeclineLeavesOwnershipUnchanged(t *testing.T) {
	f := newOwnershipFixture(t)
	ctx := context.Background()
	transfer, err := f.service.CreateOwnershipTransfer(ctx, f.ownerID, f.groupID, f.memberID)
	if err != nil {
		t.Fatal(err)
	}
	declined, err := f.service.RespondOwnershipTransfer(ctx, f.memberID, transfer.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if declined.Status != domain.OwnershipTransferDeclined {
		t.Fatalf("expected status declined, got %v", declined.Status)
	}
	group, err := f.stores.Groups.Get(ctx, f.groupID)
	if err != nil {
		t.Fatal(err)
	}
	if group.OwnerID != f.ownerID {
		t.Fatalf("expected ownership to remain unchanged after a decline, got %q", group.OwnerID)
	}
}

func TestOwnershipTransferCancelByInitiator(t *testing.T) {
	f := newOwnershipFixture(t)
	ctx := context.Background()
	transfer, err := f.service.CreateOwnershipTransfer(ctx, f.ownerID, f.groupID, f.memberID)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.CancelOwnershipTransfer(ctx, f.otherID, transfer.ID); err != domain.ErrForbidden {
		t.Fatalf("expected a non-initiator cancelling to be forbidden, got %v", err)
	}
	if err = f.service.CancelOwnershipTransfer(ctx, f.ownerID, transfer.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.RespondOwnershipTransfer(ctx, f.memberID, transfer.ID, true); err != domain.ErrConflict {
		t.Fatalf("expected responding to a cancelled transfer to conflict, got %v", err)
	}
}

func containsPermission(permissions []string, target string) bool {
	for _, value := range permissions {
		if value == target || value == "*" {
			return true
		}
	}
	return false
}
