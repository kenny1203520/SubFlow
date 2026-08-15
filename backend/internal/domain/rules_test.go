package domain

import (
	"testing"
	"time"
)

func TestActiveCurrencyValidationAndDigits(t *testing.T) {
	if !IsCurrency("TWD") || !IsCurrency("XOF") || IsCurrency("XAU") || IsCurrency("ADP") {
		t.Fatal("active ISO currency validation mismatch")
	}
	for code, want := range map[Currency]int{"JPY": 0, "USD": 2, "KWD": 3} {
		got, err := CurrencyDigits(code)
		if err != nil || got != want {
			t.Fatalf("%s digits=%d err=%v", code, got, err)
		}
	}
}

func TestConvertMinorUsesFixedPointAndCurrencyDigits(t *testing.T) {
	got, err := ConvertMinor(12345, "USD", "JPY", 15000000000)
	if err != nil || got != 18518 {
		t.Fatalf("expected rounded JPY 18518, got %d err=%v", got, err)
	}
	rate, err := ParseRate("31.25000000")
	if err != nil || FormatRate(rate) != "31.25" {
		t.Fatalf("rate round trip failed: %d %v", rate, err)
	}
}

func TestMonthlyEquivalent(t *testing.T) {
	tests := []struct {
		cycle BillingCycle
		want  int64
	}{
		{BillingMonthly, 1200},
		{BillingQuarterly, 400},
		{BillingYearly, 100},
	}
	for _, tt := range tests {
		got, err := MonthlyEquivalent(1200, tt.cycle)
		if err != nil || got != tt.want {
			t.Fatalf("cycle %s: got %d, %v", tt.cycle, got, err)
		}
	}
}

// Quarterly and yearly used to truncate their division while every other
// cadence rounded, so the monthly equivalent came out a unit light on amounts
// that don't divide evenly.
func TestMonthlyEquivalentRoundsQuarterlyAndYearly(t *testing.T) {
	tests := []struct {
		cycle  BillingCycle
		amount int64
		want   int64
	}{
		{BillingYearly, 11000, 917},   // 916.67 rounds up, used to truncate to 916
		{BillingQuarterly, 1000, 333}, // 333.33 rounds down
		{BillingQuarterly, 1100, 367}, // 366.67 rounds up, used to truncate to 366
	}
	for _, tt := range tests {
		got, err := MonthlyEquivalent(tt.amount, tt.cycle)
		if err != nil || got != tt.want {
			t.Fatalf("cycle %s amount %d: got %d, want %d (err %v)", tt.cycle, tt.amount, got, tt.want, err)
		}
	}
}

func TestFlexibleSubscriptionCycles(t *testing.T) {
	start := time.Date(2026, time.August, 10, 9, 30, 0, 0, time.UTC)
	cases := []struct {
		name            string
		cycle           BillingCycle
		interval, index int
		want            time.Time
	}{
		{"daily", BillingDaily, 1, 2, time.Date(2026, time.August, 12, 9, 30, 0, 0, time.UTC)},
		{"every three days", BillingEveryNDays, 3, 2, time.Date(2026, time.August, 16, 9, 30, 0, 0, time.UTC)},
		{"weekly", BillingWeekly, 1, 2, time.Date(2026, time.August, 24, 9, 30, 0, 0, time.UTC)},
		{"every two weeks", BillingEveryNWeeks, 2, 2, time.Date(2026, time.September, 7, 9, 30, 0, 0, time.UTC)},
		{"every six hours", BillingEveryNHours, 6, 2, time.Date(2026, time.August, 10, 21, 30, 0, 0, time.UTC)},
	}
	for _, tt := range cases {
		got, err := BillingDateWithInterval(start, tt.cycle, tt.interval, tt.index)
		if err != nil || !got.Equal(tt.want) {
			t.Fatalf("%s: got %v, want %v (%v)", tt.name, got, tt.want, err)
		}
	}
}

func TestFlexibleCycleMonthlyEquivalent(t *testing.T) {
	cases := []struct {
		cycle    BillingCycle
		interval int
		want     int64
	}{
		{BillingDaily, 1, 36500},
		{BillingEveryNDays, 2, 18250},
		{BillingWeekly, 1, 5200},
		{BillingEveryNWeeks, 2, 2600},
		{BillingEveryNHours, 24, 36500},
	}
	for _, tt := range cases {
		got, err := MonthlyEquivalentWithInterval(1200, tt.cycle, tt.interval)
		if err != nil || got != tt.want {
			t.Fatalf("%s/%d: got %d, want %d (%v)", tt.cycle, tt.interval, got, tt.want, err)
		}
	}
	if ValidBillingCycle(BillingEveryNDays, 0) || ValidBillingCycle(BillingEveryNHours, 8761) {
		t.Fatal("invalid interval accepted")
	}
}

