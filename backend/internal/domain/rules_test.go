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
