package application

import "encoding/json"

// auditChange describes one field that differed between the before/after
// values of a mutation. The frontend looks up Field in its locale table to
// render a human description ("split_mode: equal -> amount") rather than
// showing raw JSON.
type auditChange struct {
	Field  string `json:"field"`
	Before any    `json:"before,omitempty"`
	After  any    `json:"after,omitempty"`
}

// auditSummary is the structured payload stored in AuditLog.Summary. Details
// carries identifying snapshot fields (name, amount, dates...) for creates,
// deletes, and other one-shot events; Changes lists what differed on an
// update. Either may be empty. Kept resource-agnostic on purpose: the audit
// row's own Action/Resource fields already say what kind of thing this is.
type auditSummary struct {
	Details map[string]any `json:"details,omitempty"`
	Changes []auditChange  `json:"changes,omitempty"`
}

// auditSummaryMaxLen mirrors the audit_logs.summary TextField's max length
// (schema.go). Truncating here means an oversized summary never fails the
// mutation it is describing.
const auditSummaryMaxLen = 4000

// encodeAuditSummary serializes details/changes to JSON for s.audit's summary
// argument. Returns "" for an empty summary so callers can pass it directly
// without an extra guard, and audit() already treats "" as no summary.
func encodeAuditSummary(details map[string]any, changes []auditChange) string {
	if len(details) == 0 && len(changes) == 0 {
		return ""
	}
	blob, err := json.Marshal(auditSummary{Details: details, Changes: changes})
	if err != nil {
		return ""
	}
	text := string(blob)
	if len(text) > auditSummaryMaxLen {
		text = text[:auditSummaryMaxLen]
	}
	return text
}

// changeSet accumulates field diffs for an update. Typed add* methods avoid
// comparing `any` values of possibly-uncomparable underlying types.
type changeSet []auditChange

func (c *changeSet) addString(field, before, after string) {
	if before != after {
		*c = append(*c, auditChange{Field: field, Before: before, After: after})
	}
}
func (c *changeSet) addInt64(field string, before, after int64) {
	if before != after {
		*c = append(*c, auditChange{Field: field, Before: before, After: after})
	}
}
func (c *changeSet) addInt(field string, before, after int) {
	if before != after {
		*c = append(*c, auditChange{Field: field, Before: before, After: after})
	}
}
func (c *changeSet) addBool(field string, before, after bool) {
	if before != after {
		*c = append(*c, auditChange{Field: field, Before: before, After: after})
	}
}

// addAny compares two JSON-marshalable values (e.g. expense splits) by their
// encoded form, for fields whose underlying type isn't directly comparable.
func (c *changeSet) addAny(field string, before, after any) {
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	if string(beforeJSON) != string(afterJSON) {
		*c = append(*c, auditChange{Field: field, Before: before, After: after})
	}
}
func (c *changeSet) addStrings(field string, before, after []string) {
	if len(before) != len(after) {
		*c = append(*c, auditChange{Field: field, Before: before, After: after})
		return
	}
	for i := range before {
		if before[i] != after[i] {
			*c = append(*c, auditChange{Field: field, Before: before, After: after})
			return
		}
	}
}
