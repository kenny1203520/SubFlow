package domain

import (
	"testing"
	"time"
)

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
