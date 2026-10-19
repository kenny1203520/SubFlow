package pocketbase

import (
	"context"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"subflow/internal/domain"
	"testing"
	"time"
)

func TestIncomeCategoryMigrationSplitsSharedAndIsIdempotent(t *testing.T) {
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
	u := core.NewRecord(users)
	u.Set("email", "category-migration@example.com")
	u.SetPassword("correct-horse-battery-staple")
	if err = app.Save(u); err != nil {
		t.Fatal(err)
	}
	stores := NewStores(app)
	ctx := context.Background()
	category := func(name string) *domain.Category {
		t.Helper()
		c := &domain.Category{Scope: "personal", OwnerID: u.Id, CustomName: name, IconKey: "tag", Archived: true}
		if err := stores.Categories.Create(ctx, c); err != nil {
			t.Fatal(err)
		}
		r, err := app.FindRecordById(CollectionCategories, c.ID)
		if err != nil {
			t.Fatal(err)
		}
		r.Set("kind", "")
		if err = app.Save(r); err != nil {
			t.Fatal(err)
		}
		return c
	}
	shared, only, unused := category("Shared"), category("Income only"), category("Unused")
	makeIncome := func(c *domain.Category) *domain.Income {
		t.Helper()
		v := &domain.Income{OwnerID: u.Id, EarnedBy: u.Id, Title: c.CustomName, CategoryID: c.ID, AmountMinor: 10000, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD, ReceivedOn: time.Now().UTC()}
		if err := stores.Incomes.Create(ctx, v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	i1, i2 := makeIncome(shared), makeIncome(only)
	e := &domain.Expense{OwnerID: u.Id, PaidBy: u.Id, Title: "Expense", CategoryID: shared.ID, AmountMinor: 10000, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD, IncurredOn: time.Now().UTC()}
	if err = stores.Expenses.Create(ctx, e); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 2; n++ {
		if err = EnsureSchema(app); err != nil {
			t.Fatal(err)
		}
		got, err := stores.Incomes.Get(ctx, i1.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.CategoryID == shared.ID {
			t.Fatal("shared category was not split")
		}
		clone, err := stores.Categories.Get(ctx, got.CategoryID)
		if err != nil {
			t.Fatal(err)
		}
		if clone.Kind != "income" || clone.CustomName != "Shared" || !clone.Archived || clone.IconKey != "tag" {
			t.Fatalf("clone lost metadata: %#v", clone)
		}
		original, err := stores.Categories.Get(ctx, shared.ID)
		if err != nil {
			t.Fatal(err)
		}
		if original.Kind != "expense" || original.IncomeCategoryID != clone.ID {
			t.Fatalf("missing compatibility mapping: %#v", original)
		}
		got, err = stores.Incomes.Get(ctx, i2.ID)
		if err != nil || got.CategoryID != only.ID {
			t.Fatalf("income-only reference changed: %#v %v", got, err)
		}
		one, err := stores.Categories.Get(ctx, only.ID)
		if err != nil || one.Kind != "income" {
			t.Fatalf("income-only category: %#v %v", one, err)
		}
		un, err := stores.Categories.Get(ctx, unused.ID)
		if err != nil || un.Kind != "expense" {
			t.Fatalf("unused category: %#v %v", un, err)
		}
		all, err := stores.Categories.List(ctx, u.Id, "", true)
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, c := range all {
			if c.CustomName == "Shared" {
				count++
			}
		}
		if count != 2 {
			t.Fatalf("migration created duplicate categories: %d", count)
		}
	}
}
