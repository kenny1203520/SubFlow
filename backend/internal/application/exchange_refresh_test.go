package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"ganigeorgiev.com/pocketbase/pocketbase/tests"

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
	errBases       map[domain.Currency]error
	quoteErr       error
}

// failingRateStore wraps the real exchange-rate store and rejects selected
// base/quote pairs, simulating a genuine persistence failure (as opposed to
// an unsupported currency code, which never reaches the store).
type failingRateStore struct {
	ports.ExchangeRateRepository
	failPairs map[[2]domain.Currency]error
}

func (f *failingRateStore) Upsert(ctx context.Context, rate *domain.ExchangeRate) error {
	if err, ok := f.failPairs[[2]domain.Currency{rate.BaseCurrency, rate.QuoteCurrency}]; ok {
		return err
	}
	return f.ExchangeRateRepository.Upsert(ctx, rate)
}

func refreshFailureAudits(t *testing.T, stores adapters.Stores) []domain.AuditLog {
	t.Helper()
	logs, err := stores.Audits.List(context.Background(), "", ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 50}, Action: "exchange_rate.refresh_failed"})
	if err != nil {
		t.Fatal(err)
	}
	return logs.Items
}

func assertCachedRate(t *testing.T, service *application.Service, from, to domain.Currency, date time.Time, want string) {
	t.Helper()
	rate, err := service.QuoteRate(context.Background(), from, to, date)
	if err != nil {
		t.Fatalf("expected %s->%s to be cached, got %v", from, to, err)
	}
	if rate.Stale || rate.Rate != want {
		t.Fatalf("expected fresh %s->%s rate %s, got %#v", from, to, want, rate)
	}
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
	if err, ok := f.errBases[from]; ok {
		return nil, err
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
	return application.New(stores, "test-only-root-key"), stores
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

// The upstream feed lists codes SubFlow does not support (observed in
// production: CLF). The exchange_rates collection only accepts active
// currencies, so such a code used to fail validation and abort the whole
// refresh after a random subset of the first base.
func TestRefreshReferenceRatesSkipsUnsupportedCurrenciesAcrossAllBases(t *testing.T) {
	service, stores := newExchangeTestService(t)
	today := time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return today }
	bases := []domain.Currency{"USD", "EUR", "JPY", "GBP", "TWD"}
	quotes := []domain.Currency{"USD", "EUR", "JPY", "GBP", "TWD", "INR"}
	feed := map[domain.Currency]map[domain.Currency]*domain.ExchangeRate{}
	for _, base := range bases {
		feed[base] = map[domain.Currency]*domain.ExchangeRate{
			"CLF": fakeRate(base, "CLF", 2_00000000, today),
			"XDR": fakeRate(base, "XDR", 3_00000000, today),
			"":    fakeRate(base, "", 4_00000000, today),
			"KRW": nil,
		}
		for _, quote := range quotes {
			if quote != base {
				feed[base][quote] = fakeRate(base, quote, 7_00000000, today)
			}
		}
	}
	service.Rates = &fakeRateProvider{quoteAllByBase: feed}

	if err := service.RefreshReferenceRates(context.Background()); err != nil {
		t.Fatalf("unsupported feed currencies must be skipped, not fail the refresh: %v", err)
	}

	service.Rates = nil
	for _, base := range bases {
		for _, quote := range quotes {
			if quote != base {
				assertCachedRate(t, service, base, quote, today, "7")
			}
		}
		if _, err := stores.ExchangeRates.LatestOnOrBefore(context.Background(), base, "CLF", today); err == nil {
			t.Fatalf("unsupported %s->CLF rate must not be stored", base)
		}
	}
	// Skipping a code the product does not support is the expected steady
	// state on every run, not a failed operation: it must not be audited.
	if logs := refreshFailureAudits(t, stores); len(logs) != 0 {
		t.Fatalf("expected no refresh-failure audit for skipped currencies, got %#v", logs)
	}
}

func TestRefreshReferenceRatesReportsStoreErrorAndContinues(t *testing.T) {
	service, stores := newExchangeTestService(t)
	today := time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return today }
	storeErr := errors.New("disk full")
	service.Stores.ExchangeRates = &failingRateStore{
		ExchangeRateRepository: stores.ExchangeRates,
		failPairs:              map[[2]domain.Currency]error{{"EUR", "TWD"}: storeErr},
	}
	service.Rates = &fakeRateProvider{quoteAllByBase: map[domain.Currency]map[domain.Currency]*domain.ExchangeRate{
		"USD": {"TWD": fakeRate("USD", "TWD", 32_00000000, today)},
		"EUR": {
			"JPY": fakeRate("EUR", "JPY", 160_00000000, today),
			"TWD": fakeRate("EUR", "TWD", 35_00000000, today),
			"USD": fakeRate("EUR", "USD", 2_00000000, today),
		},
		"JPY": {"TWD": fakeRate("JPY", "TWD", 21_00000000, today)},
		"GBP": {"TWD": fakeRate("GBP", "TWD", 41_00000000, today)},
		"TWD": {"USD": fakeRate("TWD", "USD", 3_00000000, today)},
	}}

	err := service.RefreshReferenceRates(context.Background())
	if !errors.Is(err, storeErr) {
		t.Fatalf("a store failure must be returned to the caller, got %v", err)
	}

	service.Rates = nil
	// The failing pair must not take down its siblings or later bases.
	assertCachedRate(t, service, "USD", "TWD", today, "32")
	assertCachedRate(t, service, "EUR", "JPY", today, "160")
	assertCachedRate(t, service, "EUR", "USD", today, "2")
	assertCachedRate(t, service, "JPY", "TWD", today, "21")
	assertCachedRate(t, service, "GBP", "TWD", today, "41")
	assertCachedRate(t, service, "TWD", "USD", today, "3")
	if _, quoteErr := service.QuoteRate(context.Background(), "EUR", "TWD", today); quoteErr == nil {
		t.Fatal("the rejected EUR->TWD rate must not be cached")
	}

	logs := refreshFailureAudits(t, stores)
	if len(logs) != 1 {
		t.Fatalf("expected exactly 1 refresh-failure audit entry for EUR, got %#v", logs)
	}
	entry := logs[0]
	if entry.Outcome != "failure" || entry.ResourceID != "EUR" || entry.Resource != "exchange_rate" {
		t.Fatalf("unexpected audit entry: %#v", entry)
	}
	for _, want := range []string{`"reason":"store_error"`, `"failed":1`, `"stored":2`} {
		if !strings.Contains(entry.Summary, want) {
			t.Fatalf("expected audit summary to contain %s, got %s", want, entry.Summary)
		}
	}
	if strings.Contains(entry.Summary, "disk full") {
		t.Fatalf("raw store error text must not leak into the audit summary: %s", entry.Summary)
	}
}