func TestInvitationToken(t *testing.T) {
	plain, hash, err := NewInvitationToken()
	if err != nil || plain == "" || hash == plain {
		t.Fatalf("unexpected token: %q %q %v", plain, hash, err)
	}
	if HashInvitationToken(plain) != hash {
		t.Fatal("token hash mismatch")
	}
}

func TestInvitationUsable(t *testing.T) {
	now := time.Now()
	if !InvitationUsable(Invitation{Status: InvitationPending, ExpiresAt: now.Add(time.Hour)}, now) {
		t.Fatal("pending invitation should be usable")
	}
	if InvitationUsable(Invitation{Status: InvitationAccepted, ExpiresAt: now.Add(time.Hour)}, now) {
		t.Fatal("accepted invitation must not be usable")
	}
}

func TestPermissions(t *testing.T) {
	if !CanManageGroup(RoleOwner) || CanManageGroup(RoleMember) {
		t.Fatal("group management role mismatch")
	}
	if !CanManageRecords(RoleOwner) || !CanManageRecords(RoleMember) {
		t.Fatal("record management role mismatch")
	}
}

func TestBillingDatePreservesMonthEndAnchor(t *testing.T) {
	start := time.Date(2024, time.January, 31, 0, 0, 0, 0, time.UTC)
	wants := []time.Time{start, time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC), time.Date(2024, time.March, 31, 0, 0, 0, 0, time.UTC)}
	for index, want := range wants {
		got, err := BillingDate(start, BillingMonthly, index)
		if err != nil || !got.Equal(want) {
			t.Fatalf("index %d: got %v, want %v (%v)", index, got, want, err)
		}
	}
	yearly, err := BillingDate(time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC), BillingYearly, 1)
	if err != nil || yearly.Day() != 28 || yearly.Month() != time.February {
		t.Fatalf("leap anchor: %v (%v)", yearly, err)
	}
}

func TestCanonicalSplits(t *testing.T) {
	members := []string{"payer", "b", "c"}
	equal, err := CanonicalSplits(100, "payer", SplitEqual, []ExpenseSplit{{UserID: "payer"}, {UserID: "b"}, {UserID: "c"}}, members)
	if err != nil {
		t.Fatal(err)
	}
	amounts := map[string]int64{}
	for _, split := range equal {
		amounts[split.UserID] = split.AmountMinor
	}
	if amounts["payer"] != 34 || amounts["b"] != 33 || amounts["c"] != 33 {
		t.Fatalf("unexpected equal split: %#v", amounts)
	}
	percentage, err := CanonicalSplits(101, "payer", SplitPercentage, []ExpenseSplit{{UserID: "payer", PercentageBasisPoints: 5000}, {UserID: "b", PercentageBasisPoints: 5000}}, members)
	if err != nil {
		t.Fatal(err)
	}
	amounts = map[string]int64{}
	for _, split := range percentage {
		amounts[split.UserID] = split.AmountMinor
	}
	if amounts["payer"] != 51 || amounts["b"] != 50 {
		t.Fatalf("unexpected percentage split: %#v", amounts)
	}
	if _, err = CanonicalSplits(100, "payer", SplitPercentage, []ExpenseSplit{{UserID: "payer", PercentageBasisPoints: 9000}}, members); err == nil {
		t.Fatal("invalid percentage total should fail")
	}
}

func TestCanonicalBaseSplitsAllocatesRemainderToPayer(t *testing.T) {
	cases := []struct {
		name   string
		total  int64
		payer  string
		values []ExpenseSplit
	}{
		{"even split, no remainder", 300, "b", []ExpenseSplit{{UserID: "a", BaseAmountMinor: 100}, {UserID: "b", BaseAmountMinor: 100}, {UserID: "c", BaseAmountMinor: 100}}},
		{"uneven split, remainder to payer", 100, "b", []ExpenseSplit{{UserID: "a", BaseAmountMinor: 33}, {UserID: "b", BaseAmountMinor: 33}, {UserID: "c", BaseAmountMinor: 33}}},
		{"single payer takes everything", 500, "solo", []ExpenseSplit{{UserID: "solo", BaseAmountMinor: 0}}},
		{"payer not first in slice", 100, "c", []ExpenseSplit{{UserID: "a", BaseAmountMinor: 40}, {UserID: "b", BaseAmountMinor: 30}, {UserID: "c", BaseAmountMinor: 29}}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := CanonicalBaseSplits(tt.total, tt.payer, tt.values)
			var sum int64
			var payerAmount int64
			payerFound := false
			for _, split := range got {
				sum += split.BaseAmountMinor
				if split.UserID == tt.payer {
					payerAmount = split.BaseAmountMinor
					payerFound = true
				}
			}
			if sum != tt.total {
				t.Fatalf("splits sum to %d, want %d: %#v", sum, tt.total, got)
			}
			if !payerFound {
				t.Fatalf("payer %q missing from result: %#v", tt.payer, got)
			}
			var otherAllocated int64
			for _, v := range tt.values {
				if v.UserID != tt.payer {
					otherAllocated += v.BaseAmountMinor
				}
			}
			if want := tt.total - otherAllocated; payerAmount != want {
				t.Fatalf("payer absorbs remainder: got %d, want %d", payerAmount, want)
			}
		})
	}
}

