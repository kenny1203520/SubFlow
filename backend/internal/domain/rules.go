package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
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
	if start.IsZero() { return time.Time{}, ErrInvalid }
	step := 0
	switch cycle { case BillingMonthly: step = 1; case BillingQuarterly: step = 3; case BillingYearly: step = 12; default: return time.Time{}, ErrInvalid }
	value := start
	for value.Before(now) { value = value.AddDate(0, step, 0) }
	return value, nil
}

func SubscriptionLifecycle(v Subscription, now time.Time) string {
	if v.Status == SubscriptionCancelled { return "cancelled" }
	if v.Status == SubscriptionPaused { return "paused" }
	if v.EndsOn != nil { if now.After(*v.EndsOn) { return "ended" }; return "ending" }
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
