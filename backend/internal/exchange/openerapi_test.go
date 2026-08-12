package exchange

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"subflow/internal/domain"
)

func TestOpenERAPIQuoteParsesRate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"result":"success","time_last_update_utc":"Sun, 09 Aug 2026 00:00:01 +0000","rates":{"USD":1,"TWD":32.5,"INR":83.12}}`))
	}))
	defer server.Close()
	provider := &OpenERAPIProvider{Client: server.Client(), URL: server.URL}
	requested := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)
	quote, err := provider.Quote(context.Background(), "USD", "TWD", requested)
	if err != nil {
		t.Fatal(err)
	}
	if quote.Rate != "32.5" {
		t.Fatalf("expected rate 32.5, got %s", quote.Rate)
	}
	if quote.Stale {
		t.Fatalf("expected fresh quote for matching date, got stale: %#v", quote)
	}
}

func TestOpenERAPIQuoteCoversMinorCurrency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"result":"success","time_last_update_utc":"Sun, 09 Aug 2026 00:00:01 +0000","rates":{"USD":1,"TWD":32.5,"INR":83.12}}`))
	}))
	defer server.Close()
	provider := &OpenERAPIProvider{Client: server.Client(), URL: server.URL}
	quote, err := provider.Quote(context.Background(), "USD", "INR", time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if quote.Rate != "83.12" {
		t.Fatalf("expected rate 83.12, got %s", quote.Rate)
	}
}

func TestOpenERAPIQuoteMissingCurrencyReturnsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"result":"success","time_last_update_utc":"Sun, 09 Aug 2026 00:00:01 +0000","rates":{"USD":1}}`))
	}))
	defer server.Close()
	provider := &OpenERAPIProvider{Client: server.Client(), URL: server.URL}
	_, err := provider.Quote(context.Background(), "USD", "ZAR", time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC))
	if err != domain.ErrRateUnavailable {
		t.Fatalf("expected unavailable, got %v", err)
	}
}

func TestOpenERAPIQuoteNonOKStatusReturnsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	provider := &OpenERAPIProvider{Client: server.Client(), URL: server.URL}
	_, err := provider.Quote(context.Background(), "USD", "TWD", time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC))
	if err != domain.ErrRateUnavailable {
		t.Fatalf("expected unavailable, got %v", err)
	}
}

func TestOpenERAPIQuoteAllReturnsEveryCurrency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"result":"success","time_last_update_utc":"Sun, 09 Aug 2026 00:00:01 +0000","rates":{"USD":1,"TWD":32.5,"INR":83.12}}`))
	}))
	defer server.Close()
	provider := &OpenERAPIProvider{Client: server.Client(), URL: server.URL}
	rates, err := provider.QuoteAll(context.Background(), "USD", time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(rates) != 2 { // TWD and INR; USD (the base) is excluded as its own quote currency
		t.Fatalf("expected 2 quote currencies (excluding the base), got %d: %#v", len(rates), rates)
	}
	if rates["TWD"] == nil || rates["TWD"].Rate != "32.5" {
		t.Fatalf("expected TWD rate 32.5, got %#v", rates["TWD"])
	}
	if rates["INR"] == nil || rates["INR"].Rate != "83.12" {
		t.Fatalf("expected INR rate 83.12, got %#v", rates["INR"])
	}
}

// roundTripFunc lets a test simulate a transport-level failure (a dropped
// connection, not an HTTP error status) on the first attempt and success on
// the second, to exercise OpenERAPIProvider's single retry.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOpenERAPIQuoteRetriesOnceAfterTransportFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"result":"success","time_last_update_utc":"Sun, 09 Aug 2026 00:00:01 +0000","rates":{"TWD":32.5}}`))
	}))
	defer server.Close()

	attempts := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return nil, errors.New("simulated connection reset")
		}
		return http.DefaultTransport.RoundTrip(r)
	})}
	provider := &OpenERAPIProvider{Client: client, URL: server.URL}
	quote, err := provider.Quote(context.Background(), "USD", "TWD", time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected the retry to recover from the first transport failure, got %v", err)
	}
	if quote.Rate != "32.5" {
		t.Fatalf("expected rate 32.5, got %s", quote.Rate)
	}
	if attempts != 2 {
		t.Fatalf("expected exactly 2 attempts (1 failure + 1 retry), got %d", attempts)
	}
}

func TestOpenERAPIQuoteGivesUpAfterRepeatedTransportFailure(t *testing.T) {
	attempts := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts++
		return nil, errors.New("simulated connection reset")
	})}
	provider := &OpenERAPIProvider{Client: client, URL: "http://127.0.0.1:1"}
	_, err := provider.Quote(context.Background(), "USD", "TWD", time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC))
	if err != domain.ErrRateUnavailable {
		t.Fatalf("expected unavailable after exhausting retries, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected exactly 2 attempts total, got %d", attempts)
	}
}

func TestOpenERAPIQuoteSameCurrencyIsIdentity(t *testing.T) {
	provider := NewOpenERAPIProvider()
	requested := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)
	quote, err := provider.Quote(context.Background(), "TWD", "TWD", requested)
	if err != nil {
		t.Fatal(err)
	}
	if quote.Rate != "1" || quote.Provider != "identity" {
		t.Fatalf("unexpected identity quote: %#v", quote)
	}
}
