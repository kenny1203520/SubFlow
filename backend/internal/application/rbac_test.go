package application_test

import (
	"context"
	"errors"
	"testing"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

func defaultAuditQuery() ports.AuditQuery {
	return ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 50}}
}

func adminSystemRoleID(t *testing.T, f *historicalFixture) string {
	t.Helper()
	roles, err := f.stores.Roles.List(context.Background(), "system", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range roles {
		if role.Key == "admin" {
			return role.ID
		}
	}
	t.Fatal("expected the seeded admin system role to exist")
	return ""
}

// The seeded "owner" group role holds the full permission set and "member"
// holds a deliberately narrower one (see ensureGroupRoleSeeds in
// schema.go) — GroupPermissions must resolve each member to their own role's
// permissions, not to some shared default.
func TestGroupPermissionsResolvesSeededOwnerAndMemberRoles(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()

	ownerPermissions, err := f.service.GroupPermissions(ctx, f.owner, f.group.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"group.roles.manage", "group.members.manage", "ledger.records.historical_write", "group.audit.read"} {
		if !contains(ownerPermissions, want) {
			t.Fatalf("expected owner permissions to include %q, got %#v", want, ownerPermissions)
		}
	}

	memberPermissions, err := f.service.GroupPermissions(ctx, f.member, f.group.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"group.roles.manage", "group.members.manage", "ledger.records.historical_write", "group.audit.read"} {
		if contains(memberPermissions, forbidden) {
			t.Fatalf("expected the seeded member role to NOT include %q, got %#v", forbidden, memberPermissions)
		}
	}
	if !contains(memberPermissions, "ledger.expenses.write") {
		t.Fatalf("expected the seeded member role to still include ledger.expenses.write, got %#v", memberPermissions)
	}
}

func TestGroupPermissionsForbiddenForNonMember(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	if _, err := f.service.GroupPermissions(ctx, "not-a-member", f.group.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for a non-member, got %v", err)
	}
}

func TestCreateGroupRoleRequiresPermission(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	if _, err := f.service.CreateGroupRole(ctx, f.member, domain.Role{GroupID: f.group.ID, Name: "Billing", Permissions: []string{"ledger.expenses.read"}}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for a member without group.roles.manage, got %v", err)
	}
	role, err := f.service.CreateGroupRole(ctx, f.owner, domain.Role{GroupID: f.group.ID, Name: "Billing", Permissions: []string{"ledger.expenses.read"}})
	if err != nil {
		t.Fatalf("expected the owner to be allowed to create a role: %v", err)
	}
	if role.Key == "" || role.Scope != "group" {
		t.Fatalf("expected a generated key and group scope, got %#v", role)
	}
}

func TestCreateGroupRoleRejectsDuplicateKey(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	if _, err := f.service.CreateGroupRole(ctx, f.owner, domain.Role{GroupID: f.group.ID, Name: "Custom Owner", Key: "owner", Permissions: []string{"group.view"}}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict for a key colliding with the seeded owner role, got %v", err)
	}
}

func TestCreateGroupRoleRejectsProtectedFlag(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	if _, err := f.service.CreateGroupRole(ctx, f.owner, domain.Role{GroupID: f.group.ID, Name: "Sneaky", Protected: true}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid when a caller tries to self-mark a new role protected, got %v", err)
	}
}

func TestUpdateGroupRoleRejectsProtectedRole(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	roles, err := f.service.ListGroupRoles(ctx, f.owner, f.group.ID)
	if err != nil {
		t.Fatal(err)
	}
	var ownerRole domain.Role
	for _, role := range roles {
		if role.Key == "owner" {
			ownerRole = role
		}
	}
	if ownerRole.ID == "" {
		t.Fatal("expected to find the seeded owner role")
	}
	ownerRole.Name = "Renamed"
	if _, err = f.service.UpdateGroupRole(ctx, f.owner, ownerRole); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden when updating a protected role, got %v", err)
	}
}

func TestDeleteGroupRoleRejectsProtectedRoleAndStillAssignedRole(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	roles, err := f.service.ListGroupRoles(ctx, f.owner, f.group.ID)
	if err != nil {
		t.Fatal(err)
	}
	var memberRoleID, ownerRoleID string
	for _, role := range roles {
		if role.Key == "member" {
			memberRoleID = role.ID
		}
		if role.Key == "owner" {
			ownerRoleID = role.ID
		}
	}
	if memberRoleID == "" || ownerRoleID == "" {
		t.Fatal("expected to find the seeded owner and member roles")
	}

	if err = f.service.DeleteGroupRole(ctx, f.owner, ownerRoleID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden deleting a protected role, got %v", err)
	}

	custom, err := f.service.CreateGroupRole(ctx, f.owner, domain.Role{GroupID: f.group.ID, Name: "Billing", Permissions: []string{"ledger.expenses.read"}})
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.AssignGroupRole(ctx, f.owner, f.group.ID, f.member, custom.ID); err != nil {
		t.Fatalf("owner should be able to assign the new role: %v", err)
	}
	if err = f.service.DeleteGroupRole(ctx, f.owner, custom.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict deleting a role still assigned to a member, got %v", err)
	}
}

