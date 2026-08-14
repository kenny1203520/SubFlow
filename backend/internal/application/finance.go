package application

import (
	"context"
	"sort"
	"time"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

type DashboardQuery struct{ Scope, GroupID, Month string }
type BillingDatePage struct {
	Dates      []time.Time `json:"dates"`
	NextCursor string      `json:"nextCursor,omitempty"`
}

func billingDatesBetween(start time.Time, cycle domain.BillingCycle, interval int, from, to time.Time) ([]time.Time, error) {
	if !from.Before(to) {
		return nil, nil
	}
	dates := make([]time.Time, 0)
	for index := 0; index < 12000; index++ {
		value, err := domain.BillingDateWithInterval(start, cycle, interval, index)
		if err != nil {
			return nil, err
		}
		if value.Before(from) {
			continue
		}
		if !value.Before(to) {
			break
		}
		dates = append(dates, value)
	}
	return dates, nil
}

func subscriptionHasPostedOccurrence(subscription domain.Subscription, billingAt time.Time) bool {
	for _, occurrence := range subscription.Occurrences {
		if occurrence.BillingAt.Equal(billingAt) && occurrence.ExpenseID != "" {
			return true
		}
	}
	return false
}

func subscriptionExpenseOccurrence(subscription domain.Subscription, billingAt time.Time) domain.Expense {
	result := domain.Expense{GroupID: subscription.GroupID, OwnerID: subscription.OwnerID, SubscriptionID: subscription.ID, Title: subscription.Name, Category: subscription.Category, CategoryID: subscription.CategoryID, AmountMinor: subscription.AmountMinor, Currency: subscription.Currency, BaseCurrency: subscription.BaseCurrency, BaseAmountMinor: subscription.BaseAmountMinor, ExchangeRate: subscription.ExchangeRate, RateScaled: subscription.RateScaled, ExchangeRateDate: subscription.ExchangeRateDate, RateMode: subscription.RateMode, PaidBy: subscription.PaidBy, IncurredOn: billingAt, Notes: subscription.Notes, SplitMode: subscription.SplitMode, Splits: append([]domain.ExpenseSplit(nil), subscription.Splits...)}
	if revision, ok := subscriptionRevisionAt(subscription.Revisions, billingAt); ok {
		result.Title = revision.Name
		result.Category = revision.Category
		result.CategoryID = revision.CategoryID
		result.AmountMinor = revision.AmountMinor
		result.Currency = revision.Currency
		result.BaseCurrency = revision.BaseCurrency
		result.BaseAmountMinor = revision.BaseAmountMinor
		result.ExchangeRate = revision.ExchangeRate
		result.RateScaled = revision.RateScaled
		result.ExchangeRateDate = revision.ExchangeRateDate
		result.RateMode = revision.RateMode
		result.PaidBy = revision.PaidBy
		result.Notes = revision.Notes
		result.SplitMode = revision.SplitMode
		// A revision without splits must not discard the subscription's own,
		// otherwise the fallback below charges the payer the full amount.
		if len(revision.Splits) > 0 {
			result.Splits = append([]domain.ExpenseSplit(nil), revision.Splits...)
		}
	}
	if len(result.Splits) == 0 {
		result.SplitMode = domain.SplitAmount
		result.Splits = []domain.ExpenseSplit{{UserID: result.PaidBy, AmountMinor: result.AmountMinor, BaseAmountMinor: result.BaseAmountMinor}}
	}
	return result
}

func subscriptionExpensesBetween(subscription domain.Subscription, from, to time.Time, location *time.Location) ([]domain.Expense, error) {
	if location == nil {
		location = time.UTC
	}
	dates, err := billingDatesBetween(subscription.StartsOn.In(location), subscription.BillingCycle, subscription.BillingInterval, from, to)
	if err != nil {
		return nil, err
	}
	values := make([]domain.Expense, 0, len(dates))
	for _, date := range dates {
		if subscription.EndsOn != nil && date.After(*subscription.EndsOn) {
			break
		}
		if subscriptionHasPostedOccurrence(subscription, date) {
			continue
		}
		values = append(values, subscriptionExpenseOccurrence(subscription, date))
	}
	return values, nil
}

// subscriptionRateFor resolves the price actually in effect during the month
// being viewed: the first billing date inside that month when there is one,
// otherwise the rate standing at the month's end (for cadences that don't bill
// every month, such as yearly). It deliberately goes through
// subscriptionExpenseOccurrence, the same resolver the month's cash-flow
// figures use, so the "monthly average" cards can never disagree with the
// cash-flow cards sitting beside them.
func subscriptionRateFor(subscription domain.Subscription, monthDates []time.Time, monthEnd time.Time) domain.Expense {
	reference := monthEnd.Add(-time.Nanosecond)
	if len(monthDates) > 0 {
		reference = monthDates[0]
	}
	return subscriptionExpenseOccurrence(subscription, reference)
}

// subscriptionUserShare returns the portion of a subscription's displayAmount
// that belongs to userID's split, so the dashboard can show the viewer's own
// monthly subscription commitment alongside the group's total. Splits are
// passed in rather than read off the subscription so callers can supply the
// splits of the revision governing the period in question. Personal
// subscriptions (no groupID) belong entirely to their owner.
func subscriptionUserShare(splits []domain.ExpenseSplit, groupID, userID, scope string, displayAmount int64) int64 {
	if groupID == "" {
		return displayAmount
	}
	for _, split := range splits {
		if split.UserID == userID {
			if scope == "group" {
				return split.BaseAmountMinor
			}
			return split.AmountMinor
		}
	}
	return 0
}

func (s *Service) accountingLocation(ctx context.Context, userID, groupID string) *time.Location {
	location := time.UTC
	zone := ""
	if groupID != "" {
		if group, err := s.Stores.Groups.Get(ctx, groupID); err == nil {
			zone = group.Timezone
		}
	}
	if zone == "" {
		if user, err := s.Stores.Users.Get(ctx, userID); err == nil {
			zone = user.Timezone
		}
	}
	if zone != "" {
		if value, loadErr := time.LoadLocation(zone); loadErr == nil {
			location = value
		}
	}
	return location
}

func (s *Service) monthRange(ctx context.Context, userID, groupID, month string) (time.Time, time.Time, error) {
	location := s.accountingLocation(ctx, userID, groupID)
	if month == "" {
		month = s.Now().In(location).Format("2006-01")
	}
	start, err := time.ParseInLocation("2006-01", month, location)
	if err != nil {
		return time.Time{}, time.Time{}, domain.ErrInvalid
	}
	return start, start.AddDate(0, 1, 0), nil
}

func (s *Service) WorkspaceDashboard(ctx context.Context, userID string, query DashboardQuery) (domain.DashboardSummary, error) {
	if query.Scope == "" {
		query.Scope = "personal"
	}
	if query.Scope != "personal" && query.Scope != "group" && query.Scope != "all" {
		return domain.DashboardSummary{}, domain.ErrInvalid
	}
	rangeGroup := ""
	if query.Scope == "group" {
		rangeGroup = query.GroupID
	}
	start, end, err := s.monthRange(ctx, userID, rangeGroup, query.Month)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	result := domain.DashboardSummary{Month: start.Format("2006-01")}
	var expenses []domain.Expense
	var subscriptions []domain.Subscription
	groupTimezone := map[string]string{}
	loadGroup := func(groupID string) error {
		expensePage, loadErr := s.ListExpenses(ctx, userID, groupID, ports.PageRequest{Page: 1, PerPage: 100, Sort: "-incurred_on"})
		if loadErr != nil {
			return loadErr
		}
		subscriptionPage, loadErr := s.ListSubscriptions(ctx, userID, groupID, ports.PageRequest{Page: 1, PerPage: 100, Sort: "next_billing"})
		if loadErr != nil {
			return loadErr
		}
		group, loadErr := s.Stores.Groups.Get(ctx, groupID)
		if loadErr != nil {
			return loadErr
		}
		groupTimezone[groupID] = group.Timezone
		expenses = append(expenses, expensePage.Items...)
		subscriptions = append(subscriptions, subscriptionPage.Items...)
		return nil
	}
	switch query.Scope {
	case "personal":
		expensePage, loadErr := s.ListPersonalExpenses(ctx, userID, ports.PageRequest{Page: 1, PerPage: 100, Sort: "-incurred_on"})
		if loadErr != nil {
			return result, loadErr
		}
		subscriptionPage, loadErr := s.ListPersonalSubscriptions(ctx, userID, ports.PageRequest{Page: 1, PerPage: 100, Sort: "next_billing"})
		if loadErr != nil {
			return result, loadErr
		}
		expenses = expensePage.Items
		subscriptions = subscriptionPage.Items
	case "group":
		if query.GroupID == "" {
			return result, domain.ErrInvalid
		}
		if err = loadGroup(query.GroupID); err != nil {
			return result, err
		}
	case "all":
		groups, loadErr := s.ListGroups(ctx, userID, ports.PageRequest{Page: 1, PerPage: 100})
		if loadErr != nil {
			return result, loadErr
		}
		for _, group := range groups.Items {
			if err = loadGroup(group.ID); err != nil {
				return result, err
			}
		}
	}
	buckets := map[domain.Currency]*domain.CurrencyDashboard{}
	bucket := func(currency domain.Currency) *domain.CurrencyDashboard {
		if currency == "" {
			currency = domain.CurrencyTWD
		}
		if buckets[currency] == nil {
			buckets[currency] = &domain.CurrencyDashboard{Currency: currency}
		}
		return buckets[currency]
	}
	recordRange := func(groupID string) (time.Time, time.Time) {
		if groupID == "" || query.Scope == "group" {
			return start, end
		}
		zone := groupTimezone[groupID]
		if zone == "" {
			if group, groupErr := s.Stores.Groups.Get(ctx, groupID); groupErr == nil {
				zone = group.Timezone
				groupTimezone[groupID] = zone
			}
		}
		location := time.UTC
		if value, loadErr := time.LoadLocation(zone); loadErr == nil {
			location = value
		}
		value, _ := time.ParseInLocation("2006-01", start.Format("2006-01"), location)
		return value, value.AddDate(0, 1, 0)
	}
	consumeExpense := func(expense domain.Expense) {
		recordStart, recordEnd := recordRange(expense.GroupID)
		if expense.IncurredOn.Before(recordStart) || !expense.IncurredOn.Before(recordEnd) {
			return
		}
		displayCurrency, displayAmount := expense.Currency, expense.AmountMinor
		if query.Scope == "group" {
			displayCurrency, displayAmount = expense.BaseCurrency, expense.BaseAmountMinor
		}
		item := bucket(displayCurrency)
		share := int64(0)
		for _, split := range expense.Splits {
			if split.UserID == userID {
				share = split.AmountMinor
				if query.Scope == "group" {
					share = split.BaseAmountMinor
				}
				break
			}
		}
		if expense.GroupID == "" {
			share = displayAmount
		}
		if query.Scope == "group" {
			item.CashOutflowMinor += displayAmount
		} else if expense.PaidBy == userID {
			item.CashOutflowMinor += displayAmount
		}
		item.PersonalShareMinor += share
		if expense.PaidBy == userID && displayAmount > share {
			item.ReimbursableMinor += displayAmount - share
		}
		result.MonthExpenseMinor += displayAmount
	}
	for _, expense := range expenses {
		consumeExpense(expense)
	}
	now := s.Now()
	monthSubscriptionExpenses := make([]domain.Expense, 0)
	historicalSubscriptionExpenses := make([]domain.Expense, 0)
	for _, subscription := range subscriptions {
		lifecycle := domain.SubscriptionLifecycle(subscription, now)
		if lifecycle != "active" && lifecycle != "ending" {
			continue
		}
		recordStart, recordEnd := recordRange(subscription.GroupID)
		location := s.accountingLocation(ctx, userID, subscription.GroupID)
		dates, _ := billingDatesBetween(subscription.StartsOn.In(location), subscription.BillingCycle, subscription.BillingInterval, recordStart, recordEnd)
		// Price the monthly figures off the revision governing the month being
		// viewed, not off the subscription's current (NextBilling) settings —
		// otherwise looking at an earlier month reports today's price next to
		// that month's real cash flow.
		rate := subscriptionRateFor(subscription, dates, recordEnd)
		displayCurrency, displayAmount := rate.Currency, rate.AmountMinor
		if query.Scope == "group" {
			displayCurrency, displayAmount = rate.BaseCurrency, rate.BaseAmountMinor
		}
		item := bucket(displayCurrency)
		monthly, _ := domain.MonthlyEquivalentWithInterval(displayAmount, subscription.BillingCycle, subscription.BillingInterval)
		subscriptionShare := subscriptionUserShare(rate.Splits, subscription.GroupID, userID, query.Scope, displayAmount)
		personalMonthly, _ := domain.MonthlyEquivalentWithInterval(subscriptionShare, subscription.BillingCycle, subscription.BillingInterval)
		item.MonthlySubscriptionMinor += monthly
		item.PersonalMonthlySubscriptionMinor += personalMonthly
		item.ActiveSubscriptions++
		result.MonthlySubscriptionMinor += monthly
		result.PersonalMonthlySubscriptionMinor += personalMonthly
		result.ActiveSubscriptions++
		addedUpcoming := false
		for _, date := range dates {
			if subscription.EndsOn == nil || !date.After(*subscription.EndsOn) {
				item.ChargeCount++
				if !addedUpcoming {
					result.Upcoming = append(result.Upcoming, subscription)
					addedUpcoming = true
				}
			}
		}
		monthExpenses, err := subscriptionExpensesBetween(subscription, recordStart, recordEnd, location)
		if err != nil {
			return result, err
		}
		monthSubscriptionExpenses = append(monthSubscriptionExpenses, monthExpenses...)
		if query.Scope == "group" {
			historicalExpenses, err := subscriptionExpensesBetween(subscription, subscription.StartsOn.In(location), end, location)
			if err != nil {
				return result, err
			}
			historicalSubscriptionExpenses = append(historicalSubscriptionExpenses, historicalExpenses...)
		}
	}
	for _, expense := range monthSubscriptionExpenses {
		consumeExpense(expense)
	}
	keys := make([]string, 0, len(buckets))
	for currency := range buckets {
		keys = append(keys, string(currency))
	}
	sort.Strings(keys)
	for _, currency := range keys {
		result.Currencies = append(result.Currencies, *buckets[domain.Currency(currency)])
	}
	if query.Scope == "group" {
		if group, groupErr := s.Stores.Groups.Get(ctx, query.GroupID); groupErr == nil {
			result.ReportingCurrency = group.Currency
		}
		allExpenses, loadErr := s.ListExpenses(ctx, userID, query.GroupID, ports.PageRequest{Page: 1, PerPage: 100})
		if loadErr != nil {
			return result, loadErr
		}
		filteredExpenses := make([]domain.Expense, 0, len(allExpenses.Items)+len(historicalSubscriptionExpenses))
		filteredExpenses = append(filteredExpenses, allExpenses.Items...)
		filteredExpenses = append(filteredExpenses, historicalSubscriptionExpenses...)
		filteredExpenses = filteredExpenses[:0]
		for _, expense := range append(allExpenses.Items, historicalSubscriptionExpenses...) {
			if expense.IncurredOn.Before(end) {
				filteredExpenses = append(filteredExpenses, expense)
			}
		}
		settlements, loadErr := s.Stores.Settlements.List(ctx, query.GroupID, ports.PageRequest{Page: 1, PerPage: 100, Sort: "-settled_on"})
		if loadErr != nil {
			return result, loadErr
		}
		filteredSettlements := settlements.Items[:0]
		for _, settlement := range settlements.Items {
			if settlement.SettledOn.Before(end) {
				filteredSettlements = append(filteredSettlements, settlement)
			}
		}
		resolvedExpenses, resolvedSettlements := s.resolvePlaceholderAliases(ctx, filteredExpenses, filteredSettlements)
		result.Balances = domain.MemberBalances(resolvedExpenses, resolvedSettlements)
	}
	return result, nil
}

// resolvePlaceholderAliases folds a bound placeholder "temp member" (see
// Service.CreateTempMember / CollaborationService.accept) into the real
// account it's linked to, so a balance accrued while someone was still a
// placeholder correctly counts toward their real account's total once they
// join, without rewriting any stored expense_splits/settlements rows. Stored
// history keeps pointing at the placeholder's own ID; only this read-time
// aggregation resolves through the link.
func (s *Service) resolvePlaceholderAliases(ctx context.Context, expenses []domain.Expense, settlements []domain.Settlement) ([]domain.Expense, []domain.Settlement) {
	aliases := map[string]string{}
	resolve := func(id string) string {
		if id == "" {
			return id
		}
		if linked, ok := aliases[id]; ok {
			return linked
		}
		linked := id
		if user, err := s.Stores.Users.Get(ctx, id); err == nil && user.Placeholder && user.LinkedUserID != "" {
			linked = user.LinkedUserID
		}
		aliases[id] = linked
		return linked
	}
	resolvedExpenses := make([]domain.Expense, len(expenses))
	for i, expense := range expenses {
		expense.PaidBy = resolve(expense.PaidBy)
		if len(expense.Splits) > 0 {
			splits := make([]domain.ExpenseSplit, len(expense.Splits))
			for j, split := range expense.Splits {
				split.UserID = resolve(split.UserID)
				splits[j] = split
			}
			expense.Splits = splits
		}
		resolvedExpenses[i] = expense
	}
	resolvedSettlements := make([]domain.Settlement, len(settlements))
	for i, settlement := range settlements {
		settlement.FromUserID = resolve(settlement.FromUserID)
		settlement.ToUserID = resolve(settlement.ToUserID)
		resolvedSettlements[i] = settlement
	}
	return resolvedExpenses, resolvedSettlements
}

// includePast lists billing dates that have already gone by, so a caller with
// ledger.records.historical_write can revise a closed period. It is ignored for
// anyone without that permission rather than failing, so the picker degrades to
// upcoming dates instead of erroring.
func (s *Service) BillingDates(ctx context.Context, userID, id, cursor string, limit int, includePast bool) (BillingDatePage, error) {
	subscription, err := s.Stores.Subscriptions.Get(ctx, id)
	if err != nil {
		return BillingDatePage{}, err
	}
	if subscription.GroupID == "" {
		if subscription.OwnerID != userID {
			return BillingDatePage{}, domain.ErrForbidden
		}
		includePast = false
	} else if err = s.role(ctx, subscription.GroupID, userID, false); err != nil {
		return BillingDatePage{}, err
	} else if includePast && s.groupPermission(ctx, userID, subscription.GroupID, "ledger.records.historical_write") != nil {
		includePast = false
	}
	location := s.accountingLocation(ctx, userID, subscription.GroupID)
	from := subscription.NextBilling.In(location)
	if includePast {
		from = subscription.StartsOn.In(location)
	}
	if cursor != "" {
		value, parseErr := time.ParseInLocation("2006-01-02", cursor, location)
		if parseErr != nil {
			return BillingDatePage{}, domain.ErrInvalid
		}
		from = value.AddDate(0, 0, 1)
	}
	dates, err := domain.BillingDatesWithInterval(subscription.StartsOn.In(location), subscription.BillingCycle, subscription.BillingInterval, from, limit)
	if err != nil {
		return BillingDatePage{}, err
	}
	result := BillingDatePage{Dates: dates}
	if len(dates) == limit {
		result.NextCursor = dates[len(dates)-1].Format("2006-01-02")
	}
	return result, nil
}

func (s *Service) ListSettlements(ctx context.Context, userID, groupID string, page ports.PageRequest) (ports.Page[domain.Settlement], error) {
	if err := s.role(ctx, groupID, userID, false); err != nil {
		return ports.Page[domain.Settlement]{}, err
	}
	return s.Stores.Settlements.List(ctx, groupID, page)
}
func (s *Service) CreateSettlement(ctx context.Context, userID string, value domain.Settlement) (*domain.Settlement, error) {
	if err := s.role(ctx, value.GroupID, userID, false); err != nil {
		return nil, err
	}
	if value.AmountMinor <= 0 || value.FromUserID == value.ToUserID || value.SettledOn.IsZero() {
		return nil, domain.ErrInvalid
	}
	if _, err := s.Stores.Memberships.GetRole(ctx, value.GroupID, value.FromUserID); err != nil {
		return nil, domain.ErrInvalid
	}
	if _, err := s.Stores.Memberships.GetRole(ctx, value.GroupID, value.ToUserID); err != nil {
		return nil, domain.ErrInvalid
	}
	group, err := s.Stores.Groups.Get(ctx, value.GroupID)
	if err != nil {
		return nil, err
	}
	// Recording your own repayment (self -> anyone) is a basic action every
	// member can do; recording on someone else's behalf (any from/to pair)
	// requires the dedicated permission so it isn't silently open to whoever
	// happens to hold the default member role.
	if userID != value.FromUserID {
		if permErr := s.groupPermission(ctx, userID, value.GroupID, "ledger.settlements.write"); permErr != nil {
			s.audit(ctx, userID, value.GroupID, "settlement.created", "settlement", "", "failure", encodeAuditSummary(map[string]any{"from_user_id": value.FromUserID, "to_user_id": value.ToUserID, "amount_minor": value.AmountMinor}, nil))
			return nil, domain.ErrForbidden
		}
	}
	value.CreatedBy = userID
	value.Currency, value.BaseCurrency = group.Currency, group.Currency
	value.BaseAmountMinor, value.RateScaled, value.ExchangeRate, value.ExchangeRateDate = value.AmountMinor, domain.ExchangeRateScale, "1", value.SettledOn
	if err = s.Stores.Settlements.Create(ctx, &value); err != nil {
		return nil, err
	}
	s.audit(ctx, userID, value.GroupID, "settlement.created", "settlement", value.ID, "success", encodeAuditSummary(map[string]any{"from_user_id": value.FromUserID, "to_user_id": value.ToUserID, "amount_minor": value.AmountMinor, "settled_on": value.SettledOn.Format("2006-01-02")}, nil))
	return &value, nil
}
func (s *Service) DeleteSettlement(ctx context.Context, userID, id string) error {
	value, err := s.Stores.Settlements.Get(ctx, id)
	if err != nil {
		return err
	}
	if _, err = s.Stores.Groups.Get(ctx, value.GroupID); err != nil {
		return err
	}
	if value.CreatedBy != userID {
		if permErr := s.groupPermission(ctx, userID, value.GroupID, "ledger.settlements.write"); permErr != nil {
			s.audit(ctx, userID, value.GroupID, "settlement.deleted", "settlement", id, "failure", encodeAuditSummary(map[string]any{"from_user_id": value.FromUserID, "to_user_id": value.ToUserID, "amount_minor": value.AmountMinor}, nil))
			return domain.ErrForbidden
		}
	}
	err = s.Stores.Settlements.Delete(ctx, id)
	if err == nil {
		s.audit(ctx, userID, value.GroupID, "settlement.deleted", "settlement", id, "success", encodeAuditSummary(map[string]any{"from_user_id": value.FromUserID, "to_user_id": value.ToUserID, "amount_minor": value.AmountMinor}, nil))
	}
	return err
}
