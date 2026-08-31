package application

import (
	"context"
	"sort"
	"strings"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

func listAllGroupIncomes(ctx context.Context, s *Service, userID, groupID string) ([]domain.Income, error) {
	var all []domain.Income
	for page := 1; ; page++ {
		result, err := s.ListGroupIncomes(ctx, userID, groupID, ports.PageRequest{Page: page, PerPage: exportPageSize, Sort: "-received_on"})
		if err != nil {
			return nil, err
		}
		all = append(all, result.Items...)
		if page >= result.TotalPages || len(result.Items) == 0 {
			return all, nil
		}
	}
}

func listAllPersonalIncomes(ctx context.Context, s *Service, userID string) ([]domain.Income, error) {
	var all []domain.Income
	for page := 1; ; page++ {
		result, err := s.ListPersonalIncomes(ctx, userID, ports.PageRequest{Page: page, PerPage: exportPageSize, Sort: "-received_on"})
		if err != nil {
			return nil, err
		}
		all = append(all, result.Items...)
		if page >= result.TotalPages || len(result.Items) == 0 {
			return all, nil
		}
	}
}

func validGroupIncome(v *domain.Income) bool {
	return strings.TrimSpace(v.Title) != "" &&
		v.AmountMinor >= 0 &&
		domain.IsCurrency(v.Currency) &&
		domain.IsCurrency(v.BaseCurrency) &&
		!v.ReceivedOn.IsZero() &&
		v.EarnedBy != ""
}

func (s *Service) groupIncomeFailure(ctx context.Context, userID, groupID, action, id, reason string, cause error) error {
	if auditErr := s.audit(ctx, userID, groupID, action, "income", id, "failure", encodeAuditSummary(map[string]any{"reason": reason}, nil)); auditErr != nil {
		return auditErr
	}
	return cause
}

func (s *Service) prepareGroupIncome(ctx context.Context, userID string, v *domain.Income, current *domain.Income) error {
	if current != nil {
		v.GroupID = current.GroupID
		v.OwnerID = current.OwnerID
		if v.Title == "" {
			v.Title = current.Title
		} else {
			v.Title = strings.TrimSpace(v.Title)
		}
		if v.AmountMinor == 0 && current.AmountMinor != 0 {
			v.AmountMinor = current.AmountMinor
		}
		if v.Currency == "" {
			v.Currency = current.Currency
		}
		if v.CategoryID == "" {
			v.CategoryID = current.CategoryID
		}
		if v.Category == "" {
			v.Category = current.Category
		}
		if v.EarnedBy == "" {
			v.EarnedBy = current.EarnedBy
		}
		if v.ReceivedOn.IsZero() {
			v.ReceivedOn = current.ReceivedOn
		}
		if v.RateMode == "" {
			v.RateMode = current.RateMode
		}
		if v.Notes == "" {
			v.Notes = current.Notes
		}
		if v.SplitMode == "" {
			v.SplitMode = current.SplitMode
		}
		if len(v.Splits) == 0 {
			v.Splits = append([]*domain.IncomeSplit(nil), current.Splits...)
		}
	}
	if v.EarnedBy == "" {
		v.EarnedBy = userID
	}
	if v.RateMode == "" {
		v.RateMode = domain.RateAutomatic
	}
	group, err := s.Stores.Groups.Get(ctx, v.GroupID)
	if err != nil {
		return err
	}
	if v.Currency == "" {
		v.Currency = group.Currency
	}
	v.BaseCurrency = group.Currency
	if _, err = s.Stores.Memberships.GetRole(ctx, v.GroupID, v.EarnedBy); err != nil {
		return domain.ErrInvalid
	}
	members, err := s.memberIDs(ctx, v.GroupID)
	if err != nil {
		return err
	}
	if len(v.Splits) == 0 {
		v.SplitMode = domain.SplitAmount
		v.Splits = []*domain.IncomeSplit{{BaseSplit: domain.BaseSplit{UserID: v.EarnedBy, AmountMinor: v.AmountMinor}}}
	}
	v.Splits, err = domain.CanonicalSplits(v.AmountMinor, v.EarnedBy, v.SplitMode, v.Splits, members)
	if err != nil {
		return err
	}
	if _, err = s.validateCategory(ctx, userID, v.GroupID, v.CategoryID); err != nil {
		return err
	}
	if v.CategoryID != "" && v.Category == "" {
		if category := s.hydrateCategory(ctx, v.CategoryID); category != nil {
			v.Category = category.CustomName
		}
	}
	baseAmount, rate, rateText, rateDate, err := s.conversion(ctx, v.Currency, v.BaseCurrency, v.AmountMinor, v.RateMode, v.ExchangeRate, v.ReceivedOn)
	if err != nil {
		return err
	}
	v.BaseAmountMinor, v.RateScaled, v.ExchangeRate, v.ExchangeRateDate = baseAmount, rate, rateText, rateDate
	for i := range v.Splits {
		v.Splits[i].BaseAmountMinor, err = domain.ConvertMinor(v.Splits[i].AmountMinor, v.Currency, v.BaseCurrency, rate)
		if err != nil {
			return err
		}
	}
	v.Splits = domain.CanonicalBaseSplits(v.BaseAmountMinor, v.EarnedBy, v.Splits)
	if !validGroupIncome(v) {
		return domain.ErrInvalid
	}
	return nil
}

func (s *Service) CreateGroupIncome(ctx context.Context, userID string, v domain.Income) (*domain.Income, error) {
	v.GroupID = strings.TrimSpace(v.GroupID)
	if err := s.groupPermission(ctx, userID, v.GroupID, "ledger.incomes.write"); err != nil {
		return nil, s.groupIncomeFailure(ctx, userID, v.GroupID, "income.created", "", "forbidden", err)
	}
	v.OwnerID = ""
	if err := s.prepareGroupIncome(ctx, userID, &v, nil); err != nil {
		return nil, s.groupIncomeFailure(ctx, userID, v.GroupID, "income.created", "", "invalid_or_write_failed", err)
	}
	if err := s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if createErr := s.Stores.Incomes.Create(tx, &v); createErr != nil {
			return createErr
		}
		return s.Stores.Incomes.ReplaceSplits(tx, v.ID, v.Splits)
	}); err != nil {
		return nil, s.groupIncomeFailure(ctx, userID, v.GroupID, "income.created", "", "write_failed", err)
	}
	if err := s.audit(ctx, userID, v.GroupID, "income.created", "income", v.ID, "success", incomeAuditSummary(&v)); err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *Service) ListGroupIncomes(ctx context.Context, userID, groupID string, p ports.PageRequest) (ports.Page[domain.Income], error) {
	if err := s.groupPermission(ctx, userID, groupID, "ledger.incomes.read"); err != nil {
		return ports.Page[domain.Income]{}, err
	}
	result, err := s.Stores.Incomes.List(ctx, groupID, p)
	if err != nil {
		return result, err
	}
	for i := range result.Items {
		result.Items[i].CategoryInfo = s.hydrateCategory(ctx, result.Items[i].CategoryID)
	}
	return result, nil
}

