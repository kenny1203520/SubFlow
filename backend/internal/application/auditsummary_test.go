package application

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestChangeSetAddStringOnlyRecordsRealChanges(t *testing.T) {
	var c changeSet
	c.addString("name", "same", "same")
	if len(c) != 0 {
		t.Fatalf("expected no-op value to record nothing, got %#v", c)
	}
	c.addString("name", "before", "after")
	if len(c) != 1 || c[0] != (auditChange{Field: "name", Before: "before", After: "after"}) {
		t.Fatalf("expected one recorded change, got %#v", c)
	}
}

func TestChangeSetAddInt64(t *testing.T) {
	var c changeSet
	c.addInt64("amount", 100, 100)
	c.addInt64("amount", 100, 200)
	if len(c) != 1 || c[0] != (auditChange{Field: "amount", Before: int64(100), After: int64(200)}) {
		t.Fatalf("expected one recorded change, got %#v", c)
	}
}

func TestChangeSetAddInt(t *testing.T) {
	var c changeSet
	c.addInt("interval", 1, 1)
	c.addInt("interval", 1, 3)
	if len(c) != 1 || c[0] != (auditChange{Field: "interval", Before: 1, After: 3}) {
		t.Fatalf("expected one recorded change, got %#v", c)
	}
}

func TestChangeSetAddBool(t *testing.T) {
	var c changeSet
	c.addBool("archived", false, false)
	c.addBool("archived", false, true)
	if len(c) != 1 || c[0] != (auditChange{Field: "archived", Before: false, After: true}) {
		t.Fatalf("expected one recorded change, got %#v", c)
	}
}

func TestChangeSetAddAnyComparesByEncodedForm(t *testing.T) {
	var c changeSet
	type split struct {
		UserID string
		Amount int64
	}
	same := []split{{UserID: "u1", Amount: 100}}
	c.addAny("splits", same, []split{{UserID: "u1", Amount: 100}})
	if len(c) != 0 {
		t.Fatalf("expected structurally-equal values to record nothing, got %#v", c)
	}
	c.addAny("splits", same, []split{{UserID: "u1", Amount: 200}})
	if len(c) != 1 {
		t.Fatalf("expected one recorded change, got %#v", c)
	}
}

func TestChangeSetAddStrings(t *testing.T) {
	var c changeSet
	c.addStrings("permissions", []string{"a", "b"}, []string{"a", "b"})
	if len(c) != 0 {
		t.Fatalf("expected identical slices to record nothing, got %#v", c)
	}
	c.addStrings("permissions", []string{"a", "b"}, []string{"a", "c"})
	if len(c) != 1 {
		t.Fatalf("expected an element-wise difference to record a change, got %#v", c)
	}
	c.addStrings("permissions", []string{"a"}, []string{"a", "b"})
	if len(c) != 2 {
		t.Fatalf("expected a length difference to record a change, got %#v", c)
	}
}

func TestEncodeAuditSummaryEmptyReturnsEmptyString(t *testing.T) {
	if got := encodeAuditSummary(nil, nil); got != "" {
		t.Fatalf("expected empty details+changes to encode to \"\", got %q", got)
	}
	if got := encodeAuditSummary(map[string]any{}, changeSet{}); got != "" {
		t.Fatalf("expected empty (non-nil) details+changes to encode to \"\", got %q", got)
	}
}

func TestEncodeAuditSummaryRoundTripsDetailsAndChanges(t *testing.T) {
	var changes changeSet
	changes.addString("name", "old", "new")
	encoded := encodeAuditSummary(map[string]any{"amount_minor": int64(500)}, changes)
	if encoded == "" {
		t.Fatal("expected a non-empty encoded summary")
	}
	var decoded auditSummary
	if err := json.Unmarshal([]byte(encoded), &decoded); err != nil {
		t.Fatalf("expected valid JSON, got error: %v", err)
	}
	if got, want := decoded.Details["amount_minor"], float64(500); got != want {
		t.Fatalf("details.amount_minor = %#v, want %#v (JSON numbers decode as float64)", got, want)
	}
	want := []auditChange{{Field: "name", Before: "old", After: "new"}}
	if !reflect.DeepEqual(decoded.Changes, want) {
		t.Fatalf("changes = %#v, want %#v", decoded.Changes, want)
	}
}

func TestEncodeAuditSummaryTruncatesOversizedPayload(t *testing.T) {
	huge := make(map[string]any, 1)
	// Comfortably larger than auditSummaryMaxLen once JSON-encoded.
	padding := make([]byte, auditSummaryMaxLen*2)
	for i := range padding {
		padding[i] = 'x'
	}
	huge["padding"] = string(padding)
	encoded := encodeAuditSummary(huge, nil)
	if len(encoded) != auditSummaryMaxLen {
		t.Fatalf("expected the encoded summary to be truncated to %d bytes, got %d", auditSummaryMaxLen, len(encoded))
	}
}
