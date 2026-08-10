package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/currency"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrForbidden       = errors.New("forbidden")
	ErrConflict        = errors.New("conflict")
	ErrInvalid         = errors.New("invalid input")
	ErrRateUnavailable = errors.New("exchange rate unavailable")
	ErrSetupDisabled   = errors.New("setup disabled")
	ErrSetupToken      = errors.New("setup token invalid")
)

func NormalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func IsCurrency(value Currency) bool {
	return activeCurrencyCodes[value]
}

func ActiveCurrencies() []CurrencyInfo {
	result := make([]CurrencyInfo, 0, len(activeCurrencyCodes))
	for code := range activeCurrencyCodes {
		digits, _ := CurrencyDigits(code)
		result = append(result, CurrencyInfo{Code: code, Digits: digits})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	return result
}

var activeCurrencyCodes = func() map[Currency]bool {
	values := strings.Fields("AED AFN ALL AMD AOA ARS AUD AWG AZN BAM BBD BDT BGN BHD BIF BMD BND BOB BRL BSD BTN BWP BYN BZD CAD CDF CHF CLP CNY COP CRC CUP CVE CZK DJF DKK DOP DZD EGP ERN ETB EUR FJD FKP GBP GEL GHS GIP GMD GNF GTQ GYD HKD HNL HTG HUF IDR ILS INR IQD IRR ISK JMD JOD JPY KES KGS KHR KMF KPW KRW KWD KYD KZT LAK LBP LKR LRD LSL LYD MAD MDL MGA MKD MMK MNT MOP MRU MUR MVR MWK MXN MYR MZN NAD NGN NIO NOK NPR NZD OMR PAB PEN PGK PHP PKR PLN PYG QAR RON RSD RUB RWF SAR SBD SCR SDG SEK SGD SHP SLE SOS SRD SSP STN SVC SYP SZL THB TJS TMT TND TOP TRY TTD TWD TZS UAH UGX USD UYU UZS VES VND VUV WST XAF XCD XCG XOF XPF YER ZAR ZMW ZWG")
	result := make(map[Currency]bool, len(values))
	for _, value := range values {
		result[Currency(value)] = true
	}
	return result
}()

func CurrencyDigits(value Currency) (int, error) {
	if !IsCurrency(value) {
		return 0, ErrInvalid
	}
	if strings.Contains(" BIF CLP DJF GNF ISK JPY KMF KRW PYG RWF UGX VND VUV XAF XOF XPF ", " "+string(value)+" ") {
		return 0, nil
	}
	if strings.Contains(" BHD IQD JOD KWD LYD OMR TND ", " "+string(value)+" ") {
		return 3, nil
	}
	unit, err := currency.ParseISO(string(value))
	if err != nil {
		return 2, nil
	}
	scale, _ := currency.Standard.Rounding(unit)
	return scale, nil
}

func ParseRate(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, ErrInvalid
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, ErrInvalid
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole < 0 {
		return 0, ErrInvalid
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > 8 {
		return 0, ErrInvalid
	}
	for len(fraction) < 8 {
		fraction += "0"
	}
	fractionValue := int64(0)
	if fraction != "" {
		fractionValue, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, ErrInvalid
		}
	}
	if whole > (int64(^uint64(0)>>1)-fractionValue)/ExchangeRateScale {
		return 0, ErrInvalid
	}
	result := whole*ExchangeRateScale + fractionValue
	if result <= 0 {
		return 0, ErrInvalid
	}
	return result, nil
}

func FormatRate(value int64) string {
	if value <= 0 {
		return ""
	}
	whole := value / ExchangeRateScale
	fraction := strconv.FormatInt(value%ExchangeRateScale, 10)
	fraction = strings.Repeat("0", 8-len(fraction)) + fraction
	fraction = strings.TrimRight(fraction, "0")
	if fraction == "" {
		return strconv.FormatInt(whole, 10)
	}
	return strconv.FormatInt(whole, 10) + "." + fraction
}

func ConvertMinor(amount int64, from, to Currency, rateScaled int64) (int64, error) {
	if amount < 0 || rateScaled <= 0 || !IsCurrency(from) || !IsCurrency(to) {
		return 0, ErrInvalid
	}
	fromDigits, err := CurrencyDigits(from)
	if err != nil {
		return 0, err
	}
	toDigits, err := CurrencyDigits(to)
	if err != nil {
		return 0, err
	}
	numerator := new(big.Int).Mul(big.NewInt(amount), big.NewInt(rateScaled))
	numerator.Mul(numerator, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(toDigits)), nil))
	denominator := new(big.Int).Mul(big.NewInt(ExchangeRateScale), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(fromDigits)), nil))
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(numerator, denominator, remainder)
	if new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if !quotient.IsInt64() {
		return 0, ErrInvalid
	}
	return quotient.Int64(), nil
}