func (s *Service) UpdateGroupIncome(ctx context.Context, userID string, v domain.Income) (*domain.Income, error) {
	current, err := s.Stores.Incomes.Get(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	if current.GroupID == "" {
		return nil, s.groupIncomeFailure(ctx, userID, "", "income.updated", v.ID, "not_group_income", domain.ErrInvalid)
	}
	if err = s.groupPermission(ctx, userID, current.GroupID, "ledger.incomes.write"); err != nil {
		return nil, s.groupIncomeFailure(ctx, userID, current.GroupID, "income.updated", v.ID, "forbidden", err)
	}
	if err = s.prepareGroupIncome(ctx, userID, &v, current); err != nil {
		return nil, s.groupIncomeFailure(ctx, userID, current.GroupID, "income.updated", v.ID, "invalid_or_write_failed", err)
	}
	v.ID, v.OwnerID = current.ID, ""
	if err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if updateErr := s.Stores.Incomes.Update(tx, &v); updateErr != nil {
			return updateErr
		}
		return s.Stores.Incomes.ReplaceSplits(tx, v.ID, v.Splits)
	}); err != nil {
		return nil, s.groupIncomeFailure(ctx, userID, current.GroupID, "income.updated", v.ID, "write_failed", err)
	}
	if err = s.audit(ctx, userID, current.GroupID, "income.updated", "income", v.ID, "success", incomeAuditSummary(&v)); err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *Service) DeleteGroupIncome(ctx context.Context, userID, id string) error {
	current, err := s.Stores.Incomes.Get(ctx, id)
	if err != nil {
		return err
	}
	if current.GroupID == "" {
		return s.groupIncomeFailure(ctx, userID, "", "income.deleted", id, "not_group_income", domain.ErrInvalid)
	}
	if err = s.groupPermission(ctx, userID, current.GroupID, "ledger.incomes.delete"); err != nil {
		return s.groupIncomeFailure(ctx, userID, current.GroupID, "income.deleted", id, "forbidden", err)
	}
	if err = s.Stores.Incomes.Delete(ctx, id); err != nil {
		return s.groupIncomeFailure(ctx, userID, current.GroupID, "income.deleted", id, "delete_failed", err)
	}
	return s.audit(ctx, userID, current.GroupID, "income.deleted", "income", id, "success", incomeAuditSummary(current))
}

