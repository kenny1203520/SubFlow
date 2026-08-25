package application

import (
	"testing"
	"time"

	"subflow/internal/domain"
)

func TestValidShareRequiresAUsableSourceAndContent(t *testing.T) {
	base := domain.Share{Name: "Trip", OwnerID: "user", AccessMode: "link", RangeMode: "all", ShowExpenses: true}
	if !validShare(base) {
		t.Fatal("expected a personal expense share to be valid")
	}
	base.OwnerID, base.GroupID = "user", "group"
	if validShare(base) {
		t.Fatal("a share must have exactly one source")
	}
	base.OwnerID, base.GroupID, base.ShowExpenses, base.ShowSettlements = "user", "", false, true
	if validShare(base) {
		t.Fatal("a personal share cannot expose only group settlements")
	}
}

func TestValidShareValidatesConfiguredRanges(t *testing.T) {
	value := domain.Share{Name: "Quarter", GroupID: "group", AccessMode: "accounts", RangeMode: "rolling", RollingDays: 31, ShowSummary: true}
	if validShare(value) {
		t.Fatal("unexpected rolling period accepted")
	}
	value.RollingDays = 90
	if !validShare(value) {
		t.Fatal("expected configured rolling period to be valid")
	}
	value.RangeMode = "fixed"
	value.StartsOn = time.Now()
	value.EndsOn = value.StartsOn.AddDate(0, 0, -1)
	if validShare(value) {
		t.Fatal("inverted fixed range accepted")
	}
}
func TestShareDateConfiguredTreatsLegacyZeroAsUnset(t *testing.T) {
	if shareDateConfigured(time.Time{}) {
		t.Fatal("zero time should be treated as no expiry")
	}
	if shareDateConfigured(time.Date(0, time.January, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("legacy year zero should be treated as no expiry")
	}
	if !shareDateConfigured(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("configured date should be recognized")
	}
}

func TestSubscriptionInShareRangeAllowsOpenEndedSubscriptions(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)
	item := domain.Subscription{StartsOn: time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)}
	if !subscriptionInShareRange(item, start, end) {
		t.Fatal("open-ended subscription should be included")
	}

	ended := end.AddDate(0, 0, -1)
	item.EndsOn = &ended
	if !subscriptionInShareRange(item, start, end) {
		t.Fatal("subscription ending inside range should be included")
	}
}