func TestRefreshReferenceRatesReportsProviderErrorAndContinues(t *testing.T) {
	service, stores := newExchangeTestService(t)
	today := time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return today }
	providerErr := errors.New("unexpected EOF")
	service.Rates = &fakeRateProvider{
		errBases: map[domain.Currency]error{"USD": providerErr},
		quoteAllByBase: map[domain.Currency]map[domain.Currency]*domain.ExchangeRate{
			"EUR": {"TWD": fakeRate("EUR", "TWD", 35_00000000, today)},
			"TWD": {"USD": fakeRate("TWD", "USD", 3_00000000, today)},
		},
	}

	err := service.RefreshReferenceRates(context.Background())
	if !errors.Is(err, providerErr) {
		t.Fatalf("an unexpected provider failure must be returned to the caller, got %v", err)
	}

	service.Rates = nil
	assertCachedRate(t, service, "EUR", "TWD", today, "35")
	assertCachedRate(t, service, "TWD", "USD", today, "3")

	logs := refreshFailureAudits(t, stores)
	if len(logs) != 1 || logs[0].Outcome != "failure" || logs[0].ResourceID != "USD" || !strings.Contains(logs[0].Summary, `"reason":"provider_error"`) {
		t.Fatalf("expected one provider_error refresh-failure audit for USD, got %#v", logs)
	}
}

func TestRefreshReferenceRatesStopsWhenContextIsCancelled(t *testing.T) {
	service, stores := newExchangeTestService(t)
	today := time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return today }
	service.Rates = &fakeRateProvider{quoteAllByBase: map[domain.Currency]map[domain.Currency]*domain.ExchangeRate{
		"USD": {"TWD": fakeRate("USD", "TWD", 32_00000000, today)},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := service.RefreshReferenceRates(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected the cancellation to be returned, got %v", err)
	}
	if _, err := stores.ExchangeRates.LatestOnOrBefore(context.Background(), "USD", "TWD", today); err == nil {
		t.Fatal("no rate may be stored once the context is cancelled")
	}
}

// AGENTS.md: an audit write failure may not be silently ignored. The refresh
// is a best-effort cache warm-up, so it still stores what it can, but the
// caller must learn that the failure could not be recorded.
func TestRefreshReferenceRatesReturnsAuditWriteFailure(t *testing.T) {
	service, stores := newExchangeTestService(t)
	today := time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return today }
	service.AuditKey = nil
	service.Rates = &fakeRateProvider{
		failBases: map[domain.Currency]bool{"USD": true},
		quoteAllByBase: map[domain.Currency]map[domain.Currency]*domain.ExchangeRate{
			"EUR": {"TWD": fakeRate("EUR", "TWD", 35_00000000, today)},
		},
	}

	err := service.RefreshReferenceRates(context.Background())
	if !errors.Is(err, domain.ErrAuditUnavailable) {
		t.Fatalf("expected the audit write failure to be returned, got %v", err)
	}
	if logs := refreshFailureAudits(t, stores); len(logs) != 0 {
		t.Fatalf("expected no audit entry without a signing key, got %#v", logs)
	}
	service.Rates = nil
	assertCachedRate(t, service, "EUR", "TWD", today, "35")
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
