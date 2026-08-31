package application_test

import (
	"context"
	"testing"
	"time"

	"subflow/internal/domain"
)

func TestPersonalIncomePersistsSingleSplit(t *testing.T) {
	fixture := newMemberTransferFixture(t)
	ctx := context.Background()
	receivedOn := time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC)

	created, err := fixture.service.CreateIncome(ctx, fixture.ownerID, domain.Income{
		Title:       "Personal salary",
		AmountMinor: 12345,
		Currency:    domain.CurrencyTWD,
		ReceivedOn:  receivedOn,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.EarnedBy != fixture.ownerID || len(created.Splits) != 1 || created.Splits[0].UserID != fixture.ownerID || created.Splits[0].AmountMinor != 12345 || created.Splits[0].BaseAmountMinor != 12345 {
		t.Fatalf("personal income must receive one owner split: %#v", created)
	}

	stored, err := fixture.stores.Incomes.Get(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Splits) != 1 || stored.Splits[0].IncomeID != created.ID || stored.Splits[0].UserID != fixture.ownerID {
		t.Fatalf("personal income split was not stored in income_splits: %#v", stored.Splits)
	}

	updated, err := fixture.service.UpdateIncome(ctx, fixture.ownerID, domain.Income{
		ID:          created.ID,
		Title:       created.Title,
		AmountMinor: 20000,
		Currency:    domain.CurrencyTWD,
		ReceivedOn:  receivedOn,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Splits) != 1 || updated.Splits[0].UserID != fixture.ownerID || updated.Splits[0].AmountMinor != 20000 || updated.Splits[0].BaseAmountMinor != 20000 {
		t.Fatalf("personal income update must replace the single owner split: %#v", updated.Splits)
	}
}
