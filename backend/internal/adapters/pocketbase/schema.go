package pocketbase

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"subflow/internal/domain"
)

const (
	CollectionGroups        = "groups"
	CollectionMembers       = "group_members"
	CollectionInvitations   = "group_invitations"
	CollectionSubscriptions = "subscriptions"
	CollectionExpenses      = "expenses"
	CollectionExpenseSplits = "expense_splits"
	CollectionSettlements   = "settlements"
	CollectionCategories    = "categories"
	CollectionExchangeRates = "exchange_rates"
)

var systemCategoryKeys = []string{"food_dining", "transport", "housing", "utilities", "shopping", "entertainment", "health", "education", "travel", "insurance", "software_digital", "memberships", "taxes_fees", "gifts_donations", "other"}

func currencyValues() []string {
	values := domain.ActiveCurrencies()
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = string(value.Code)
	}
	return result
}

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
		c.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 120}, &core.TextField{Name: "description", Max: 2000}, &core.SelectField{Name: "currency", Required: true, Values: currencyValues(), MaxSelect: 1}, &core.TextField{Name: "timezone", Max: 64}, &core.TextField{Name: "color", Max: 32}, &core.RelationField{Name: "owner", Required: true, CollectionId: users.Id, MaxSelect: 1, CascadeDelete: true})
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
	categories, err := ensureCollection(app, CollectionCategories, func(c *core.Collection) {
		c.Fields.Add(&core.SelectField{Name: "scope", Required: true, Values: []string{"system", "personal", "group"}, MaxSelect: 1}, &core.RelationField{Name: "owner", CollectionId: users.Id, MaxSelect: 1, CascadeDelete: true}, &core.RelationField{Name: "group", CollectionId: groups.Id, MaxSelect: 1, CascadeDelete: true}, &core.TextField{Name: "system_key", Max: 80}, &core.TextField{Name: "custom_name", Max: 120}, &core.RelationField{Name: "created_by", CollectionId: users.Id, MaxSelect: 1}, &core.BoolField{Name: "archived"})
		c.AddIndex("idx_categories_scope", false, "scope, owner, `group`, archived", "")
	})
	if err != nil {
		return err
	}
	_, err = ensureCollection(app, CollectionExchangeRates, func(c *core.Collection) {
		c.Fields.Add(&core.SelectField{Name: "base_currency", Required: true, Values: currencyValues(), MaxSelect: 1}, &core.SelectField{Name: "quote_currency", Required: true, Values: currencyValues(), MaxSelect: 1}, &core.NumberField{Name: "rate_scaled", Required: true, OnlyInt: true}, &core.DateField{Name: "effective_date", Required: true}, &core.TextField{Name: "provider", Required: true, Max: 80}, &core.DateField{Name: "fetched_at", Required: true})
		c.AddIndex("idx_exchange_rates_unique", true, "base_currency, quote_currency, effective_date", "")
	})
	if err != nil {
		return err
	}
	_, err = ensureCollection(app, CollectionSubscriptions, func(c *core.Collection) {
		c.Fields.Add(&core.RelationField{Name: "group", CollectionId: groups.Id, MaxSelect: 1, CascadeDelete: true}, &core.RelationField{Name: "owner", CollectionId: users.Id, MaxSelect: 1, CascadeDelete: true}, &core.RelationField{Name: "paid_by", CollectionId: users.Id, MaxSelect: 1}, &core.TextField{Name: "name", Required: true, Max: 160}, &core.TextField{Name: "category", Max: 120}, &core.RelationField{Name: "category_ref", CollectionId: categories.Id, MaxSelect: 1}, &core.NumberField{Name: "amount_minor", Required: true, OnlyInt: true}, &core.SelectField{Name: "currency", Required: true, Values: currencyValues(), MaxSelect: 1}, &core.SelectField{Name: "base_currency", Values: currencyValues(), MaxSelect: 1}, &core.NumberField{Name: "base_amount_minor", OnlyInt: true}, &core.NumberField{Name: "exchange_rate_scaled", OnlyInt: true}, &core.DateField{Name: "exchange_rate_date"}, &core.SelectField{Name: "rate_mode", Values: []string{"automatic", "manual"}, MaxSelect: 1}, &core.SelectField{Name: "billing_cycle", Required: true, Values: []string{"monthly", "quarterly", "yearly"}, MaxSelect: 1}, &core.DateField{Name: "starts_on", Required: true}, &core.DateField{Name: "ends_on"}, &core.DateField{Name: "next_billing", Required: true}, &core.SelectField{Name: "status", Required: true, Values: []string{"active", "paused", "cancelled"}, MaxSelect: 1}, &core.TextField{Name: "notes", Max: 4000})
		c.AddIndex("idx_subscriptions_group_next", false, "`group`, next_billing", "")
	})
	if err != nil {
		return err
	}
	_, err = ensureCollection(app, CollectionExpenses, func(c *core.Collection) {
		c.Fields.Add(&core.RelationField{Name: "group", CollectionId: groups.Id, MaxSelect: 1, CascadeDelete: true}, &core.RelationField{Name: "owner", CollectionId: users.Id, MaxSelect: 1, CascadeDelete: true}, &core.TextField{Name: "title", Required: true, Max: 160}, &core.TextField{Name: "category", Max: 120}, &core.RelationField{Name: "category_ref", CollectionId: categories.Id, MaxSelect: 1}, &core.NumberField{Name: "amount_minor", Required: true, OnlyInt: true}, &core.SelectField{Name: "currency", Values: currencyValues(), MaxSelect: 1}, &core.SelectField{Name: "base_currency", Values: currencyValues(), MaxSelect: 1}, &core.NumberField{Name: "base_amount_minor", OnlyInt: true}, &core.NumberField{Name: "exchange_rate_scaled", OnlyInt: true}, &core.DateField{Name: "exchange_rate_date"}, &core.SelectField{Name: "rate_mode", Values: []string{"automatic", "manual"}, MaxSelect: 1}, &core.RelationField{Name: "paid_by", Required: true, CollectionId: users.Id, MaxSelect: 1}, &core.DateField{Name: "incurred_on", Required: true}, &core.SelectField{Name: "split_mode", Values: []string{"equal", "amount", "percentage"}, MaxSelect: 1}, &core.TextField{Name: "notes", Max: 4000})
		c.AddIndex("idx_expenses_group_date", false, "`group`, incurred_on", "")
	})
	if err != nil {
		return err
	}
	_, err = ensureCollection(app, CollectionExpenseSplits, func(c *core.Collection) {
		c.Fields.Add(&core.RelationField{Name: "expense", Required: true, CollectionId: mustCollectionID(app, CollectionExpenses), MaxSelect: 1, CascadeDelete: true}, &core.RelationField{Name: "user", Required: true, CollectionId: users.Id, MaxSelect: 1}, &core.NumberField{Name: "amount_minor", Required: true, OnlyInt: true}, &core.NumberField{Name: "base_amount_minor", OnlyInt: true}, &core.NumberField{Name: "percentage_bp", OnlyInt: true})
		c.AddIndex("idx_splits_expense_user", true, "expense, user", "")
	})
	if err != nil {
		return err
	}
	_, err = ensureCollection(app, CollectionSettlements, func(c *core.Collection) {
		c.Fields.Add(&core.RelationField{Name: "group", Required: true, CollectionId: groups.Id, MaxSelect: 1, CascadeDelete: true}, &core.RelationField{Name: "from_user", Required: true, CollectionId: users.Id, MaxSelect: 1}, &core.RelationField{Name: "to_user", Required: true, CollectionId: users.Id, MaxSelect: 1}, &core.RelationField{Name: "created_by", Required: true, CollectionId: users.Id, MaxSelect: 1}, &core.NumberField{Name: "amount_minor", Required: true, OnlyInt: true}, &core.SelectField{Name: "currency", Values: currencyValues(), MaxSelect: 1}, &core.SelectField{Name: "base_currency", Values: currencyValues(), MaxSelect: 1}, &core.NumberField{Name: "base_amount_minor", OnlyInt: true}, &core.NumberField{Name: "exchange_rate_scaled", OnlyInt: true}, &core.DateField{Name: "exchange_rate_date"}, &core.DateField{Name: "settled_on", Required: true}, &core.TextField{Name: "notes", Max: 4000})
		c.AddIndex("idx_settlements_group_date", false, "`group`, settled_on", "")
	})
	if err != nil {
		return err
	}
	return backfillFinance(app)
}

