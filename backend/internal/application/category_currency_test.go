package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"subflow/internal/domain"
)

func TestListCategoriesRejectsInvalidScopeAndRequiresMembership(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	if _, err := f.service.ListCategories(ctx, f.owner, "bogus", f.group.ID, false); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for an unknown scope, got %v", err)
	}
	if _, err := f.service.ListCategories(ctx, "not-a-member", "group", f.group.ID, false); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for a non-member listing group categories, got %v", err)
	}
	if _, err := f.service.ListCategories(ctx, f.owner, "group", f.group.ID, false); err != nil {
		t.Fatalf("expected the owner to list group categories: %v", err)
	}
}

func TestCreateCategoryRejectsEmptyNameAndBadScope(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	if _, err := f.service.CreateCategory(ctx, f.owner, domain.Category{Scope: "personal", CustomName: "   "}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for a blank name, got %v", err)
	}
	if _, err := f.service.CreateCategory(ctx, f.owner, domain.Category{Scope: "system", CustomName: "Sneaky"}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for a disallowed scope, got %v", err)
	}
}

func TestCreateCategoryGroupScopeRequiresMembership(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	if _, err := f.service.CreateCategory(ctx, "not-a-member", domain.Category{Scope: "group", GroupID: f.group.ID, CustomName: "Snacks"}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for a non-member creating a group category, got %v", err)
	}
}

