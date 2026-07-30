package pocketbase

import (
	"context"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

func TestRepositoryCRUDContract(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	if err = EnsureSchema(app); err != nil {
		t.Fatal(err)
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(users)
	record.Set("email", "owner@example.com")
	record.Set("name", "Owner")
	record.Set("timezone", "Asia/Taipei")
	record.SetPassword("correct-horse-battery-staple")
	if err = app.Save(record); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	stores := NewStores(app)
	group := &domain.Group{Name: "Home", Currency: domain.CurrencyTWD, Color: "#7057e8", OwnerID: record.Id}
	if err = stores.Groups.Create(ctx, group); err != nil {
		t.Fatal(err)
	}
	if group.ID == "" {
		t.Fatal("group id was not assigned")
	}
	member := &domain.Membership{GroupID: group.ID, UserID: record.Id, Role: domain.RoleOwner}
	if err = stores.Memberships.Create(ctx, member); err != nil {
		t.Fatal(err)
	}
	if role, roleErr := stores.Memberships.GetRole(ctx, group.ID, record.Id); roleErr != nil || role != domain.RoleOwner {
		t.Fatalf("unexpected role %q: %v", role, roleErr)
	}
	now := time.Now().UTC().Truncate(time.Second)
	inv := &domain.Invitation{GroupID: group.ID, Email: "member@example.com", TokenHash: "contract-hash", Status: domain.InvitationPending, InvitedBy: record.Id, ExpiresAt: now.Add(7 * 24 * time.Hour)}
	if err = stores.Invitations.Create(ctx, inv); err != nil {
		t.Fatal(err)
	}
	if found, findErr := stores.Invitations.GetByTokenHash(ctx, inv.TokenHash); findErr != nil || found.ID != inv.ID {
		t.Fatalf("invitation lookup failed: %v", findErr)
	}
	sub := &domain.Subscription{GroupID: group.ID, Name: "Music", AmountMinor: 29900, Currency: domain.CurrencyTWD, BillingCycle: domain.BillingMonthly, NextBilling: now.AddDate(0, 0, 5), Status: domain.SubscriptionActive}
	if err = stores.Subscriptions.Create(ctx, sub); err != nil {
		t.Fatal(err)
	}
	exp := &domain.Expense{GroupID: group.ID, Title: "Dinner", AmountMinor: 120000, PaidBy: record.Id, IncurredOn: now}
	if err = stores.Expenses.Create(ctx, exp); err != nil {
		t.Fatal(err)
	}
	if list, listErr := stores.Subscriptions.List(ctx, group.ID, ports.PageRequest{Page: 1, PerPage: 20}); listErr != nil || len(list.Items) != 1 {
		t.Fatalf("subscription list contract failed: %#v %v", list, listErr)
	}
	if list, listErr := stores.Expenses.List(ctx, group.ID, ports.PageRequest{Page: 1, PerPage: 20}); listErr != nil || len(list.Items) != 1 {
		t.Fatalf("expense list contract failed: %#v %v", list, listErr)
	}
	if err = stores.Transactions.Within(ctx, func(tx context.Context) error {
		loaded, loadErr := stores.Groups.Get(tx, group.ID)
		if loadErr != nil {
			return loadErr
		}
		loaded.Description = "updated"
		return stores.Groups.Update(tx, loaded)
	}); err != nil {
		t.Fatal(err)
	}
	if err = stores.Subscriptions.Delete(ctx, sub.ID); err != nil {
		t.Fatal(err)
	}
	if err = stores.Expenses.Delete(ctx, exp.ID); err != nil {
		t.Fatal(err)
	}
}
