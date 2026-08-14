package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tests"

	"subflow/internal/adapters"
	"subflow/internal/adapters/pocketbase"
	"subflow/internal/application"
	"subflow/internal/domain"
	"subflow/internal/ports"
)

// fakeRateProvider lets these tests control exactly what the external feed
// "returns" without a real HTTP call, including simulating a base currency
// that fails entirely (e.g. the feed is unreachable for that request).
type fakeRateProvider struct {
	quoteAllByBase map[domain.Currency]map[domain.Currency]*domain.ExchangeRate
	failBases      map[domain.Currency]bool
	quoteErr       error
}

func (f *fakeRateProvider) Quote(_ context.Context, from, to domain.Currency, date time.Time) (*domain.ExchangeRate, error) {
	if f.quoteErr != nil {
		return nil, f.quoteErr
	}
	if rates, ok := f.quoteAllByBase[from]; ok {
		if rate, ok := rates[to]; ok {
			return rate, nil
		}
	}
	return nil, domain.ErrRateUnavailable
}

func (f *fakeRateProvider) QuoteAll(_ context.Context, from domain.Currency, _ time.Time) (map[domain.Currency]*domain.ExchangeRate, error) {
	if f.failBases[from] {
		return nil, domain.ErrRateUnavailable
	}
	return f.quoteAllByBase[from], nil
}

func fakeRate(base, quote domain.Currency, scaled int64, effective time.Time) *domain.ExchangeRate {
	return &domain.ExchangeRate{BaseCurrency: base, QuoteCurrency: quote, RateScaled: scaled, EffectiveDate: effective, Provider: "fake", FetchedAt: effective}
}

func newExchangeTestService(t *testing.T) (*application.Service, adapters.Stores) {
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
	return application.New(stores), stores
}

func TestRefreshReferenceRatesWarmsEveryCurrencyFromOneQuoteAllCall(t *testing.T) {
	service, _ := newExchangeTestService(t)
	today := time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return today }
	service.Rates = &fakeRateProvider{quoteAllByBase: map[domain.Currency]map[domain.Currency]*domain.ExchangeRate{
		"USD": {
			"TWD": fakeRate("USD", "TWD", 32_50000000, today),
			"INR": fakeRate("USD", "INR", 83_00000000, today),
		},
		"EUR": {"TWD": fakeRate("EUR", "TWD", 35_00000000, today)},
		"JPY": {"TWD": fakeRate("JPY", "TWD", 21_00000000, today)},
		"GBP": {"TWD": fakeRate("GBP", "TWD", 41_00000000, today)},
		"TWD": {"USD": fakeRate("TWD", "USD", 3_00000000, today)},
	}}

	if err := service.RefreshReferenceRates(context.Background()); err != nil {
		t.Fatal(err)
	}

	// A pair that was only ever warmed via QuoteAll (not requested directly)
	// must already be cached: QuoteRate should resolve it purely from cache,
	// without falling through to the (nil, in this call) live provider.
	service.Rates = nil
	rate, err := service.QuoteRate(context.Background(), "USD", "INR", today)
	if err != nil {
		t.Fatalf("expected the INR rate to be pre-warmed by QuoteAll, got %v", err)
	}
	if rate.Rate != "83" {
		t.Fatalf("expected rate 83, got %s", rate.Rate)
	}

	twdToUSD, err := service.QuoteRate(context.Background(), "TWD", "USD", today)
	if err != nil {
		t.Fatalf("expected the reverse TWD->USD direction to also be pre-warmed, got %v", err)
	}
	if twdToUSD.Rate != "3" {
		t.Fatalf("expected rate 3, got %s", twdToUSD.Rate)
	}
}

func TestRefreshReferenceRatesAuditsFailureAndContinuesOtherBases(t *testing.T) {
	service, stores := newExchangeTestService(t)
	today := time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return today }
	service.Rates = &fakeRateProvider{
		failBases: map[domain.Currency]bool{"USD": true},
		quoteAllByBase: map[domain.Currency]map[domain.Currency]*domain.ExchangeRate{
			"EUR": {"TWD": fakeRate("EUR", "TWD", 35_00000000, today)},
			"JPY": {"TWD": fakeRate("JPY", "TWD", 21_00000000, today)},
			"GBP": {"TWD": fakeRate("GBP", "TWD", 41_00000000, today)},
			"TWD": {},
		},
	}

	if err := service.RefreshReferenceRates(context.Background()); err != nil {
		t.Fatalf("a single base failing should not abort the whole refresh, got %v", err)
	}

	// EUR->TWD still got warmed despite USD failing.
	service.Rates = nil
	if _, err := service.QuoteRate(context.Background(), "EUR", "TWD", today); err != nil {
		t.Fatalf("expected EUR->TWD to be warmed despite the USD failure, got %v", err)
	}

	logs, err := stores.Audits.List(context.Background(), "", ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 50}, Action: "exchange_rate.refresh_failed"})
	if err != nil {
		t.Fatal(err)
	}
	if logs.TotalItems != 1 {
		t.Fatalf("expected exactly 1 refresh-failure audit entry for USD, got %d: %#v", logs.TotalItems, logs.Items)
	}
	if logs.Items[0].Outcome != "failure" || logs.Items[0].ResourceID != "USD" {
		t.Fatalf("unexpected audit entry: %#v", logs.Items[0])
	}
}

func TestQuoteRateFallsBackToStaleCacheWhenProviderFails(t *testing.T) {
	service, stores := newExchangeTestService(t)
	today := time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return today }
	yesterday := today.AddDate(0, 0, -1)
	if err := stores.ExchangeRates.Upsert(context.Background(), fakeRate("USD", "TWD", 32_00000000, yesterday)); err != nil {
		t.Fatal(err)
	}
	service.Rates = &fakeRateProvider{quoteErr: domain.ErrRateUnavailable}

	rate, err := service.QuoteRate(context.Background(), "USD", "TWD", today)
	if err != nil {
		t.Fatalf("expected a stale cached fallback instead of an error, got %v", err)
	}
	if !rate.Stale {
		t.Fatalf("expected the fallback rate to be marked stale, got %#v", rate)
	}
	if rate.Rate != "32" {
		t.Fatalf("expected rate 32, got %s", rate.Rate)
	}
}
