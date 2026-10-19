package application_test

import (
	"context"
	"errors"
	"subflow/internal/domain"
	"testing"
	"time"
)

func TestIncomeCategoryKindsAndAtomicAudits(t *testing.T) {
	f := newMemberTransferFixture(t)
	ctx := context.Background()
	expense, err := f.service.CreateCategory(ctx, f.ownerID, domain.Category{Scope: "personal", Kind: "expense", CustomName: "Other"})
	if err != nil {
		t.Fatal(err)
	}
	income, err := f.service.CreateCategory(ctx, f.ownerID, domain.Category{Scope: "personal", Kind: "income", CustomName: "Other"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.CreateIncome(ctx, f.ownerID, domain.Income{Title: "Salary", CategoryID: expense.ID, AmountMinor: 100, Currency: domain.CurrencyTWD, ReceivedOn: time.Now().UTC()}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expense category accepted: %v", err)
	}
	if _, err = f.service.CreateIncome(ctx, f.ownerID, domain.Income{Title: "Salary", CategoryID: income.ID, AmountMinor: 100, Currency: domain.CurrencyTWD, ReceivedOn: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	values, err := f.service.ListCategories(ctx, f.ownerID, "personal", "", false, "income")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range values {
		if v.Kind != "income" {
			t.Fatalf("wrong kind: %#v", v)
		}
	}
	f.service.Stores.Audits = unavailableIncomeAudit{f.stores.Audits}
	if _, err = f.service.UpdateCategory(ctx, f.ownerID, domain.Category{ID: income.ID, CustomName: "Changed"}); !errors.Is(err, domain.ErrAuditUnavailable) {
		t.Fatalf("audit failure: %v", err)
	}
	stored, err := f.stores.Categories.Get(ctx, income.ID)
	if err != nil || stored.CustomName != "Other" {
		t.Fatalf("category escaped rollback: %#v %v", stored, err)
	}
}
