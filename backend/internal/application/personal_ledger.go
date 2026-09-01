package application

import (
	"context"
	"sort"
	"strings"
	"subflow/internal/domain"
	"subflow/internal/ports"
	"time"
)

func incomeAuditSummary(v *domain.Income) string {
	return encodeAuditSummary(map[string]any{"title": v.Title, "amount_minor": v.AmountMinor, "currency": string(v.Currency), "received_on": v.ReceivedOn.Format("2006-01-02")}, nil)
}

func (s *Service) CreateIncome(ctx context.Context, userID string, v domain.Income) (*domain.Income, error) {
	u, err := s.Stores.Users.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	v.GroupID = ""
	v.OwnerID = userID
	v.EarnedBy = userID
	v.Title = strings.TrimSpace(v.Title)
	if v.Currency == "" {
		v.Currency = u.DefaultCurrency
	}
	if v.Currency == "" {
		v.Currency = domain.CurrencyTWD
	}
	if v.BaseCurrency == "" {
		v.BaseCurrency = u.DefaultCurrency
	}
	if v.BaseCurrency == "" {
		v.BaseCurrency = v.Currency
	}
	if v.RateMode == "" {
		v.RateMode = domain.RateAutomatic
	}
	if v.ReceivedOn.IsZero() {
		v.ReceivedOn = s.Now().UTC()
	}
	if v.CategoryID != "" {
		c, e := s.validateCategory(ctx, userID, "", v.CategoryID)
		if e != nil {
			_ = s.audit(ctx, userID, "", "income.created", "income", "", "failure", encodeAuditSummary(map[string]any{"reason": "category_invalid"}, nil))
			return nil, e
		}
		if v.Category == "" {
			v.Category = c.CustomName
		}
	}
	if v.Title == "" || v.AmountMinor < 0 || !domain.IsCurrency(v.Currency) || !domain.IsCurrency(v.BaseCurrency) {
		_ = s.audit(ctx, userID, "", "income.created", "income", "", "failure", encodeAuditSummary(map[string]any{"reason": "invalid_input"}, nil))
		return nil, domain.ErrInvalid
	}
	b, r, rt, rd, e := s.conversion(ctx, v.Currency, v.BaseCurrency, v.AmountMinor, v.RateMode, v.ExchangeRate, v.ReceivedOn)
	if e != nil {
		_ = s.audit(ctx, userID, "", "income.created", "income", "", "failure", encodeAuditSummary(map[string]any{"reason": "conversion_failed"}, nil))
		return nil, e
	}
	v.BaseAmountMinor, v.RateScaled, v.ExchangeRate, v.ExchangeRateDate = b, r, rt, rd
	v.SplitMode = domain.SplitAmount
	v.Splits = []*domain.IncomeSplit{{BaseSplit: domain.BaseSplit{UserID: userID, AmountMinor: v.AmountMinor, BaseAmountMinor: v.BaseAmountMinor}}}
	if e = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if createErr := s.Stores.Incomes.Create(tx, &v); createErr != nil {
			return createErr
		}
		return s.Stores.Incomes.ReplaceSplits(tx, v.ID, v.Splits)
	}); e != nil {
		_ = s.audit(ctx, userID, "", "income.created", "income", "", "failure", encodeAuditSummary(map[string]any{"reason": "write_failed"}, nil))
		return nil, e
	}
	if e = s.audit(ctx, userID, "", "income.created", "income", v.ID, "success", incomeAuditSummary(&v)); e != nil {
		return nil, e
	}
	return &v, nil
}
func (s *Service) ListPersonalIncomes(ctx context.Context, userID string, p ports.PageRequest) (ports.Page[domain.Income], error) {
	r, e := s.Stores.Incomes.ListPersonal(ctx, userID, p)
	if e != nil {
		return r, e
	}
	for i := range r.Items {
		r.Items[i].CategoryInfo = s.hydrateCategory(ctx, r.Items[i].CategoryID)
	}
	return r, nil
}
func (s *Service) UpdateIncome(ctx context.Context, userID string, v domain.Income) (*domain.Income, error) {
	c, e := s.Stores.Incomes.Get(ctx, v.ID)
	if e != nil {
		return nil, e
	}
	if c.OwnerID != userID {
		_ = s.audit(ctx, userID, "", "income.updated", "income", v.ID, "failure", encodeAuditSummary(map[string]any{"reason": "forbidden"}, nil))
		return nil, domain.ErrForbidden
	}
	if strings.TrimSpace(v.Title) == "" {
		v.Title = c.Title
	} else {
		v.Title = strings.TrimSpace(v.Title)
	}
	if v.AmountMinor == 0 && c.AmountMinor != 0 {
		v.AmountMinor = c.AmountMinor
	}
	if v.Currency == "" {
		v.Currency = c.Currency
	}
	if v.BaseCurrency == "" {
		v.BaseCurrency = c.BaseCurrency
	}
	if v.CategoryID == "" {
		v.CategoryID = c.CategoryID
	}
	if v.Category == "" {
		v.Category = c.Category
	}
	if v.ReceivedOn.IsZero() {
		v.ReceivedOn = c.ReceivedOn
	}
	if v.RateMode == "" {
		v.RateMode = c.RateMode
	}
	if v.RateMode == "" {
		v.RateMode = domain.RateAutomatic
	}
	if v.Notes == "" {
		v.Notes = c.Notes
	}
	if _, e = s.validateCategory(ctx, userID, "", v.CategoryID); e != nil {
		_ = s.audit(ctx, userID, "", "income.updated", "income", v.ID, "failure", encodeAuditSummary(map[string]any{"reason": "category_invalid"}, nil))
		return nil, e
	}
	if v.Title == "" || v.AmountMinor < 0 || !domain.IsCurrency(v.Currency) || !domain.IsCurrency(v.BaseCurrency) {
		_ = s.audit(ctx, userID, "", "income.updated", "income", v.ID, "failure", encodeAuditSummary(map[string]any{"reason": "invalid_input"}, nil))
		return nil, domain.ErrInvalid
	}
	b, r, rt, rd, e := s.conversion(ctx, v.Currency, v.BaseCurrency, v.AmountMinor, v.RateMode, v.ExchangeRate, v.ReceivedOn)
	if e != nil {
		return nil, e
	}
	v.GroupID = ""
	v.OwnerID = userID
	v.EarnedBy = userID
	v.BaseAmountMinor, v.RateScaled, v.ExchangeRate, v.ExchangeRateDate = b, r, rt, rd
	v.SplitMode = domain.SplitAmount
	v.Splits = []*domain.IncomeSplit{{BaseSplit: domain.BaseSplit{UserID: userID, AmountMinor: v.AmountMinor, BaseAmountMinor: v.BaseAmountMinor}}}
	if e = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if updateErr := s.Stores.Incomes.Update(tx, &v); updateErr != nil {
			return updateErr
		}
		return s.Stores.Incomes.ReplaceSplits(tx, v.ID, v.Splits)
	}); e != nil {
		return nil, e
	}
	if e = s.audit(ctx, userID, "", "income.updated", "income", v.ID, "success", incomeAuditSummary(&v)); e != nil {
		return nil, e
	}
	return &v, nil
}
func (s *Service) DeleteIncome(ctx context.Context, userID, id string) error {
	v, e := s.Stores.Incomes.Get(ctx, id)
	if e != nil {
		return e
	}
	if v.OwnerID != userID {
		_ = s.audit(ctx, userID, "", "income.deleted", "income", id, "failure", encodeAuditSummary(map[string]any{"reason": "forbidden"}, nil))
		return domain.ErrForbidden
	}
	if e = s.Stores.Incomes.Delete(ctx, id); e != nil {
		return e
	}
	return s.audit(ctx, userID, "", "income.deleted", "income", id, "success", incomeAuditSummary(v))
}
func (s *Service) personalLedgerBounds(ctx context.Context, userID, requested string) (string, string, time.Time, time.Time, error) {
	u, e := s.Stores.Users.Get(ctx, userID)
	if e != nil {
		return "", "", time.Time{}, time.Time{}, e
	}
	tz := strings.TrimSpace(u.Timezone)
	loc := time.UTC
	if tz != "" {
		if l, x := time.LoadLocation(tz); x == nil {
			loc = l
		} else {
			tz = "UTC"
		}
	} else {
		tz = "UTC"
	}
	d := strings.TrimSpace(requested)
	if d == "" {
		d = s.Now().In(loc).Format("2006-01-02")
	}
	x, e := time.ParseInLocation("2006-01-02", d, loc)
	if e != nil {
		return "", "", time.Time{}, time.Time{}, domain.ErrInvalid
	}
	from := time.Date(x.Year(), x.Month(), x.Day(), 0, 0, 0, 0, loc)
	return d, tz, from, from.AddDate(0, 0, 1), nil
}
func (s *Service) PersonalLedger(ctx context.Context, userID, requested string, includeGroups bool) (domain.DailyLedger, error) {
	d, tz, from, to, e := s.personalLedgerBounds(ctx, userID, requested)
	if e != nil {
		return domain.DailyLedger{}, e
	}
	ex, e := s.Stores.Expenses.ListPersonalBetween(ctx, userID, from.UTC(), to.UTC())
	if e != nil {
		return domain.DailyLedger{}, e
	}
	ins, e := s.Stores.Incomes.ListPersonalBetween(ctx, userID, from.UTC(), to.UTC())
	if e != nil {
		return domain.DailyLedger{}, e
	}
	if includeGroups {
		groups, groupErr := listAllGroups(ctx, s, userID)
		if groupErr != nil {
			return domain.DailyLedger{}, groupErr
		}
		for _, group := range groups {
			if permissionErr := s.groupPermission(ctx, userID, group.ID, "ledger.expenses.read"); permissionErr == nil {
				values, listErr := s.Stores.Expenses.ListBetween(ctx, group.ID, from.UTC(), to.UTC())
				if listErr != nil {
					return domain.DailyLedger{}, listErr
				}
				ex = append(ex, values...)
			} else if permissionErr != domain.ErrForbidden {
				return domain.DailyLedger{}, permissionErr
			}
			if permissionErr := s.groupPermission(ctx, userID, group.ID, "ledger.incomes.read"); permissionErr == nil {
				values, listErr := s.Stores.Incomes.ListBetween(ctx, group.ID, from.UTC(), to.UTC())
				if listErr != nil {
					return domain.DailyLedger{}, listErr
				}
				ins = append(ins, values...)
			} else if permissionErr != domain.ErrForbidden {
				return domain.DailyLedger{}, permissionErr
			}
		}
	}
	items := make([]domain.LedgerItem, 0, len(ex)+len(ins))
	subIDs := map[string]bool{}
	for i := range ex {
		x := ex[i]
		if e = s.hydrateExpense(ctx, &x); e != nil {
			return domain.DailyLedger{}, e
		}
		k := "expense"
		if x.SubscriptionID != "" {
			k = "subscription"
			subIDs[x.SubscriptionID] = true
		}
		items = append(items, domain.LedgerItem{ID: x.ID, Kind: k, RecordID: x.ID, SubscriptionID: x.SubscriptionID, GroupID: x.GroupID, OccurredAt: x.IncurredOn, Title: x.Title, Category: x.Category, CategoryID: x.CategoryID, AmountMinor: x.AmountMinor, Currency: x.Currency, BaseCurrency: x.BaseCurrency, BaseAmountMinor: x.BaseAmountMinor, Notes: x.Notes, Status: "recorded"})
	}
	for i := range ins {
		x := ins[i]
		items = append(items, domain.LedgerItem{ID: x.ID, Kind: "income", RecordID: x.ID, GroupID: x.GroupID, OccurredAt: x.ReceivedOn, Title: x.Title, Category: x.Category, CategoryID: x.CategoryID, AmountMinor: x.AmountMinor, Currency: x.Currency, BaseCurrency: x.BaseCurrency, BaseAmountMinor: x.BaseAmountMinor, Notes: x.Notes, Status: "recorded"})
	}
	subs, e := listAllPersonalSubscriptions(ctx, s, userID)
	if e != nil {
		return domain.DailyLedger{}, e
	}
	if includeGroups {
		groups, groupErr := listAllGroups(ctx, s, userID)
		if groupErr != nil {
			return domain.DailyLedger{}, groupErr
		}
		for _, group := range groups {
			if permissionErr := s.groupPermission(ctx, userID, group.ID, "ledger.subscriptions.read"); permissionErr == nil {
				values, listErr := listAllSubscriptions(ctx, s, userID, group.ID)
				if listErr != nil {
					return domain.DailyLedger{}, listErr
				}
				subs = append(subs, values...)
			} else if permissionErr != domain.ErrForbidden {
				return domain.DailyLedger{}, permissionErr
			}
		}
	}
	for i := range subs {
		sub := subs[i]
		failed, scheduled := false, false
		for _, o := range sub.Occurrences {
			if o.BillingAt.In(from.Location()).Format("2006-01-02") != d {
				continue
			}
			if o.ExpenseID != "" && subIDs[sub.ID] {
				continue
			}
			if o.Status == "failed" {
				failed = true
			}
			if o.Status == "pending" {
				scheduled = true
			}
		}
		if sub.NextBilling.In(from.Location()).Format("2006-01-02") == d && sub.Status == domain.SubscriptionActive && !subIDs[sub.ID] {
			scheduled = true
		}
		if failed || scheduled {
			st := "scheduled"
			if failed {
				st = "failed"
			}
			items = append(items, domain.LedgerItem{ID: sub.ID + ":" + d, Kind: "subscription", SubscriptionID: sub.ID, GroupID: sub.GroupID, OccurredAt: sub.NextBilling, Title: sub.Name, Category: sub.Category, CategoryID: sub.CategoryID, AmountMinor: sub.AmountMinor, Currency: sub.Currency, BaseCurrency: sub.BaseCurrency, BaseAmountMinor: sub.BaseAmountMinor, Notes: sub.Notes, Status: st})
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].OccurredAt.Before(items[j].OccurredAt) })
	type agg struct {
		in, ex, sub int64
		count       int
	}
	m := map[domain.Currency]*agg{}
	for _, x := range items {
		a := m[x.Currency]
		if a == nil {
			a = &agg{}
			m[x.Currency] = a
		}
		a.count++
		if x.Kind == "income" {
			a.in += x.AmountMinor
		} else {
			a.ex += x.AmountMinor
			if x.Kind == "subscription" {
				a.sub += x.AmountMinor
			}
		}
	}
	cs := make([]domain.Currency, 0, len(m))
	for c := range m {
		cs = append(cs, c)
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i] < cs[j] })
	ss := make([]domain.DailyLedgerCurrencySummary, 0, len(cs))
	for _, c := range cs {
		a := m[c]
		ss = append(ss, domain.DailyLedgerCurrencySummary{Currency: c, IncomeMinor: a.in, ExpenseMinor: a.ex, SubscriptionMinor: a.sub, NetMinor: a.in - a.ex, Count: a.count})
	}
	return domain.DailyLedger{Date: d, Timezone: tz, Summaries: ss, Items: items}, nil
}
