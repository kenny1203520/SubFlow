package exchange

import (
	"context"
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
