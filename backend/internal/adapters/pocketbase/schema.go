package pocketbase

import "github.com/pocketbase/pocketbase/core"

const (
	CollectionGroups        = "groups"
	CollectionMembers       = "group_members"
	CollectionInvitations   = "group_invitations"
	CollectionSubscriptions = "subscriptions"
	CollectionExpenses      = "expenses"
	CollectionExpenseSplits = "expense_splits"
	CollectionSettlements   = "settlements"
)

// EnsureSchema installs the SubFlow baseline idempotently. Domain collections
// have no public rules because business access is restricted to /api/subflow/v1.
func EnsureSchema(app core.App) error {
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	if users.Fields.GetByName("timezone") == nil {
		users.Fields.Add(&core.TextField{Name: "timezone", Max: 64})
		if err = app.Save(users); err != nil {
			return err
		}
	}
	groups, err := ensureCollection(app, CollectionGroups, func(c *core.Collection) {
		c.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 120}, &core.TextField{Name: "description", Max: 2000}, &core.SelectField{Name: "currency", Required: true, Values: []string{"TWD", "USD", "JPY", "EUR"}, MaxSelect: 1}, &core.TextField{Name: "color", Max: 32}, &core.RelationField{Name: "owner", Required: true, CollectionId: users.Id, MaxSelect: 1, CascadeDelete: true})
		c.AddIndex("idx_groups_owner", false, "owner", "")
	})
	if err != nil {
		return err
	}
	_, err = ensureCollection(app, CollectionMembers, func(c *core.Collection) {
		c.Fields.Add(&core.RelationField{Name: "group", Required: true, CollectionId: groups.Id, MaxSelect: 1, CascadeDelete: true}, &core.RelationField{Name: "user", Required: true, CollectionId: users.Id, MaxSelect: 1, CascadeDelete: true}, &core.SelectField{Name: "role", Required: true, Values: []string{"owner", "member"}, MaxSelect: 1})
		c.AddIndex("idx_group_members_unique", true, "`group`, `user`", "")
	})
	if err != nil {
		return err
	}
	_, err = ensureCollection(app, CollectionInvitations, func(c *core.Collection) {
		c.Fields.Add(&core.RelationField{Name: "group", Required: true, CollectionId: groups.Id, MaxSelect: 1, CascadeDelete: true}, &core.EmailField{Name: "email", Required: true}, &core.TextField{Name: "token_hash", Required: true, Hidden: true, Max: 128}, &core.DateField{Name: "expires_at", Required: true}, &core.RelationField{Name: "invited_by", Required: true, CollectionId: users.Id, MaxSelect: 1}, &core.RelationField{Name: "accepted_by", CollectionId: users.Id, MaxSelect: 1}, &core.SelectField{Name: "status", Required: true, Values: []string{"pending", "delivery_failed", "accepted", "revoked", "expired"}, MaxSelect: 1})
		c.AddIndex("idx_invitations_token", true, "token_hash", "")
		c.AddIndex("idx_invitations_group_email", false, "`group`, email", "")
	})
	if err != nil {
		return err
	}
	_, err = ensureCollection(app, CollectionSubscriptions, func(c *core.Collection) {
		c.Fields.Add(&core.RelationField{Name: "group", CollectionId: groups.Id, MaxSelect: 1, CascadeDelete: true}, &core.RelationField{Name: "owner", CollectionId: users.Id, MaxSelect: 1, CascadeDelete: true}, &core.RelationField{Name: "paid_by", CollectionId: users.Id, MaxSelect: 1}, &core.TextField{Name: "name", Required: true, Max: 160}, &core.TextField{Name: "category", Max: 80}, &core.NumberField{Name: "amount_minor", Required: true, OnlyInt: true}, &core.SelectField{Name: "currency", Required: true, Values: []string{"TWD", "USD", "JPY", "EUR"}, MaxSelect: 1}, &core.SelectField{Name: "billing_cycle", Required: true, Values: []string{"monthly", "quarterly", "yearly"}, MaxSelect: 1}, &core.DateField{Name: "starts_on", Required: true}, &core.DateField{Name: "ends_on"}, &core.DateField{Name: "next_billing", Required: true}, &core.SelectField{Name: "status", Required: true, Values: []string{"active", "paused", "cancelled"}, MaxSelect: 1}, &core.TextField{Name: "notes", Max: 4000})
		c.AddIndex("idx_subscriptions_group_next", false, "`group`, next_billing", "")
	})
	if err != nil {
		return err
	}
	_, err = ensureCollection(app, CollectionExpenses, func(c *core.Collection) {
		c.Fields.Add(&core.RelationField{Name: "group", CollectionId: groups.Id, MaxSelect: 1, CascadeDelete: true}, &core.RelationField{Name: "owner", CollectionId: users.Id, MaxSelect: 1, CascadeDelete: true}, &core.TextField{Name: "title", Required: true, Max: 160}, &core.TextField{Name: "category", Max: 80}, &core.NumberField{Name: "amount_minor", Required: true, OnlyInt: true}, &core.RelationField{Name: "paid_by", Required: true, CollectionId: users.Id, MaxSelect: 1}, &core.DateField{Name: "incurred_on", Required: true}, &core.SelectField{Name: "split_mode", Values: []string{"equal", "amount", "percentage"}, MaxSelect: 1}, &core.TextField{Name: "notes", Max: 4000})
		c.AddIndex("idx_expenses_group_date", false, "`group`, incurred_on", "")
	})
	if err != nil { return err }
	_, err = ensureCollection(app, CollectionExpenseSplits, func(c *core.Collection) {
		c.Fields.Add(&core.RelationField{Name: "expense", Required: true, CollectionId: mustCollectionID(app, CollectionExpenses), MaxSelect: 1, CascadeDelete: true}, &core.RelationField{Name: "user", Required: true, CollectionId: users.Id, MaxSelect: 1}, &core.NumberField{Name: "amount_minor", Required: true, OnlyInt: true}, &core.NumberField{Name: "percentage_bp", OnlyInt: true})
		c.AddIndex("idx_splits_expense_user", true, "expense, user", "")
	})
	if err != nil { return err }
	_, err = ensureCollection(app, CollectionSettlements, func(c *core.Collection) {
		c.Fields.Add(&core.RelationField{Name: "group", Required: true, CollectionId: groups.Id, MaxSelect: 1, CascadeDelete: true}, &core.RelationField{Name: "from_user", Required: true, CollectionId: users.Id, MaxSelect: 1}, &core.RelationField{Name: "to_user", Required: true, CollectionId: users.Id, MaxSelect: 1}, &core.RelationField{Name: "created_by", Required: true, CollectionId: users.Id, MaxSelect: 1}, &core.NumberField{Name: "amount_minor", Required: true, OnlyInt: true}, &core.DateField{Name: "settled_on", Required: true}, &core.TextField{Name: "notes", Max: 4000})
		c.AddIndex("idx_settlements_group_date", false, "`group`, settled_on", "")
	})
	return err
}

func mustCollectionID(app core.App, name string) string { c, _ := app.FindCollectionByNameOrId(name); return c.Id }

func ensureCollection(app core.App, name string, define func(*core.Collection)) (*core.Collection, error) {
	if existing, err := app.FindCollectionByNameOrId(name); err == nil {
		changed := addAutodates(existing)
		if changed {
			if err = app.Save(existing); err != nil {
				return nil, err
			}
		}
		return existing, nil
	}
	c := core.NewBaseCollection(name)
	define(c)
	addAutodates(c)
	if err := app.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}
func addAutodates(c *core.Collection) bool {
	changed := false
	if c.Fields.GetByName("created") == nil {
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		changed = true
	}
	if c.Fields.GetByName("updated") == nil {
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		changed = true
	}
	return changed
}
