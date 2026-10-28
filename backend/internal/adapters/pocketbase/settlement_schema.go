package pocketbase

import (
	"ganigeorgiev.com/pocketbase/pocketbase/core"
)

const CollectionSettlementReviews = "settlement_reviews"

// EnsureSettlementApprovalSchema applies only the schema additions introduced
// by settlement approvals. It is kept separate from the frozen alpha baseline.
func EnsureSettlementApprovalSchema(app core.App) error {
	groups, err := ensureCollection(app, CollectionGroups, func(c *core.Collection) {
		c.Fields.Add(&core.BoolField{Name: "settlement_approval_enabled"})
	})
	if err != nil {
		return err
	}
	settlements, err := ensureCollection(app, CollectionSettlements, func(c *core.Collection) {
		c.Fields.Add(
			&core.SelectField{Name: "status", Values: []string{"confirmed", "pending", "rejected", "voided", "withdrawn"}, MaxSelect: 1},
			&core.BoolField{Name: "approval_required"},
			&core.NumberField{Name: "version", OnlyInt: true},
		)
	})
	if err != nil {
		return err
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	_, err = ensureCollection(app, CollectionSettlementReviews, func(c *core.Collection) {
		c.Fields.Add(
			&core.RelationField{Name: "group", Required: true, CollectionId: groups.Id, MaxSelect: 1, CascadeDelete: true},
			&core.RelationField{Name: "settlement", Required: true, CollectionId: settlements.Id, MaxSelect: 1, CascadeDelete: true},
			&core.SelectField{Name: "action", Required: true, Values: []string{"create", "change", "void"}, MaxSelect: 1},
			&core.JSONField{Name: "proposal", MaxSize: 16384},
			&core.RelationField{Name: "submitted_by", Required: true, CollectionId: users.Id, MaxSelect: 1},
			&core.SelectField{Name: "status", Required: true, Values: []string{"pending", "accepted", "rejected", "withdrawn"}, MaxSelect: 1},
			&core.RelationField{Name: "decided_by", CollectionId: users.Id, MaxSelect: 1},
			&core.TextField{Name: "reason", Max: 1000},
			&core.NumberField{Name: "expected_version", OnlyInt: true},
		)
		c.AddIndex("idx_settlement_reviews_group_status", false, "`group`, status, created", "")
		c.AddIndex("idx_settlement_reviews_settlement_status", false, "settlement, status", "")
		c.AddIndex("idx_settlement_reviews_pending_unique", true, "settlement", "status='pending'")
	})
	return err
}