func TestCanonicalBaseSplitsEmptyInput(t *testing.T) {
	if got := CanonicalBaseSplits(100, "payer", nil); len(got) != 0 {
		t.Fatalf("expected empty result for empty input, got %#v", got)
	}
}

func TestParseRateFormatRateRoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"whole number", "31", "31"},
		{"trailing zeros trimmed", "31.25000000", "31.25"},
		{"eight fractional digits", "1.23456780", "1.2345678"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			scaled, err := ParseRate(tt.input)
			if err != nil {
				t.Fatalf("ParseRate(%q) unexpected error: %v", tt.input, err)
			}
			if got := FormatRate(scaled); got != tt.want {
				t.Fatalf("FormatRate(ParseRate(%q))=%q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseRateRejectsMalformedInput(t *testing.T) {
	cases := []string{"", "   ", "abc", "-5", "-5.5", "1.2.3", ".5", "1.123456789", "0", "0.0"}
	for _, input := range cases {
		if _, err := ParseRate(input); err == nil {
			t.Fatalf("ParseRate(%q) expected an error, got none", input)
		}
	}
}

func TestFormatRateNonPositiveReturnsEmpty(t *testing.T) {
	if got := FormatRate(0); got != "" {
		t.Fatalf("FormatRate(0)=%q, want empty", got)
	}
	if got := FormatRate(-1); got != "" {
		t.Fatalf("FormatRate(-1)=%q, want empty", got)
	}
}

func TestNormalizeEmail(t *testing.T) {
	cases := map[string]string{
		"  User@Example.com  ": "user@example.com",
		"already@lower.com":    "already@lower.com",
		"":                     "",
		"\tSpaced@Out.COM\n":   "spaced@out.com",
	}
	for input, want := range cases {
		if got := NormalizeEmail(input); got != want {
			t.Fatalf("NormalizeEmail(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestSubscriptionLifecycleAllStatuses(t *testing.T) {
	now := time.Date(2026, time.August, 15, 0, 0, 0, 0, time.UTC)
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	cases := []struct {
		name string
		sub  Subscription
		want string
	}{
		{"cancelled takes priority", Subscription{Status: SubscriptionCancelled}, "cancelled"},
		{"paused takes priority", Subscription{Status: SubscriptionPaused}, "paused"},
		{"active with no end date", Subscription{Status: SubscriptionActive}, "active"},
		{"ending: end date in the future", Subscription{Status: SubscriptionActive, EndsOn: &future}, "ending"},
		{"ended: end date in the past", Subscription{Status: SubscriptionActive, EndsOn: &past}, "ended"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := SubscriptionLifecycle(tt.sub, now); got != tt.want {
				t.Fatalf("SubscriptionLifecycle()=%q, want %q", got, tt.want)
			}
		})
	}
}

func TestNextBillingReturnsFirstOccurrenceNotBeforeNow(t *testing.T) {
	start := time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC)

	now := time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC)
	got, err := NextBilling(start, BillingMonthly, now)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("NextBilling()=%v, want %v", got, want)
	}

	// now exactly on a billing date must return that same date, not the next one.
	exact := time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC)
	got, err = NextBilling(start, BillingMonthly, exact)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(exact) {
		t.Fatalf("NextBilling() on an exact match=%v, want %v", got, exact)
	}
}

func TestNextBillingRejectsZeroStart(t *testing.T) {
	if _, err := NextBilling(time.Time{}, BillingMonthly, time.Now()); err == nil {
		t.Fatal("expected an error for a zero start time")
	}
}

func TestNextBillingWithIntervalRespectsCustomInterval(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, time.January, 20, 0, 0, 0, 0, time.UTC)

	got, err := NextBillingWithInterval(start, BillingEveryNDays, 7, now)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, time.January, 22, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("NextBillingWithInterval()=%v, want %v", got, want)
	}
}

func TestMemberBalancesIncludeSettlements(t *testing.T) {
	expenses := []Expense{{PaidBy: "a", AmountMinor: 900, Splits: []ExpenseSplit{{UserID: "a", AmountMinor: 300}, {UserID: "b", AmountMinor: 600}}}}
	settlements := []Settlement{{FromUserID: "b", ToUserID: "a", AmountMinor: 200}}
	balances := MemberBalances(expenses, settlements)
	values := map[string]int64{}
	for _, balance := range balances {
		values[balance.UserID] = balance.AmountMinor
	}
	if values["a"] != 400 || values["b"] != -400 {
		t.Fatalf("unexpected balances: %#v", values)
	}
}
