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

func (p *OpenERAPIProvider) Quote(ctx context.Context, from, to domain.Currency, requested time.Time) (*domain.ExchangeRate, error) {
	if from == to {
		return &domain.ExchangeRate{BaseCurrency: from, QuoteCurrency: to, RateScaled: domain.ExchangeRateScale, Rate: "1", EffectiveDate: requested, Provider: "identity", FetchedAt: time.Now()}, nil
	}
	base := p.URL
	if base == "" {
		base = OpenERAPIURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/"+string(from), nil)
	if err != nil {
		return nil, err
	}
	res, err := p.Client.Do(req)
	if err != nil {
		return nil, domain.ErrRateUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, domain.ErrRateUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var payload openERAPIResponse
	if json.Unmarshal(body, &payload) != nil || payload.Result != "success" {
		return nil, domain.ErrRateUnavailable
	}
	raw, ok := payload.Rates[string(to)]
	if !ok {
		return nil, domain.ErrRateUnavailable
	}
	rate, ok := new(big.Rat).SetString(raw.String())
	if !ok || rate.Sign() <= 0 {
		return nil, domain.ErrRateUnavailable
	}
	scaled := new(big.Rat).Mul(rate, big.NewRat(domain.ExchangeRateScale, 1))
	quotient := new(big.Int).Quo(scaled.Num(), scaled.Denom())
	remainder := new(big.Int).Rem(scaled.Num(), scaled.Denom())
	if new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(scaled.Denom()) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if !quotient.IsInt64() || quotient.Sign() <= 0 {
		return nil, domain.ErrRateUnavailable
	}
	value := quotient.Int64()
	effective := requested
	if parsed, parseErr := time.Parse(time.RFC1123Z, payload.TimeLastUpdateUTC); parseErr == nil {
		effective = parsed
	}
	stale := effective.Format("2006-01-02") != requested.Format("2006-01-02")
	return &domain.ExchangeRate{BaseCurrency: from, QuoteCurrency: to, RateScaled: value, Rate: domain.FormatRate(value), EffectiveDate: effective, Provider: "open-er-api", FetchedAt: time.Now(), Stale: stale}, nil
}
