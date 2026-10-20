package application

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"subflow/internal/domain"
	"subflow/internal/security"
)

func TestShareInputDecodesEnabledFalse(t *testing.T) {
	var input ShareInput
	if err := json.Unmarshal([]byte(`{"name":"Disabled","enabled":false}`), &input); err != nil {
		t.Fatal(err)
	}
	if input.EnabledInput == nil || *input.EnabledInput || input.Share.Enabled {
		t.Fatalf("explicit disabled value was not preserved: enabled=%v input=%v", input.Share.Enabled, input.EnabledInput)
	}
}

func TestShareInputPreservesOptionalAttachmentSettingPresence(t *testing.T) {
	var omitted, enabled ShareInput
	if err := json.Unmarshal([]byte(`{"name":"Legacy edit"}`), &omitted); err != nil {
		t.Fatal(err)
	}
	if omitted.ShowAttachmentsInput != nil || !omitted.decodedJSON || omitted.ShowAttachments {
		t.Fatalf("legacy request should leave attachments disabled and absent: %#v", omitted)
	}
	if err := json.Unmarshal([]byte(`{"name":"Attachment edit","showAttachments":true}`), &enabled); err != nil {
		t.Fatal(err)
	}
	if enabled.ShowAttachmentsInput == nil || !*enabled.ShowAttachmentsInput || !enabled.ShowAttachments {
		t.Fatalf("explicit attachment opt-in was not preserved: %#v", enabled)
	}
}

func TestShareAttachmentReferencesAreEncryptedAndBoundToShare(t *testing.T) {
	service := &Service{Cipher: security.NewSettingsCipher("share-attachment-test-key")}
	share := &domain.Share{ID: "share-private-id"}
	ref, err := service.makeShareAttachmentReference(share, shareAttachmentLocator{Type: "income", TargetID: "income-private-id"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(ref, share.ID) || strings.Contains(ref, "income-private-id") {
		t.Fatalf("reference leaked an internal id: %q", ref)
	}
	locator, err := service.parseShareAttachmentReference(ref)
	if err != nil {
		t.Fatal(err)
	}
	if locator.ShareID != share.ID || locator.Type != "income" || locator.TargetID != "income-private-id" {
		t.Fatalf("unexpected decoded locator: %#v", locator)
	}
	if ref[0] == 'A' {
		ref = "B" + ref[1:]
	} else {
		ref = "A" + ref[1:]
	}
	if _, err = service.parseShareAttachmentReference(ref); err == nil {
		t.Fatal("tampered locator should be rejected")
	}
}

func TestShareInputDistinguishesOmittedEnabledFromExplicitFalse(t *testing.T) {
	var omitted, disabled ShareInput
	if err := json.Unmarshal([]byte(`{"name":"Keep state"}`), &omitted); err != nil {
		t.Fatal(err)
	}
	if omitted.EnabledInput != nil {
		t.Fatal("omitted enabled should remain distinguishable")
	}
	if err := json.Unmarshal([]byte(`{"name":"Disable","enabled":false}`), &disabled); err != nil {
		t.Fatal(err)
	}
	if disabled.EnabledInput == nil || *disabled.EnabledInput {
		t.Fatal("explicit false was not preserved")
	}
}

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

func TestValidateShareRequiresFixedDatesAndAllowsSameDay(t *testing.T) {
	value := domain.Share{Name: "Fixed", OwnerID: "owner", AccessMode: "link", RangeMode: "fixed", ShowExpenses: true}
	if fields := validateShare(value, "", true); fields["startsOn"] != "required" || fields["endsOn"] != "required" {
		t.Fatalf("missing dates should report both fields, got %#v", fields)
	}
	day := time.Date(2026, time.August, 31, 16, 0, 0, 0, time.UTC)
	value.StartsOn, value.EndsOn = day, day
	if fields := validateShare(value, "", true); len(fields) != 0 {
		t.Fatalf("same-day fixed range should be valid, got %#v", fields)
	}
}

func TestValidateSharePasswordRulesOnCreateAndUpdate(t *testing.T) {
	value := domain.Share{Name: "Locked", OwnerID: "owner", AccessMode: "password", RangeMode: "all", ShowExpenses: true}
	if fields := validateShare(value, "short", true); fields["password"] != "min_length" {
		t.Fatalf("short create password should be rejected, got %#v", fields)
	}
	if fields := validateShare(value, "", false); len(fields) != 0 {
		t.Fatalf("blank update password should preserve the existing password, got %#v", fields)
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

func TestPaginateShareRecordsUsesOneMergedPage(t *testing.T) {
	records := make([]shareRecord, 0, 27)
	for i := 0; i < 25; i++ {
		records = append(records, shareRecord{kind: "expenses", row: map[string]any{"type": "expense"}})
	}
	records = append(records,
		shareRecord{kind: "subscriptions", row: map[string]any{"type": "subscription"}},
		shareRecord{kind: "settlements", row: map[string]any{"type": "settlement"}},
	)

	page, items, current, total, next := paginateShareRecords(records, "all", 1, 5)
	if len(page) != 5 || items != 27 || current != 1 || total != 6 || next != 2 {
		t.Fatalf("page 1 should contain exactly 5 merged records, got len=%d current=%d total=%d next=%d", len(page), current, total, next)
	}
	page, items, current, total, next = paginateShareRecords(records, "all", 99, 5)
	if len(page) != 2 || items != 27 || current != 6 || total != 6 || next != 0 {
		t.Fatalf("out-of-range page should clamp to last page, got len=%d current=%d total=%d next=%d", len(page), current, total, next)
	}
	page, items, current, total, next = paginateShareRecords(records, "settlements", 1, 5)
	if len(page) != 1 || items != 1 || current != 1 || total != 1 || next != 0 {
		t.Fatalf("section filter should paginate only selected records, got len=%d current=%d total=%d next=%d", len(page), current, total, next)
	}
	page, items, current, total, next = paginateShareRecords(nil, "all", 4, 5)
	if len(page) != 0 || items != 0 || current != 1 || total != 1 || next != 0 {
		t.Fatalf("empty feed should expose a stable first page, got len=%d current=%d total=%d next=%d", len(page), current, total, next)
	}
}