func CanonicalBaseSplits(total int64, payer string, values []ExpenseSplit) []ExpenseSplit {
	result := append([]ExpenseSplit(nil), values...)
	var allocated int64
	target := 0
	for i := range result {
		allocated += result[i].BaseAmountMinor
		if result[i].UserID == payer {
			target = i
		}
	}
	if len(result) > 0 {
		result[target].BaseAmountMinor += total - allocated
	}
	return result
}

func CanManageGroup(role MemberRole) bool { return role == RoleOwner }

func CanManageRecords(role MemberRole) bool { return role == RoleOwner || role == RoleMember }

func MonthlyEquivalent(amount int64, cycle BillingCycle) (int64, error) {
	return MonthlyEquivalentWithInterval(amount, cycle, 1)
}

// MonthlyEquivalentWithInterval estimates the monthly cost for the selected
// billing cadence. Interval is only meaningful for every-N cadences.
func MonthlyEquivalentWithInterval(amount int64, cycle BillingCycle, interval int) (int64, error) {
	if amount < 0 {
		return 0, ErrInvalid
	}
	interval = normalizeBillingInterval(cycle, interval)
	if interval < 1 {
		return 0, ErrInvalid
	}
	switch cycle {
	case BillingDaily:
		return monthlyRatio(amount, 365, 12)
	case BillingEveryNDays:
		return monthlyRatio(amount, 365, 12*int64(interval))
	case BillingWeekly:
		return monthlyRatio(amount, 52, 12)
	case BillingEveryNWeeks:
		return monthlyRatio(amount, 52, 12*int64(interval))
	case BillingEveryNHours:
		return monthlyRatio(amount, 365*24, 12*int64(interval))
	case BillingMonthly:
		return amount, nil
	case BillingQuarterly:
		return amount / 3, nil
	case BillingYearly:
		return amount / 12, nil
	default:
		return 0, ErrInvalid
	}
}

func monthlyRatio(amount, numerator, denominator int64) (int64, error) {
	if denominator < 1 || numerator < 1 || amount > math.MaxInt64/numerator {
		return 0, ErrInvalid
	}
	return (amount*numerator + denominator/2) / denominator, nil
}

func ValidBillingCycle(cycle BillingCycle, interval int) bool {
	if normalizeBillingInterval(cycle, interval) < 1 {
		return false
	}
	switch cycle {
	case BillingDaily, BillingEveryNDays, BillingWeekly, BillingEveryNWeeks, BillingEveryNHours, BillingMonthly, BillingQuarterly, BillingYearly:
		return true
	default:
		return false
	}
}

func NormalizeBillingInterval(cycle BillingCycle, interval int) int {
	return normalizeBillingInterval(cycle, interval)
}

func normalizeBillingInterval(cycle BillingCycle, interval int) int {
	switch cycle {
	case BillingEveryNDays, BillingEveryNWeeks, BillingEveryNHours:
		if interval < 1 || interval > 8760 {
			return 0
		}
		return interval
	case BillingDaily, BillingWeekly, BillingMonthly, BillingQuarterly, BillingYearly:
		return 1
	default:
		return 0
	}
}

func NextBilling(start time.Time, cycle BillingCycle, now time.Time) (time.Time, error) {
	return NextBillingWithInterval(start, cycle, 1, now)
}

func NextBillingWithInterval(start time.Time, cycle BillingCycle, interval int, now time.Time) (time.Time, error) {
	if start.IsZero() {
		return time.Time{}, ErrInvalid
	}
	for index := 0; index < 12000; index++ {
		value, err := BillingDateWithInterval(start, cycle, interval, index)
		if err != nil {
			return time.Time{}, err
		}
		if !value.Before(now) {
			return value, nil
		}
	}
	return time.Time{}, ErrInvalid
}

func BillingDate(start time.Time, cycle BillingCycle, index int) (time.Time, error) {
	return BillingDateWithInterval(start, cycle, 1, index)
}

func BillingDateWithInterval(start time.Time, cycle BillingCycle, interval, index int) (time.Time, error) {
	if start.IsZero() || index < 0 {
		return time.Time{}, ErrInvalid
	}
	interval = normalizeBillingInterval(cycle, interval)
	if interval < 1 {
		return time.Time{}, ErrInvalid
	}
	switch cycle {
	case BillingDaily:
		return start.AddDate(0, 0, index), nil
	case BillingEveryNDays:
		return start.AddDate(0, 0, index*interval), nil
	case BillingWeekly:
		return start.AddDate(0, 0, index*7), nil
	case BillingEveryNWeeks:
		return start.AddDate(0, 0, index*7*interval), nil
	case BillingEveryNHours:
		hours := int64(index) * int64(interval)
		if hours > math.MaxInt64/int64(time.Hour) {
			return time.Time{}, ErrInvalid
		}
		return start.Add(time.Duration(hours) * time.Hour), nil
	}
	months := 0
	switch cycle {
	case BillingMonthly:
		months = index
	case BillingQuarterly:
		months = index * 3
	case BillingYearly:
		months = index * 12
	default:
		return time.Time{}, ErrInvalid
	}
	total := int(start.Month()) - 1 + months
	year := start.Year() + total/12
	month := time.Month(total%12 + 1)
	day := start.Day()
	last := time.Date(year, month+1, 0, 0, 0, 0, 0, start.Location()).Day()
	if day > last {
		day = last
	}
	return time.Date(year, month, day, start.Hour(), start.Minute(), start.Second(), start.Nanosecond(), start.Location()), nil
}