// The protected "owner" group_roles record can only move through the
// ownership-transfer flow, never through general-purpose role assignment —
// otherwise a second member could end up with full owner permissions while
// Group.OwnerID still points at someone else.
func TestAssignGroupRoleCannotGrantOwnerRole(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	roles, err := f.service.ListGroupRoles(ctx, f.owner, f.group.ID)
	if err != nil {
		t.Fatal(err)
	}
	var ownerRoleID string
	for _, role := range roles {
		if role.Key == "owner" {
			ownerRoleID = role.ID
		}
	}
	if ownerRoleID == "" {
		t.Fatal("expected to find the seeded owner role")
	}
	if err = f.service.AssignGroupRole(ctx, f.owner, f.group.ID, f.member, ownerRoleID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden assigning the owner role directly, got %v", err)
	}
}

func TestAssignGroupRoleRejectsRetargetingTheCurrentOwner(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	custom, err := f.service.CreateGroupRole(ctx, f.owner, domain.Role{GroupID: f.group.ID, Name: "Billing", Permissions: []string{"ledger.expenses.read"}})
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.AssignGroupRole(ctx, f.owner, f.group.ID, f.owner, custom.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden reassigning the group owner's own role, got %v", err)
	}
}

func TestListGroupAuditRequiresPermission(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	if _, err := f.service.ListGroupAudit(ctx, f.member, f.group.ID, defaultAuditQuery()); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for a member without group.audit.read, got %v", err)
	}
	if _, err := f.service.ListGroupAudit(ctx, f.owner, f.group.ID, defaultAuditQuery()); err != nil {
		t.Fatalf("expected the owner to be allowed to read the audit log: %v", err)
	}
}

func TestSystemPermissionsEmptyForUserWithoutSystemRole(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	permissions, err := f.service.SystemPermissions(ctx, f.owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(permissions) != 0 {
		t.Fatalf("expected no system permissions for a user with no system role assigned, got %#v", permissions)
	}
}

func TestCreateSystemRoleRequiresPermissionAndRejectsDuplicateKey(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	if _, err := f.service.CreateSystemRole(ctx, f.owner, domain.Role{Name: "Support"}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for a user without system.roles.manage, got %v", err)
	}
	if err := f.stores.Users.SetSystemRole(ctx, f.owner, adminSystemRoleID(t, f)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CreateSystemRole(ctx, f.owner, domain.Role{Name: "Duplicate Admin", Key: "admin"}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict for a key colliding with the seeded admin role, got %v", err)
	}
	role, err := f.service.CreateSystemRole(ctx, f.owner, domain.Role{Name: "Support", Permissions: []string{"system.audit.read"}})
	if err != nil {
		t.Fatalf("expected an admin to be allowed to create a system role: %v", err)
	}
	if role.Scope != "system" || role.Key == "" {
		t.Fatalf("expected a generated key and system scope, got %#v", role)
	}
}

// System and group roles live in entirely separate collections (see
// GetRoleRecord/roleCollection in the pocketbase adapter), so a group role's
// ID can never resolve as a system role in the first place -- this asserts
// that structural separation holds rather than a same-table scope check.
func TestAssignSystemRoleRejectsUnknownRoleID(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	if err := f.stores.Users.SetSystemRole(ctx, f.owner, adminSystemRoleID(t, f)); err != nil {
		t.Fatal(err)
	}
	roles, err := f.service.ListGroupRoles(ctx, f.owner, f.group.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.AssignSystemRole(ctx, f.owner, f.member, roles[0].ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound assigning a group role's ID as a system role, got %v", err)
	}
}

// RecoverSystemAdmin exists to bootstrap the very first system administrator
// and must refuse once one already holds the role, or any authenticated
// user could grant themselves admin after the fact.
func TestRecoverSystemAdminOnlyWorksOnce(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	if err := f.stores.Settings.Save(ctx, domain.SystemSettings{Initialized: true, SiteName: "Test", DefaultCurrency: domain.CurrencyTWD}); err != nil {
		t.Fatal(err)
	}
	if err := f.service.RecoverSystemAdmin(ctx, f.owner); err != nil {
		t.Fatalf("expected the first recovery to succeed: %v", err)
	}
	permissions, err := f.service.SystemPermissions(ctx, f.owner)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(permissions, "system.roles.manage") {
		t.Fatalf("expected the recovered admin to hold system.roles.manage, got %#v", permissions)
	}
	if err = f.service.RecoverSystemAdmin(ctx, f.member); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict once an admin already exists, got %v", err)
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
