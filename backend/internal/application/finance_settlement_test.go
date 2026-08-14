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

// settlementFixture builds a group with an owner and two additional members
// (a plain default-role member and, on request, a member holding a custom
// role) so settlement authorization can be exercised from every angle.
type settlementFixture struct {
	service                    *application.Service
	stores                     adapters.Stores
	groupID                    string
	ownerID, memberID, otherID string
}

func newSettlementFixture(t *testing.T) settlementFixture {
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

	ownerID := createUser("settlement-owner@example.com", "Owner")
	memberID := createUser("settlement-member@example.com", "Member")
	otherID := createUser("settlement-other@example.com", "Other")

	group := &domain.Group{Name: "Settlement Group", Currency: domain.CurrencyTWD, Color: "#7057e8", OwnerID: ownerID, Timezone: "UTC"}
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

	return settlementFixture{service: service, stores: stores, groupID: group.ID, ownerID: ownerID, memberID: memberID, otherID: otherID}
}

func TestCreateSettlementSelfToOtherIsAlwaysAllowed(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	settlement, err := f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{
		GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 500, SettledOn: time.Now(),
	})
	if err != nil {
		t.Fatalf("expected a default member to record their own repayment, got %v", err)
	}
	if settlement.CreatedBy != f.memberID {
		t.Fatalf("expected CreatedBy to be the recording member, got %q", settlement.CreatedBy)
	}
}

func TestCreateSettlementOnBehalfOfOthersRequiresPermission(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	_, err := f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{
		GroupID: f.groupID, FromUserID: f.otherID, ToUserID: f.ownerID, AmountMinor: 500, SettledOn: time.Now(),
	})
	if err != domain.ErrForbidden {
		t.Fatalf("expected a default member recording someone else's repayment to be forbidden, got %v", err)
	}
}

func TestCreateSettlementOnBehalfOfOthersAllowedWithGrantedPermission(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	role, err := f.service.CreateGroupRole(ctx, f.ownerID, domain.Role{GroupID: f.groupID, Name: "Treasurer", Permissions: []string{"group.view", "ledger.expenses.read", "ledger.settlements.read", "ledger.settlements.write"}})
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.AssignGroupRole(ctx, f.ownerID, f.groupID, f.memberID, role.ID); err != nil {
		t.Fatal(err)
	}
	settlement, err := f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{
		GroupID: f.groupID, FromUserID: f.otherID, ToUserID: f.ownerID, AmountMinor: 500, SettledOn: time.Now(),
	})
	if err != nil {
		t.Fatalf("expected the treasurer role to permit recording another member's repayment, got %v", err)
	}
	if settlement.FromUserID != f.otherID || settlement.ToUserID != f.ownerID {
		t.Fatalf("expected the settlement to reflect the on-behalf-of parties, got %#v", settlement)
	}
}

func TestCreateSettlementOwnerCanAlwaysRecordOnBehalfOfOthers(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	_, err := f.service.CreateSettlement(ctx, f.ownerID, domain.Settlement{
		GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 500, SettledOn: time.Now(),
	})
	if err != nil {
		t.Fatalf("expected the owner to record a repayment on behalf of others, got %v", err)
	}
}

func TestDeleteSettlementCreatorCanAlwaysDelete(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	settlement, err := f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{
		GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 500, SettledOn: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.DeleteSettlement(ctx, f.memberID, settlement.ID); err != nil {
		t.Fatalf("expected the creator to delete their own settlement, got %v", err)
	}
}

func TestDeleteSettlementRequiresPermissionForNonCreator(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	settlement, err := f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{
		GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 500, SettledOn: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.DeleteSettlement(ctx, f.otherID, settlement.ID); err != domain.ErrForbidden {
		t.Fatalf("expected a non-creator without ledger.settlements.write to be forbidden, got %v", err)
	}
	if err = f.service.DeleteSettlement(ctx, f.ownerID, settlement.ID); err != nil {
		t.Fatalf("expected the owner to delete any settlement, got %v", err)
	}
}
