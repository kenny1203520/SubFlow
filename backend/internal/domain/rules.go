package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
	ErrConflict  = errors.New("conflict")
	ErrInvalid   = errors.New("invalid input")
)

func NormalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func IsCurrency(value Currency) bool {
	switch value {
	case CurrencyTWD, CurrencyUSD, CurrencyJPY, CurrencyEUR:
		return true
	default:
		return false
	}
}

func CanManageGroup(role MemberRole) bool { return role == RoleOwner }

func CanManageRecords(role MemberRole) bool { return role == RoleOwner || role == RoleMember }

func MonthlyEquivalent(amount int64, cycle BillingCycle) (int64, error) {
	if amount < 0 {
		return 0, ErrInvalid
	}
	switch cycle {
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

func NextBilling(start time.Time, cycle BillingCycle, now time.Time) (time.Time, error) {
	if start.IsZero() {
		return time.Time{}, ErrInvalid
	}
	for index := 0; index < 12000; index++ {
		value, err := BillingDate(start, cycle, index)
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
	if start.IsZero() || index < 0 {
		return time.Time{}, ErrInvalid
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
	if limit < 1 || limit > 100 {
		return nil, ErrInvalid
	}
	dates := make([]time.Time, 0, limit)
	for i := 0; i < 12000 && len(dates) < limit; i++ {
		value, err := BillingDate(start, cycle, i)
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
		totals[expense.PaidBy] += expense.AmountMinor
		for _, split := range expense.Splits {
			totals[split.UserID] -= split.AmountMinor
		}
	}
	for _, settlement := range settlements {
		totals[settlement.FromUserID] += settlement.AmountMinor
		totals[settlement.ToUserID] -= settlement.AmountMinor
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
