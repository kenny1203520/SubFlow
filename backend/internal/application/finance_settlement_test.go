package application_test

import (
	"context"
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

func TestCreateSettlementSelfToOtherAllowedWithCreatePermission(t *testing.T) {
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

func TestSettlementCreateRequiresExplicitCreatePermission(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	role, err := f.service.CreateGroupRole(ctx, f.ownerID, domain.Role{GroupID: f.groupID, Name: "Read-only settlements", Permissions: []string{"group.view", "ledger.settlements.read"}})
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.AssignGroupRole(ctx, f.ownerID, f.groupID, f.memberID, role.ID); err != nil {
		t.Fatal(err)
	}
	_, err = f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 500, SettledOn: time.Now()})
	if err != domain.ErrForbidden {
		t.Fatalf("expected a member without ledger.settlements.create to be forbidden, got %v", err)
	}
}

func TestCreateSettlementOnBehalfOfOthersAllowedWithManagePermission(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	role, err := f.service.CreateGroupRole(ctx, f.ownerID, domain.Role{GroupID: f.groupID, Name: "Treasurer", Permissions: []string{"group.view", "ledger.settlements.read", "ledger.settlements.create", "ledger.settlements.manage"}})
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

func TestDeleteSettlementCreatorAllowedWithDeletePermission(t *testing.T) {
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
		t.Fatalf("expected a non-creator without ledger.settlements.manage to be forbidden, got %v", err)
	}
	if err = f.service.DeleteSettlement(ctx, f.ownerID, settlement.ID); err != nil {
		t.Fatalf("expected the owner to delete any settlement, got %v", err)
	}
}

func TestSettlementDeleteRequiresExplicitDeletePermission(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	settlement, err := f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 500, SettledOn: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	role, err := f.service.CreateGroupRole(ctx, f.ownerID, domain.Role{GroupID: f.groupID, Name: "No delete", Permissions: []string{"group.view", "ledger.settlements.read", "ledger.settlements.create", "ledger.settlements.update"}})
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.AssignGroupRole(ctx, f.ownerID, f.groupID, f.memberID, role.ID); err != nil {
		t.Fatal(err)
	}
	if err = f.service.DeleteSettlement(ctx, f.memberID, settlement.ID); err != domain.ErrForbidden {
		t.Fatalf("expected a creator without ledger.settlements.delete to be forbidden, got %v", err)
	}
}

