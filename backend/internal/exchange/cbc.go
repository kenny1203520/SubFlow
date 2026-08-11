package exchange

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"subflow/internal/domain"
)

const CBCURL = "https://cpx.cbc.gov.tw/API/DataAPI/Get?FileName=BP01D01en"

type CBCProvider struct {
	Client *http.Client
	URL    string
}

func NewCBCProvider() *CBCProvider {
	return &CBCProvider{Client: &http.Client{Timeout: 30 * time.Second}, URL: CBCURL}
}

type response struct {
	Data struct {
		DataSets  [][]string `json:"dataSets"`
		Structure struct {
			Table1 []struct {
				Data string `json:"data"`
			} `json:"Table1"`
		} `json:"structure"`
	} `json:"data"`
}

func (p *CBCProvider) Quote(ctx context.Context, from, to domain.Currency, requested time.Time) (*domain.ExchangeRate, error) {
	if from == to {
		return &domain.ExchangeRate{BaseCurrency: from, QuoteCurrency: to, RateScaled: domain.ExchangeRateScale, Rate: "1", EffectiveDate: requested, Provider: "identity", FetchedAt: time.Now()}, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.URL, nil)
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
	body, err := io.ReadAll(io.LimitReader(res.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	var payload response
	if json.Unmarshal(body, &payload) != nil || len(payload.Data.DataSets) == 0 {
		return nil, domain.ErrRateUnavailable
	}
	labels := make([]string, len(payload.Data.Structure.Table1))
	for i, item := range payload.Data.Structure.Table1 {
		labels[i] = strings.TrimSpace(item.Data)
	}
	var chosen []string
	var effective time.Time
	for _, row := range payload.Data.DataSets {
		if len(row) == 0 {
			continue
		}
		date, dateErr := time.Parse("20060102", strings.TrimSpace(row[0]))
		if dateErr != nil || date.After(requested) {
			continue
		}
		if chosen == nil || date.After(effective) {
			chosen, effective = row, date
		}
	}
	if chosen == nil {
		return nil, domain.ErrRateUnavailable
	}
	usdPer := map[domain.Currency]*big.Rat{"USD": big.NewRat(1, 1)}
	for i := 1; i < len(chosen) && i < len(labels); i++ {
		parts := strings.Split(labels[i], "/")
		if len(parts) != 2 || strings.TrimSpace(chosen[i]) == "" {
			continue
		}
		rate := new(big.Rat)
		if _, ok := rate.SetString(strings.TrimSpace(chosen[i])); !ok || rate.Sign() <= 0 {
			continue
		}
		left, right := domain.Currency(parts[0]), domain.Currency(parts[1])
		if left == "NTD" {
			left = "TWD"
		}
		if right == "NTD" {
			right = "TWD"
		}
		if right == "USD" {
			usdPer[left] = new(big.Rat).Inv(rate)
		}
		if left == "USD" {
			usdPer[right] = rate
		}
	}
	a, aok := usdPer[from]
	b, bok := usdPer[to]
	if !aok || !bok {
		return nil, domain.ErrRateUnavailable
	}
	rate := new(big.Rat).Quo(a, b)
	scaled := new(big.Rat).Mul(rate, big.NewRat(domain.ExchangeRateScale, 1))
	quotient := new(big.Int).Quo(scaled.Num(), scaled.Denom())
	remainder := new(big.Int).Rem(scaled.Num(), scaled.Denom())
	if new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(scaled.Denom()) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if !quotient.IsInt64() || quotient.Sign() <= 0 {
		return nil, errors.New("CBC exchange rate overflow")
	}
	value := quotient.Int64()
	return &domain.ExchangeRate{BaseCurrency: from, QuoteCurrency: to, RateScaled: value, Rate: domain.FormatRate(value), EffectiveDate: effective, Provider: "cbc", FetchedAt: time.Now(), Stale: effective.Format("2006-01-02") != requested.Format("2006-01-02")}, nil
}