func BillingDates(start time.Time, cycle BillingCycle, from time.Time, limit int) ([]time.Time, error) {
	return BillingDatesWithInterval(start, cycle, 1, from, limit)
}

func BillingDatesWithInterval(start time.Time, cycle BillingCycle, interval int, from time.Time, limit int) ([]time.Time, error) {
	if limit < 1 || limit > 1200 {
		return nil, ErrInvalid
	}
	dates := make([]time.Time, 0, limit)
	for i := 0; i < 12000 && len(dates) < limit; i++ {
		value, err := BillingDateWithInterval(start, cycle, interval, i)
		if err != nil {
			return nil, err
		}
		if !value.Before(from) {
			dates = append(dates, value)
		}
	}
	return dates, nil
}

func CanonicalSplits(amount int64, payer string, mode SplitMode, input []ExpenseSplit, members []string) ([]ExpenseSplit, error) {
	if amount < 0 || len(input) == 0 {
		return nil, ErrInvalid
	}
	allowed := map[string]bool{}
	for _, id := range members {
		allowed[id] = true
	}
	seen := map[string]bool{}
	values := append([]ExpenseSplit(nil), input...)
	for _, v := range values {
		if v.UserID == "" || seen[v.UserID] || !allowed[v.UserID] {
			return nil, ErrInvalid
		}
		seen[v.UserID] = true
	}
	sort.Slice(values, func(i, j int) bool { return values[i].UserID < values[j].UserID })
	remainderTarget := 0
	for i := range values {
		if values[i].UserID == payer {
			remainderTarget = i
			break
		}
	}
	switch mode {
	case SplitEqual:
		q := amount / int64(len(values))
		remainder := amount - q*int64(len(values))
		for i := range values {
			values[i].AmountMinor = q
			values[i].PercentageBasisPoints = 0
		}
		values[remainderTarget].AmountMinor += remainder
	case SplitAmount:
		var total int64
		for _, v := range values {
			if v.AmountMinor < 0 {
				return nil, ErrInvalid
			}
			total += v.AmountMinor
		}
		if total != amount {
			return nil, ErrInvalid
		}
		for i := range values {
			values[i].PercentageBasisPoints = 0
		}
	case SplitPercentage:
		totalBP := 0
		var allocated int64
		for i := range values {
			if values[i].PercentageBasisPoints < 0 {
				return nil, ErrInvalid
			}
			totalBP += values[i].PercentageBasisPoints
			values[i].AmountMinor = amount * int64(values[i].PercentageBasisPoints) / 10000
			allocated += values[i].AmountMinor
		}
		if totalBP != 10000 {
			return nil, ErrInvalid
		}
		values[remainderTarget].AmountMinor += amount - allocated
	default:
		return nil, ErrInvalid
	}
	return values, nil
}

func MemberBalances(expenses []Expense, settlements []Settlement) []MemberBalance {
	totals := map[string]int64{}
	for _, expense := range expenses {
		amount := expense.BaseAmountMinor
		if amount == 0 {
			amount = expense.AmountMinor
		}
		totals[expense.PaidBy] += amount
		for _, split := range expense.Splits {
			share := split.BaseAmountMinor
			if share == 0 {
				share = split.AmountMinor
			}
			totals[split.UserID] -= share
		}
	}
	for _, settlement := range settlements {
		amount := settlement.BaseAmountMinor
		if amount == 0 {
			amount = settlement.AmountMinor
		}
		totals[settlement.FromUserID] += amount
		totals[settlement.ToUserID] -= amount
	}
	ids := make([]string, 0, len(totals))
	for id := range totals {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]MemberBalance, 0, len(ids))
	for _, id := range ids {
		result = append(result, MemberBalance{UserID: id, AmountMinor: totals[id]})
	}
	return result
}

func SubscriptionLifecycle(v Subscription, now time.Time) string {
	if v.Status == SubscriptionCancelled {
		return "cancelled"
	}
	if v.Status == SubscriptionPaused {
		return "paused"
	}
	if v.EndsOn != nil {
		if now.After(*v.EndsOn) {
			return "ended"
		}
		return "ending"
	}
	return "active"
}

func NewInvitationToken() (plain string, hash string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(plain))
	hash = hex.EncodeToString(sum[:])
	return plain, hash, nil
}

func HashInvitationToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func InvitationUsable(invitation Invitation, now time.Time) bool {
	return invitation.Status == InvitationPending && now.Before(invitation.ExpiresAt)
}
