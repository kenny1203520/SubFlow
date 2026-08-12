package exchange

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"subflow/internal/domain"
)

// OpenERAPIURL is a free, key-less daily exchange rate feed covering ~160
// currencies (including TWD), used in place of the Central Bank of Taiwan's
// BP01D01en dataset, which stopped publishing new rows in 2012. The base
// currency is appended to the path, e.g. ".../latest/USD".
const OpenERAPIURL = "https://open.er-api.com/v6/latest"

type OpenERAPIProvider struct {
	Client *http.Client
	URL    string
}

func NewOpenERAPIProvider() *OpenERAPIProvider {
	return &OpenERAPIProvider{Client: &http.Client{Timeout: 15 * time.Second}, URL: OpenERAPIURL}
}

type openERAPIResponse struct {
	Result            string                 `json:"result"`
	TimeLastUpdateUTC string                 `json:"time_last_update_utc"`
	Rates             map[string]json.Number `json:"rates"`
}

// fetch performs the single HTTP call this feed needs per base currency,
// retrying once after a short delay on a transport-level failure (a dropped
// connection or timeout) since a lone network blip shouldn't surface all the
// way up as "exchange rate unavailable" to the user.
func (p *OpenERAPIProvider) fetch(ctx context.Context, from domain.Currency) (*openERAPIResponse, error) {
	base := p.URL
	if base == "" {
		base = OpenERAPIURL
	}
	url := strings.TrimRight(base, "/") + "/" + string(from)

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(300 * time.Millisecond):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		res, err := p.Client.Do(req)
		if err != nil {
			lastErr = domain.ErrRateUnavailable
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(res.Body, 4<<20))
		statusCode := res.StatusCode
		res.Body.Close()
		if statusCode != http.StatusOK {
			// A non-200 response (bad currency code, feed-side error) isn't a
			// transport blip a retry would fix, so fail fast.
			return nil, domain.ErrRateUnavailable
		}
		if readErr != nil {
			return nil, readErr
		}
		var payload openERAPIResponse
		if json.Unmarshal(body, &payload) != nil || payload.Result != "success" {
			return nil, domain.ErrRateUnavailable
		}
		return &payload, nil
	}
	return nil, lastErr
}

func rateFromDecimal(raw json.Number) (int64, bool) {
	rate, ok := new(big.Rat).SetString(raw.String())
	if !ok || rate.Sign() <= 0 {
		return 0, false
	}
	scaled := new(big.Rat).Mul(rate, big.NewRat(domain.ExchangeRateScale, 1))
	quotient := new(big.Int).Quo(scaled.Num(), scaled.Denom())
	remainder := new(big.Int).Rem(scaled.Num(), scaled.Denom())
	if new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(scaled.Denom()) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if !quotient.IsInt64() || quotient.Sign() <= 0 {
		return 0, false
	}
	return quotient.Int64(), true
}

func (p *OpenERAPIProvider) Quote(ctx context.Context, from, to domain.Currency, requested time.Time) (*domain.ExchangeRate, error) {
	if from == to {
		return &domain.ExchangeRate{BaseCurrency: from, QuoteCurrency: to, RateScaled: domain.ExchangeRateScale, Rate: "1", EffectiveDate: requested, Provider: "identity", FetchedAt: time.Now()}, nil
	}
	payload, err := p.fetch(ctx, from)
	if err != nil {
		return nil, err
	}
	raw, ok := payload.Rates[string(to)]
	if !ok {
		return nil, domain.ErrRateUnavailable
	}
	value, ok := rateFromDecimal(raw)
	if !ok {
		return nil, domain.ErrRateUnavailable
	}
	effective := requested
	if parsed, parseErr := time.Parse(time.RFC1123Z, payload.TimeLastUpdateUTC); parseErr == nil {
		effective = parsed
	}
	stale := effective.Format("2006-01-02") != requested.Format("2006-01-02")
	return &domain.ExchangeRate{BaseCurrency: from, QuoteCurrency: to, RateScaled: value, Rate: domain.FormatRate(value), EffectiveDate: effective, Provider: "open-er-api", FetchedAt: time.Now(), Stale: stale}, nil
}

// QuoteAll returns every currency this feed quotes against a single base in
// one HTTP call, so callers that want broad cache coverage (see
// Service.RefreshReferenceRates) don't need one round trip per currency pair.
func (p *OpenERAPIProvider) QuoteAll(ctx context.Context, from domain.Currency, requested time.Time) (map[domain.Currency]*domain.ExchangeRate, error) {
	payload, err := p.fetch(ctx, from)
	if err != nil {
		return nil, err
	}
	effective := requested
	if parsed, parseErr := time.Parse(time.RFC1123Z, payload.TimeLastUpdateUTC); parseErr == nil {
		effective = parsed
	}
	stale := effective.Format("2006-01-02") != requested.Format("2006-01-02")
	fetchedAt := time.Now()
	result := make(map[domain.Currency]*domain.ExchangeRate, len(payload.Rates))
	for code, raw := range payload.Rates {
		to := domain.Currency(code)
		if to == from {
			continue
		}
		value, ok := rateFromDecimal(raw)
		if !ok {
			continue
		}
		result[to] = &domain.ExchangeRate{BaseCurrency: from, QuoteCurrency: to, RateScaled: value, Rate: domain.FormatRate(value), EffectiveDate: effective, Provider: "open-er-api", FetchedAt: fetchedAt, Stale: stale}
	}
	return result, nil
}