func TestCreateCategoryRejectsDuplicateNameCaseInsensitive(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	if _, err := f.service.CreateCategory(ctx, f.owner, domain.Category{Scope: "group", GroupID: f.group.ID, CustomName: "Snacks"}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.CreateCategory(ctx, f.owner, domain.Category{Scope: "group", GroupID: f.group.ID, CustomName: "  SNACKS  "}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict for a case/whitespace-insensitive duplicate name, got %v", err)
	}
}

func TestUpdateCategoryRejectsSystemScope(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	systemCategories, err := f.stores.Categories.List(ctx, "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(systemCategories) == 0 {
		t.Fatal("expected seeded system categories to exist")
	}
	if _, err = f.service.UpdateCategory(ctx, f.owner, domain.Category{ID: systemCategories[0].ID, CustomName: "Hacked"}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden updating a system category, got %v", err)
	}
}

// A group category can be updated by whoever created it or by the group
// owner, but not by an unrelated member.
func TestUpdateCategoryRequiresCreatorOrGroupOwner(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	category, err := f.service.CreateCategory(ctx, f.member, domain.Category{Scope: "group", GroupID: f.group.ID, CustomName: "Snacks"})
	if err != nil {
		t.Fatal(err)
	}

	third := newRealUserForCategoryTest(t, f, "third-member@example.com")
	if _, err = f.service.UpdateCategory(ctx, third, domain.Category{ID: category.ID, CustomName: "Renamed"}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for an unrelated member, got %v", err)
	}

	if updated, err := f.service.UpdateCategory(ctx, f.member, domain.Category{ID: category.ID, CustomName: "Renamed by Creator"}); err != nil {
		t.Fatalf("expected the creator to be allowed to update their own category: %v", err)
	} else if updated.CustomName != "Renamed by Creator" {
		t.Fatalf("expected the name to be updated, got %#v", updated)
	}

	if _, err = f.service.UpdateCategory(ctx, f.owner, domain.Category{ID: category.ID, Archived: true}); err != nil {
		t.Fatalf("expected the group owner to also be allowed to update it: %v", err)
	}
}

func TestValidateCategoryRejectsArchivedAndCrossScope(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	personal, err := f.service.CreateCategory(ctx, f.owner, domain.Category{Scope: "personal", CustomName: "Personal Only"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.CreateExpense(ctx, f.owner, domain.Expense{
		GroupID: f.group.ID, Title: "Cross-scope", AmountMinor: 100, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		PaidBy: f.owner, IncurredOn: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC), CategoryID: personal.ID,
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden using a personal category on a group expense, got %v", err)
	}

	group, err := f.service.CreateCategory(ctx, f.owner, domain.Category{Scope: "group", GroupID: f.group.ID, CustomName: "Group Only"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.UpdateCategory(ctx, f.owner, domain.Category{ID: group.ID, Archived: true}); err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.CreateExpense(ctx, f.owner, domain.Expense{
		GroupID: f.group.ID, Title: "Archived category", AmountMinor: 100, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		PaidBy: f.owner, IncurredOn: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC), CategoryID: group.ID,
	}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid using an archived category, got %v", err)
	}
}

func TestPreviewAndChangeGroupCurrencyRequireOwner(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	if _, err := f.service.PreviewGroupCurrency(ctx, f.member, f.group.ID, domain.CurrencyUSD); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for a non-owner previewing a currency change, got %v", err)
	}
	if _, err := f.service.ChangeGroupCurrency(ctx, f.member, f.group.ID, domain.CurrencyUSD); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for a non-owner changing the currency, got %v", err)
	}
}

// Neither newHistoricalFixture's fixture app nor this test wire a live rate
// provider (Service.Rates stays nil), so any conversion between two
// different currencies with no cached rate is unavailable by construction --
// exactly the "missing rate" scenario the preview/change endpoints must
// surface and block on, with no extra mocking needed.
func TestPreviewGroupCurrencyReportsMissingRatesAndBlocksTheChange(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	incurredOn := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	expense, err := f.service.CreateExpense(ctx, f.owner, domain.Expense{
		GroupID: f.group.ID, Title: "Needs a rate", AmountMinor: 500, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		PaidBy: f.owner, IncurredOn: incurredOn,
	})
	if err != nil {
		t.Fatal(err)
	}

	preview, err := f.service.PreviewGroupCurrency(ctx, f.owner, f.group.ID, domain.CurrencyUSD)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, missing := range preview.Missing {
		if missing.ID == expense.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the expense with no cached rate to be reported missing, got %#v", preview.Missing)
	}

	if _, err = f.service.ChangeGroupCurrency(ctx, f.owner, f.group.ID, domain.CurrencyUSD); !errors.Is(err, domain.ErrRateUnavailable) {
		t.Fatalf("expected ErrRateUnavailable blocking the change while a rate is missing, got %v", err)
	}
}

func TestChangeGroupCurrencyRecalculatesAmountsOnceRatesAreAvailable(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	incurredOn := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	expense, err := f.service.CreateExpense(ctx, f.owner, domain.Expense{
		GroupID: f.group.ID, Title: "Convertible", AmountMinor: 1000, Currency: domain.CurrencyTWD, BaseCurrency: domain.CurrencyTWD,
		PaidBy: f.owner, IncurredOn: incurredOn,
	})
	if err != nil {
		t.Fatal(err)
	}
	// The fixture group also has a subscription (started 2024-09-11); its own
	// preview check needs a rate for its own StartsOn date too, or the
	// unrelated subscription alone would block the change.
	for _, date := range []time.Time{incurredOn, f.subscription.StartsOn} {
		rate := &domain.ExchangeRate{BaseCurrency: domain.CurrencyTWD, QuoteCurrency: domain.CurrencyUSD, RateScaled: 3_000_000, Provider: "test", EffectiveDate: date, FetchedAt: time.Now()}
		if err = f.stores.ExchangeRates.Upsert(ctx, rate); err != nil {
			t.Fatal(err)
		}
	}

	updatedGroup, err := f.service.ChangeGroupCurrency(ctx, f.owner, f.group.ID, domain.CurrencyUSD)
	if err != nil {
		t.Fatalf("expected the change to succeed once a rate is cached: %v", err)
	}
	if updatedGroup.Currency != domain.CurrencyUSD {
		t.Fatalf("expected the group currency to update to USD, got %q", updatedGroup.Currency)
	}
	updatedExpense, err := f.stores.Expenses.Get(ctx, expense.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updatedExpense.BaseCurrency != domain.CurrencyUSD {
		t.Fatalf("expected the expense's base currency to update to USD, got %q", updatedExpense.BaseCurrency)
	}
	if updatedExpense.BaseAmountMinor == 0 {
		t.Fatal("expected the expense's base amount to be recalculated to a non-zero value")
	}
}

func newRealUserForCategoryTest(t *testing.T, f *historicalFixture, email string) string {
	t.Helper()
	created, err := f.stores.Users.Create(context.Background(), domain.SetupInput{Email: email, Password: "correct-horse-battery-staple", AdminName: "Third Member", DefaultTimezone: "UTC", DefaultCurrency: domain.CurrencyTWD})
	if err != nil {
		t.Fatal(err)
	}
	if err = f.stores.Memberships.Create(context.Background(), &domain.Membership{GroupID: f.group.ID, UserID: created.ID, Role: domain.RoleMember}); err != nil {
		t.Fatal(err)
	}
	return created.ID
}
