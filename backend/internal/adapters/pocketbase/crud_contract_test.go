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
	category := &domain.Category{Scope: "group", GroupID: group.ID, CreatedBy: record.Id, CustomName: "Shared food"}
	if err = stores.Categories.Create(ctx, category); err != nil {
		t.Fatal(err)
	}
	if categories, listErr := stores.Categories.List(ctx, record.Id, group.ID, false); listErr != nil {
		t.Fatalf("group category list contract failed: %v", listErr)
	} else if !containsCategory(categories, category.ID) {
		t.Fatalf("group category was not returned: %#v", categories)
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
	sub := &domain.Subscription{GroupID: group.ID, Name: "Music", AmountMinor: 29900, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD, RateMode: domain.RateAutomatic, BillingCycle: domain.BillingMonthly, NextBilling: now.AddDate(0, 0, 5), Status: domain.SubscriptionActive}
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
	if automatic, listErr := stores.Subscriptions.ListAutomatic(ctx); listErr != nil || len(automatic) != 1 || automatic[0].ID != sub.ID {
		t.Fatalf("automatic subscription list contract failed: %#v %v", automatic, listErr)
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

// Subscription splits live in a JSON column rather than a child collection, so
// they only survive a full write/read cycle through the record layer. The
// in-memory finance tests never touch the adapter and cannot catch a broken
// round-trip here.
func TestSubscriptionSplitsSurviveRoundTrip(t *testing.T) {
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
	members := make([]string, 0, 3)
	for _, email := range []string{"a@example.com", "b@example.com", "c@example.com"} {
		record := core.NewRecord(users)
		record.Set("email", email)
		record.Set("name", email)
		record.Set("timezone", "Asia/Taipei")
		record.SetPassword("correct-horse-battery-staple")
		if err = app.Save(record); err != nil {
			t.Fatal(err)
		}
		members = append(members, record.Id)
	}
	ctx := context.Background()
	stores := NewStores(app)
	group := &domain.Group{Name: "Home", Currency: domain.CurrencyTWD, Color: "#7057e8", OwnerID: members[0]}
	if err = stores.Groups.Create(ctx, group); err != nil {
		t.Fatal(err)
	}
	splits := []domain.ExpenseSplit{
		{UserID: members[0], AmountMinor: 10000, BaseAmountMinor: 10000},
		{UserID: members[1], AmountMinor: 10000, BaseAmountMinor: 10000},
		{UserID: members[2], AmountMinor: 10000, BaseAmountMinor: 10000},
	}
	now := time.Now().UTC().Truncate(time.Second)
	sub := &domain.Subscription{GroupID: group.ID, Name: "YouTube", AmountMinor: 30000, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD, BaseAmountMinor: 30000, RateMode: domain.RateAutomatic, BillingCycle: domain.BillingMonthly, NextBilling: now.AddDate(0, 0, 5), Status: domain.SubscriptionActive, PaidBy: members[0], SplitMode: domain.SplitEqual, Splits: splits}
	if err = stores.Subscriptions.Create(ctx, sub); err != nil {
		t.Fatal(err)
	}
	loaded, err := stores.Subscriptions.Get(ctx, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertSplits(t, "subscription", loaded.Splits, splits)

	revision := &domain.SubscriptionRevision{SubscriptionID: sub.ID, Scope: "future", EffectiveBillingAt: sub.NextBilling, Name: sub.Name, AmountMinor: sub.AmountMinor, Currency: sub.Currency, BaseCurrency: sub.BaseCurrency, BaseAmountMinor: sub.BaseAmountMinor, RateMode: sub.RateMode, PaidBy: sub.PaidBy, SplitMode: sub.SplitMode, Splits: splits}
	if err = stores.Subscriptions.CreateRevision(ctx, revision); err != nil {
		t.Fatal(err)
	}
	revisions, err := stores.Subscriptions.ListRevisions(ctx, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(revisions) != 1 {
		t.Fatalf("expected 1 revision, got %d", len(revisions))
	}
	assertSplits(t, "revision", revisions[0].Splits, splits)
}

func assertSplits(t *testing.T, label string, got, want []domain.ExpenseSplit) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s splits were not restored: want %d, got %d (%#v)", label, len(want), len(got), got)
	}
	found := map[string]domain.ExpenseSplit{}
	for _, split := range got {
		found[split.UserID] = split
	}
	for _, split := range want {
		actual, ok := found[split.UserID]
		if !ok {
			t.Fatalf("%s splits missing user %s: %#v", label, split.UserID, got)
		}
		if actual.AmountMinor != split.AmountMinor || actual.BaseAmountMinor != split.BaseAmountMinor {
			t.Fatalf("%s split for %s: want %d/%d, got %d/%d", label, split.UserID, split.AmountMinor, split.BaseAmountMinor, actual.AmountMinor, actual.BaseAmountMinor)
		}
	}
}

func containsCategory(categories []domain.Category, id string) bool {
	for _, category := range categories {
		if category.ID == id {
			return true
		}
	}
	return false
}