func TestUpdateSettlementCreatorAllowedWithUpdatePermission(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	settlement, err := f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{
		GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 500, SettledOn: time.Now(), Notes: "original",
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := f.service.UpdateSettlement(ctx, f.memberID, settlement.ID, domain.Settlement{
		FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 750, SettledOn: settlement.SettledOn, Notes: "revised",
	})
	if err != nil {
		t.Fatalf("expected the creator to edit their own settlement, got %v", err)
	}
	if updated.AmountMinor != 750 || updated.Notes != "revised" {
		t.Fatalf("expected the amount/notes to be updated, got %#v", updated)
	}
}

func TestUpdateSettlementRequiresPermissionForNonCreator(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	settlement, err := f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{
		GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 500, SettledOn: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	patch := domain.Settlement{FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 900, SettledOn: settlement.SettledOn}
	if _, err = f.service.UpdateSettlement(ctx, f.otherID, settlement.ID, patch); err != domain.ErrForbidden {
		t.Fatalf("expected a non-creator without ledger.settlements.manage to be forbidden, got %v", err)
	}
	if _, err = f.service.UpdateSettlement(ctx, f.ownerID, settlement.ID, patch); err != nil {
		t.Fatalf("expected the owner to edit any settlement, got %v", err)
	}
}

func TestSettlementUpdateRequiresExplicitUpdatePermission(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	settlement, err := f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 500, SettledOn: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	role, err := f.service.CreateGroupRole(ctx, f.ownerID, domain.Role{GroupID: f.groupID, Name: "No update", Permissions: []string{"group.view", "ledger.settlements.read", "ledger.settlements.create", "ledger.settlements.delete"}})
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.AssignGroupRole(ctx, f.ownerID, f.groupID, f.memberID, role.ID); err != nil {
		t.Fatal(err)
	}
	_, err = f.service.UpdateSettlement(ctx, f.memberID, settlement.ID, domain.Settlement{FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 900, SettledOn: settlement.SettledOn})
	if err != domain.ErrForbidden {
		t.Fatalf("expected a creator without ledger.settlements.update to be forbidden, got %v", err)
	}
}

func TestUpdateSettlementRejectsInvalidAmount(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	settlement, err := f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{
		GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 500, SettledOn: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.UpdateSettlement(ctx, f.memberID, settlement.ID, domain.Settlement{FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 0, SettledOn: settlement.SettledOn}); err != domain.ErrInvalid {
		t.Fatalf("expected a non-positive amount to be rejected, got %v", err)
	}
}

func TestUpdateSettlementRejectsSameFromAndToUser(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	settlement, err := f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{
		GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 500, SettledOn: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.UpdateSettlement(ctx, f.memberID, settlement.ID, domain.Settlement{FromUserID: f.memberID, ToUserID: f.memberID, AmountMinor: 500, SettledOn: settlement.SettledOn}); err != domain.ErrInvalid {
		t.Fatalf("expected fromUser==toUser to be rejected, got %v", err)
	}
}

func TestUpdateSettlementRejectsNonMemberParty(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	settlement, err := f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{
		GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 500, SettledOn: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.UpdateSettlement(ctx, f.memberID, settlement.ID, domain.Settlement{FromUserID: f.memberID, ToUserID: "not-a-member", AmountMinor: 500, SettledOn: settlement.SettledOn}); err != domain.ErrInvalid {
		t.Fatalf("expected a non-member party to be rejected, got %v", err)
	}
}

func TestUpdateSettlementPinsGroupCurrency(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	settlement, err := f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{
		GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 500, SettledOn: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := f.service.UpdateSettlement(ctx, f.memberID, settlement.ID, domain.Settlement{
		FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 900, SettledOn: settlement.SettledOn,
		Currency: "USD", BaseCurrency: "USD", ExchangeRate: "0.5",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Currency != domain.CurrencyTWD || updated.BaseCurrency != domain.CurrencyTWD || updated.ExchangeRate != "1" {
		t.Fatalf("expected currency/rate to stay pinned to the group's own currency at rate 1 regardless of the patch, got %#v", updated)
	}
}

func TestUpdateSettlementWritesSuccessAuditWithoutNotes(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	settlement, err := f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{
		GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 500, SettledOn: time.Now(), Notes: "private original note",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.UpdateSettlement(ctx, f.memberID, settlement.ID, domain.Settlement{
		FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 750, SettledOn: settlement.SettledOn, Notes: "private revised note",
	}); err != nil {
		t.Fatal(err)
	}
	entries, err := f.stores.Audits.List(ctx, f.groupID, ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 10}, Action: "settlement.updated", Outcome: "success"})
	if err != nil || len(entries.Items) != 1 {
		t.Fatalf("expected one successful settlement update audit, got %#v (%v)", entries, err)
	}
	if strings.Contains(entries.Items[0].Summary, "private") || !strings.Contains(entries.Items[0].Summary, "notes_changed") {
		t.Fatalf("expected redacted notes change summary, got %q", entries.Items[0].Summary)
	}
}

func TestUpdateSettlementWritesFailureAudit(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	settlement, err := f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{
		GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 500, SettledOn: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.UpdateSettlement(ctx, f.memberID, settlement.ID, domain.Settlement{FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 0, SettledOn: settlement.SettledOn}); err != domain.ErrInvalid {
		t.Fatalf("expected invalid update, got %v", err)
	}
	entries, err := f.stores.Audits.List(ctx, f.groupID, ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 10}, Action: "settlement.updated", Outcome: "failure"})
	if err != nil || len(entries.Items) != 1 || !strings.Contains(entries.Items[0].Summary, "invalid_amount") {
		t.Fatalf("expected invalid update audit, got %#v (%v)", entries, err)
	}
}

func TestListSettlementsFiltersByMemberAndDateRange(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	early := time.Now().AddDate(0, 0, -10)
	late := time.Now()
	memberSettlement, err := f.service.CreateSettlement(ctx, f.memberID, domain.Settlement{GroupID: f.groupID, FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 500, SettledOn: early})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.CreateSettlement(ctx, f.ownerID, domain.Settlement{GroupID: f.groupID, FromUserID: f.ownerID, ToUserID: f.otherID, AmountMinor: 700, SettledOn: late}); err != nil {
		t.Fatal(err)
	}

	byMember, err := f.service.ListSettlements(ctx, f.ownerID, f.groupID, ports.SettlementQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 10}, MemberID: f.memberID})
	if err != nil {
		t.Fatal(err)
	}
	if len(byMember.Items) != 1 || byMember.Items[0].ID != memberSettlement.ID {
		t.Fatalf("expected the member filter to return only %s's settlement, got %#v", f.memberID, byMember.Items)
	}

	byDate, err := f.service.ListSettlements(ctx, f.ownerID, f.groupID, ports.SettlementQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 10}, From: early.AddDate(0, 0, -1), To: early.AddDate(0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	if len(byDate.Items) != 1 || byDate.Items[0].ID != memberSettlement.ID {
		t.Fatalf("expected the date range filter to return only the early settlement, got %#v", byDate.Items)
	}
}
