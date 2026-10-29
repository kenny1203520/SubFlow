package domain

import "testing"

func TestSettlementIsEffectiveKeepsLegacyAndConfirmedRowsOnly(t *testing.T) {
	for status, want := range map[string]bool{"": true, "confirmed": true, "pending": false, "rejected": false, "withdrawn": false, "voided": false} {
		if got := SettlementIsEffective(Settlement{Status: status}); got != want {
			t.Errorf("SettlementIsEffective(status=%q) = %v, want %v", status, got, want)
		}
	}
}
