package application

import (
	"context"
	"sort"
	"strings"
	"subflow/internal/domain"
	"time"
)

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