func mustCollectionID(app core.App, name string) string {
	c, _ := app.FindCollectionByNameOrId(name)
	return c.Id
}

func ensureCollection(app core.App, name string, define func(*core.Collection)) (*core.Collection, error) {
	if existing, err := app.FindCollectionByNameOrId(name); err == nil {
		// Existing installations must receive newly introduced fields too. Build the
		// desired collection separately and only append fields that do not exist.
		desired := core.NewBaseCollection(name)
		define(desired)
		changed := addAutodates(existing)
		for _, field := range desired.Fields {
			current := existing.Fields.GetByName(field.GetName())
			if current == nil {
				existing.Fields.Add(field)
				changed = true
			} else if syncField(current, field) {
				changed = true
			}
		}
		for _, index := range desired.Indexes {
			if !containsIndex(existing.Indexes, index) {
				existing.Indexes = append(existing.Indexes, index)
				changed = true
			}
		}
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
func containsIndex(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
func syncField(current, desired core.Field) bool {
	switch want := desired.(type) {
	case *core.RelationField:
		if value, ok := current.(*core.RelationField); ok && value.Required != want.Required {
			value.Required = want.Required
			return true
		}
	case *core.SelectField:
		if value, ok := current.(*core.SelectField); ok {
			changed := false
			if value.Required != want.Required {
				value.Required = want.Required
				changed = true
			}
			if strings.Join(value.Values, "\x00") != strings.Join(want.Values, "\x00") {
				value.Values = append([]string(nil), want.Values...)
				changed = true
			}
			return changed
		}
	}
	return false
}
func backfillFinance(app core.App) error {
	for _, key := range systemCategoryKeys {
		if _, findErr := app.FindFirstRecordByFilter(CollectionCategories, "system_key={:key}", map[string]any{"key": key}); findErr == nil {
			continue
		}
		record, createErr := newSchemaRecord(app, CollectionCategories)
		if createErr != nil {
			return createErr
		}
		record.Set("scope", "system")
		record.Set("system_key", key)
		if createErr = app.Save(record); createErr != nil {
			return createErr
		}
	}
	groups, err := app.FindRecordsByFilter(CollectionGroups, "", "", 0, 0, nil)
	if err != nil {
		return err
	}
	for _, group := range groups {
		if group.GetString("timezone") == "" {
			timezone := "UTC"
			if owner, findErr := app.FindRecordById("users", group.GetString("owner")); findErr == nil && owner.GetString("timezone") != "" {
				timezone = owner.GetString("timezone")
			}
			group.Set("timezone", timezone)
			if err = app.Save(group); err != nil {
				return err
			}
		}
	}
	expenses, err := app.FindRecordsByFilter(CollectionExpenses, "", "", 0, 0, nil)
	if err != nil {
		return err
	}
	for _, expense := range expenses {
		changed := false
		if expense.GetString("currency") == "" {
			currency := "TWD"
			if groupID := expense.GetString("group"); groupID != "" {
				if group, findErr := app.FindRecordById(CollectionGroups, groupID); findErr == nil {
					currency = group.GetString("currency")
				}
			}
			expense.Set("currency", currency)
			changed = true
		}
		if expense.GetString("split_mode") == "" {
			expense.Set("split_mode", "amount")
			changed = true
		}
		if expense.GetString("base_currency") == "" {
			expense.Set("base_currency", expense.GetString("currency"))
			expense.Set("base_amount_minor", expense.GetFloat("amount_minor"))
			expense.Set("exchange_rate_scaled", domain.ExchangeRateScale)
			expense.Set("exchange_rate_date", expense.GetDateTime("incurred_on").Time())
			expense.Set("rate_mode", "automatic")
			changed = true
		}
		if expense.GetString("category_ref") == "" {
			id, categoryErr := ensureLegacyCategory(app, expense.GetString("owner"), expense.GetString("group"), expense.GetString("category"))
			if categoryErr != nil {
				return categoryErr
			}
			expense.Set("category_ref", id)
			changed = true
		}
		splits, findErr := app.FindRecordsByFilter(CollectionExpenseSplits, "expense={:expense}", "", 1, 0, map[string]any{"expense": expense.Id})
		if findErr != nil {
			return findErr
		}
		if len(splits) == 0 && expense.GetString("paid_by") != "" {
			split, createErr := newSchemaRecord(app, CollectionExpenseSplits)
			if createErr != nil {
				return createErr
			}
			split.Set("expense", expense.Id)
			split.Set("user", expense.GetString("paid_by"))
			split.Set("amount_minor", expense.GetFloat("amount_minor"))
			split.Set("base_amount_minor", expense.GetFloat("base_amount_minor"))
			if createErr = app.Save(split); createErr != nil {
				return createErr
			}
		}
		for _, split := range splits {
			if split.GetFloat("base_amount_minor") == 0 {
				split.Set("base_amount_minor", split.GetFloat("amount_minor"))
				if err = app.Save(split); err != nil {
					return err
				}
			}
		}
		if changed {
			if err = app.Save(expense); err != nil {
				return err
			}
		}
	}
	subscriptions, err := app.FindRecordsByFilter(CollectionSubscriptions, "", "", 0, 0, nil)
	if err != nil {
		return err
	}
	for _, subscription := range subscriptions {
		changed := false
		if subscription.GetString("starts_on") == "" {
			subscription.Set("starts_on", subscription.GetDateTime("next_billing").Time())
			changed = true
		}
		if subscription.GetString("paid_by") == "" {
			payer := subscription.GetString("owner")
			if groupID := subscription.GetString("group"); groupID != "" {
				if group, findErr := app.FindRecordById(CollectionGroups, groupID); findErr == nil {
					payer = group.GetString("owner")
				}
			}
			subscription.Set("paid_by", payer)
			changed = true
		}
		if subscription.GetString("base_currency") == "" {
			subscription.Set("base_currency", subscription.GetString("currency"))
			subscription.Set("base_amount_minor", subscription.GetFloat("amount_minor"))
			subscription.Set("exchange_rate_scaled", domain.ExchangeRateScale)
			subscription.Set("exchange_rate_date", subscription.GetDateTime("starts_on").Time())
			subscription.Set("rate_mode", "automatic")
			changed = true
		}
		if subscription.GetString("category_ref") == "" {
			id, categoryErr := ensureLegacyCategory(app, subscription.GetString("owner"), subscription.GetString("group"), subscription.GetString("category"))
			if categoryErr != nil {
				return categoryErr
			}
			subscription.Set("category_ref", id)
			changed = true
		}
		if changed {
			if err = app.Save(subscription); err != nil {
				return err
			}
		}
	}
	settlements, err := app.FindRecordsByFilter(CollectionSettlements, "", "", 0, 0, nil)
	if err != nil {
		return err
	}
	for _, settlement := range settlements {
		if settlement.GetString("base_currency") != "" {
			continue
		}
		group, findErr := app.FindRecordById(CollectionGroups, settlement.GetString("group"))
		if findErr != nil {
			return findErr
		}
		currency := group.GetString("currency")
		settlement.Set("currency", currency)
		settlement.Set("base_currency", currency)
		settlement.Set("base_amount_minor", settlement.GetFloat("amount_minor"))
		settlement.Set("exchange_rate_scaled", domain.ExchangeRateScale)
		settlement.Set("exchange_rate_date", settlement.GetDateTime("settled_on").Time())
		if err = app.Save(settlement); err != nil {
			return err
		}
	}
	return nil
}

func ensureLegacyCategory(app core.App, ownerID, groupID, name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || trimmed == "未分類" || strings.EqualFold(trimmed, "Uncategorized") {
		record, err := app.FindFirstRecordByFilter(CollectionCategories, "system_key='other'", nil)
		if err != nil {
			return "", err
		}
		return record.Id, nil
	}
	filter := "scope='personal' && owner={:owner} && custom_name={:name}"
	params := map[string]any{"owner": ownerID, "name": trimmed}
	scope := "personal"
	createdBy := ownerID
	if groupID != "" {
		filter = "scope='group' && `group`={:group} && custom_name={:name}"
		params = map[string]any{"group": groupID, "name": trimmed}
		scope = "group"
		if group, err := app.FindRecordById(CollectionGroups, groupID); err == nil {
			createdBy = group.GetString("owner")
		}
	}
	if record, err := app.FindFirstRecordByFilter(CollectionCategories, filter, params); err == nil {
		return record.Id, nil
	}
	record, err := newSchemaRecord(app, CollectionCategories)
	if err != nil {
		return "", err
	}
	record.Set("scope", scope)
	record.Set("owner", ownerID)
	record.Set("group", groupID)
	record.Set("custom_name", trimmed)
	record.Set("created_by", createdBy)
	if err = app.Save(record); err != nil {
		return "", err
	}
	return record.Id, nil
}
func newSchemaRecord(app core.App, name string) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		return nil, err
	}
	return core.NewRecord(collection), nil
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
