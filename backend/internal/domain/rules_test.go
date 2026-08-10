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
