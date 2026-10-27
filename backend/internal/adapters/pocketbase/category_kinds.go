package pocketbase

import (
	"ganigeorgiev.com/pocketbase/pocketbase/core"
)

var incomeCategoryKeys = []string{"income_salary", "income_bonus", "income_freelance", "income_investment", "income_rental", "income_gifts", "income_other"}

// Migrate references and kinds together. A completed row is never classified
// again: later deletions must not change a category's accounting meaning.
func ensureIncomeCategoryKinds(app core.App) error {
	return app.RunInTransaction(func(tx core.App) error {
		categories, err := tx.FindRecordsByFilter(CollectionCategories, "", "", 0, 0)
		if err != nil {
			return err
		}
		incomes, err := tx.FindRecordsByFilter(CollectionIncomes, "", "", 0, 0)
		if err != nil {
			return err
		}
		incomeUse, expenseUse := map[string]bool{}, map[string]bool{}
		for _, r := range incomes {
			incomeUse[r.GetString("category_ref")] = true
		}
		for _, name := range []string{CollectionExpenses, CollectionSubscriptions, CollectionSubscriptionRevisions} {
			rows, loadErr := tx.FindRecordsByFilter(name, "", "", 0, 0)
			if loadErr != nil {
				return loadErr
			}
			for _, r := range rows {
				expenseUse[r.GetString("category_ref")] = true
			}
		}
		mapping := map[string]string{}
		for _, c := range categories {
			if mapped := c.GetString("income_category_id"); mapped != "" {
				mapping[c.Id] = mapped
			}
			if c.GetString("kind") != "" {
				continue
			}
			kind := "expense"
			if c.GetString("scope") != "system" && incomeUse[c.Id] && !expenseUse[c.Id] {
				kind = "income"
			}
			c.Set("kind", kind)
			if kind == "expense" && incomeUse[c.Id] {
				clone, createErr := newSchemaRecord(tx, CollectionCategories)
				if createErr != nil {
					return createErr
				}
				for _, field := range []string{"scope", "owner", "group", "system_key", "custom_name", "icon_key", "created_by", "archived"} {
					clone.Set(field, c.Get(field))
				}
				clone.Set("kind", "income")
				if err = tx.Save(clone); err != nil {
					return err
				}
				mapping[c.Id] = clone.Id
				c.Set("income_category_id", clone.Id)
			}
			if err = tx.Save(c); err != nil {
				return err
			}
		}
		for _, r := range incomes {
			if id := mapping[r.GetString("category_ref")]; id != "" {
				r.Set("category_ref", id)
				if err = tx.Save(r); err != nil {
					return err
				}
			}
		}
		for _, key := range incomeCategoryKeys {
			rows, findErr := tx.FindRecordsByFilter(CollectionCategories, "kind='income' && system_key={:key}", "", 1, 0, map[string]any{"key": key})
			if findErr != nil {
				return findErr
			}
			if len(rows) != 0 {
				continue
			}
			c, createErr := newSchemaRecord(tx, CollectionCategories)
			if createErr != nil {
				return createErr
			}
			c.Set("scope", "system")
			c.Set("kind", "income")
			c.Set("system_key", key)
			c.Set("icon_key", "tag")
			if err = tx.Save(c); err != nil {
				return err
			}
		}
		return nil
	})
}