func (s *Service) GroupLedger(ctx context.Context, userID, groupID, requested string) (domain.DailyLedger, error) {
	if err := s.groupPermission(ctx, userID, groupID, "group.view"); err != nil {
		return domain.DailyLedger{}, err
	}
	if err := s.groupPermission(ctx, userID, groupID, "ledger.expenses.read"); err != nil {
		return domain.DailyLedger{}, err
	}
	if err := s.groupPermission(ctx, userID, groupID, "ledger.incomes.read"); err != nil {
		return domain.DailyLedger{}, err
	}
	date, timezone, from, to, err := s.personalLedgerBounds(ctx, userID, requested)
	if err != nil {
		return domain.DailyLedger{}, err
	}
	expenses, err := s.Stores.Expenses.ListBetween(ctx, groupID, from.UTC(), to.UTC())
	if err != nil {
		return domain.DailyLedger{}, err
	}
	incomes, err := s.Stores.Incomes.ListBetween(ctx, groupID, from.UTC(), to.UTC())
	if err != nil {
		return domain.DailyLedger{}, err
	}
	items := make([]domain.LedgerItem, 0, len(expenses)+len(incomes))
	local := from.Location()
	posted := map[string]bool{}
	for i := range expenses {
		expense := expenses[i]
		if err := s.hydrateExpense(ctx, &expense); err != nil {
			return domain.DailyLedger{}, err
		}
		kind := "expense"
		if expense.SubscriptionID != "" {
			kind = "subscription"
			posted[expense.SubscriptionID+":"+expense.IncurredOn.In(local).Format("2006-01-02")] = true
		}
		items = append(items, domain.LedgerItem{ID: expense.ID, Kind: kind, RecordID: expense.ID, SubscriptionID: expense.SubscriptionID, OccurredAt: expense.IncurredOn, Title: expense.Title, Category: expense.Category, CategoryID: expense.CategoryID, AmountMinor: expense.AmountMinor, Currency: expense.Currency, BaseCurrency: expense.BaseCurrency, BaseAmountMinor: expense.BaseAmountMinor, Notes: expense.Notes, Status: "recorded"})
	}
	for i := range incomes {
		income := incomes[i]
		items = append(items, domain.LedgerItem{ID: income.ID, Kind: "income", RecordID: income.ID, OccurredAt: income.ReceivedOn, Title: income.Title, Category: income.Category, CategoryID: income.CategoryID, AmountMinor: income.AmountMinor, Currency: income.Currency, BaseCurrency: income.BaseCurrency, BaseAmountMinor: income.BaseAmountMinor, Notes: income.Notes, Status: "recorded"})
	}
	var subscriptions []domain.Subscription
	for page := 1; ; page++ {
		result, listErr := s.ListSubscriptions(ctx, userID, groupID, ports.PageRequest{Page: page, PerPage: exportPageSize, Sort: "next_billing"})
		if listErr != nil {
			return domain.DailyLedger{}, listErr
		}
		subscriptions = append(subscriptions, result.Items...)
		if page >= result.TotalPages || len(result.Items) == 0 {
			break
		}
	}
	for _, subscription := range subscriptions {
		failed, scheduled := false, false
		for _, occurrence := range subscription.Occurrences {
			if occurrence.BillingAt.In(local).Format("2006-01-02") != date {
				continue
			}
			if occurrence.ExpenseID != "" && posted[subscription.ID+":"+occurrence.BillingAt.In(local).Format("2006-01-02")] {
				continue
			}
			if occurrence.Status == "failed" {
				failed = true
			}
			if occurrence.Status == "pending" {
				scheduled = true
			}
		}
		if subscription.NextBilling.In(local).Format("2006-01-02") == date && subscription.Status == domain.SubscriptionActive && !posted[subscription.ID+":"+date] {
			scheduled = true
		}
		if failed || scheduled {
			status := "scheduled"
			if failed {
				status = "failed"
			}
			items = append(items, domain.LedgerItem{ID: subscription.ID + ":" + date, Kind: "subscription", SubscriptionID: subscription.ID, OccurredAt: subscription.NextBilling, Title: subscription.Name, Category: subscription.Category, CategoryID: subscription.CategoryID, AmountMinor: subscription.AmountMinor, Currency: subscription.Currency, BaseCurrency: subscription.BaseCurrency, BaseAmountMinor: subscription.BaseAmountMinor, Notes: subscription.Notes, Status: status})
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].OccurredAt.Before(items[j].OccurredAt) })
	type aggregate struct {
		income, expense, subscription int64
		count                         int
	}
	totals := map[domain.Currency]*aggregate{}
	for _, item := range items {
		bucket := totals[item.Currency]
		if bucket == nil {
			bucket = &aggregate{}
			totals[item.Currency] = bucket
		}
		bucket.count++
		if item.Kind == "income" {
			bucket.income += item.AmountMinor
		} else {
			bucket.expense += item.AmountMinor
			if item.Kind == "subscription" {
				bucket.subscription += item.AmountMinor
			}
		}
	}
	currencies := make([]domain.Currency, 0, len(totals))
	for currency := range totals {
		currencies = append(currencies, currency)
	}
	sort.Slice(currencies, func(i, j int) bool { return currencies[i] < currencies[j] })
	summaries := make([]domain.DailyLedgerCurrencySummary, 0, len(currencies))
	for _, currency := range currencies {
		bucket := totals[currency]
		summaries = append(summaries, domain.DailyLedgerCurrencySummary{Currency: currency, IncomeMinor: bucket.income, ExpenseMinor: bucket.expense, SubscriptionMinor: bucket.subscription, NetMinor: bucket.income - bucket.expense, Count: bucket.count})
	}
	return domain.DailyLedger{Date: date, Timezone: timezone, Summaries: summaries, Items: items}, nil
}
