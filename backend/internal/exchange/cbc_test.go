package exchange

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"subflow/internal/domain"
)

func TestCBCQuoteUsesLatestPriorDateAndCrossRate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"dataSets":[["20260807","30","150","0.8"],["20260808","32","160","0.75"]],"structure":{"Table1":[{"data":"date"},{"data":"NTD/USD"},{"data":"JPY/USD"},{"data":"USD/EUR"}]}}}`))
	}))
	defer server.Close()
	provider := &CBCProvider{Client: server.Client(), URL: server.URL}
	requested := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)
	quote, err := provider.Quote(context.Background(), "EUR", "TWD", requested)
	if err != nil {
		t.Fatal(err)
	}
	if quote.Rate != "24" {
		t.Fatalf("expected EUR/TWD cross rate 24, got %s", quote.Rate)
	}
	if quote.EffectiveDate.Format("2006-01-02") != "2026-08-08" || !quote.Stale {
		t.Fatalf("unexpected fallback metadata: %#v", quote)
	}
}

func TestCBCUnsupportedCurrencyRequiresManualRate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"dataSets":[["20260808","32"]],"structure":{"Table1":[{"data":"date"},{"data":"NTD/USD"}]}}}`))
	}))
	defer server.Close()
	provider := &CBCProvider{Client: server.Client(), URL: server.URL}
	_, err := provider.Quote(context.Background(), "USD", "ZAR", time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC))
	if err != domain.ErrRateUnavailable {
		t.Fatalf("expected unavailable, got %v", err)
	}
}
