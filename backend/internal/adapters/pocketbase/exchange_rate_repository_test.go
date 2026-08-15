package pocketbase

import (
	"context"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tests"

	"subflow/internal/domain"
)

// Upserting twice "for the same calendar day" with two different sub-day
// timestamps (provider jitter, or PocketBase's own datetime formatting) must
// update the same row, not silently insert a duplicate. Reproduces the bug
// class already fixed once in LatestExchangeRate's read path, this time on
// the write path.
func TestUpsertExchangeRateMatchesSameCalendarDayRegardlessOfSubDayTime(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	if err = EnsureSchema(app); err != nil {
		t.Fatal(err)
	}
	stores := NewStores(app)
	ctx := context.Background()

	first := &domain.ExchangeRate{
		BaseCurrency: domain.CurrencyUSD, QuoteCurrency: domain.CurrencyTWD,
		RateScaled: 3_100_000_00, Rate: "31", Provider: "test",
		EffectiveDate: time.Date(2026, time.August, 15, 0, 0, 1, 0, time.UTC),
		FetchedAt:     time.Now(),
	}
	if err = stores.ExchangeRates.Upsert(ctx, first); err != nil {
		t.Fatal(err)
	}

	second := &domain.ExchangeRate{
		BaseCurrency: domain.CurrencyUSD, QuoteCurrency: domain.CurrencyTWD,
		RateScaled: 3_150_000_00, Rate: "31.5", Provider: "test",
		EffectiveDate: time.Date(2026, time.August, 15, 23, 59, 0, 0, time.UTC),
		FetchedAt:     time.Now(),
	}
	if err = stores.ExchangeRates.Upsert(ctx, second); err != nil {
		t.Fatal(err)
	}

	if first.ID != second.ID {
		t.Fatalf("expected the second upsert to update the same row as the first (same calendar day), got separate IDs %q and %q", first.ID, second.ID)
	}

	records, err := app.FindRecordsByFilter(CollectionExchangeRates, "base_currency='USD' && quote_currency='TWD'", "", 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected exactly 1 stored row for the day, got %d", len(records))
	}
	if got := int64(records[0].GetFloat("rate_scaled")); got != second.RateScaled {
		t.Fatalf("expected the row to hold the second (latest) rate_scaled %d, got %d", second.RateScaled, got)
	}

	// A different calendar day must still insert a separate row.
	third := &domain.ExchangeRate{
		BaseCurrency: domain.CurrencyUSD, QuoteCurrency: domain.CurrencyTWD,
		RateScaled: 3_200_000_00, Rate: "32", Provider: "test",
		EffectiveDate: time.Date(2026, time.August, 16, 0, 0, 0, 0, time.UTC),
		FetchedAt:     time.Now(),
	}
	if err = stores.ExchangeRates.Upsert(ctx, third); err != nil {
		t.Fatal(err)
	}
	if third.ID == second.ID {
		t.Fatal("expected a different calendar day to insert a new row, not update the previous day's")
	}
}
