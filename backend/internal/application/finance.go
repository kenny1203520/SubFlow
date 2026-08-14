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

// SubscriptionPeriod is one billing period of a subscription with the price
// that governs it already resolved, so the UI can show what each period cost
// without re-implementing revision resolution client-side. Status mirrors the
// occurrence record: "posted" once it has become an expense, "failed" when
// posting could not produce one, and "pending" for periods not yet billed.
type SubscriptionPeriod struct {
	BillingAt       time.Time             `json:"billingAt"`
	AmountMinor     int64                 `json:"amountMinor"`
	Currency        domain.Currency       `json:"currency"`
	BaseAmountMinor int64                 `json:"baseAmountMinor"`
	BaseCurrency    domain.Currency       `json:"baseCurrency"`
	PaidBy          string                `json:"paidBy"`
	Splits          []domain.ExpenseSplit `json:"splits,omitempty"`
	Status          string                `json:"status"`
	ExpenseID       string                `json:"expenseId,omitempty"`
	Error           string                `json:"error,omitempty"`
}

type SubscriptionPeriodPage struct {
	Periods    []SubscriptionPeriod `json:"periods"`
	NextCursor string               `json:"nextCursor,omitempty"`
}

// maxBillingDatesPerWindow bounds how many billing dates one query may
// materialise. It applies to the requested window rather than to the whole
// schedule, so an old subscription stays as cheap to query as a new one.
const maxBillingDatesPerWindow = 12000

// billingIndexAt returns the first schedule index whose billing date is not
// before `from`. Billing dates increase monotonically with the index, so this
// binary-searches for the window instead of walking every period since
// StartsOn: an hourly subscription older than about a year and a half used to
// exhaust the iteration cap before it ever reached the requested month and
// then reported no billing dates at all, silently dropping the subscription
// from the dashboard.
func billingIndexAt(start time.Time, cycle domain.BillingCycle, interval int, from time.Time) (int, error) {
	if !start.Before(from) {
		return 0, nil
	}
	low, high := 0, 1
	for {
		value, err := domain.BillingDateWithInterval(start, cycle, interval, high)
		if err != nil {
			return 0, err
		}
		if !value.Before(from) {
			break
		}
		if high > 1<<40 {
			return 0, domain.ErrInvalid
		}
		low, high = high, high*2
	}
	for low < high {
		mid := low + (high-low)/2
		value, err := domain.BillingDateWithInterval(start, cycle, interval, mid)
		if err != nil {
			return 0, err
		}
		if value.Before(from) {
			low = mid + 1
		} else {
			high = mid
		}
	}
	return low, nil
}

func billingDatesBetween(start time.Time, cycle domain.BillingCycle, interval int, from, to time.Time) ([]time.Time, error) {
	if !from.Before(to) {
		return nil, nil
	}
	first, err := billingIndexAt(start, cycle, interval, from)
	if err != nil {
		return nil, err
	}
	dates := make([]time.Time, 0)
	for count := 0; count < maxBillingDatesPerWindow; count++ {
		value, err := domain.BillingDateWithInterval(start, cycle, interval, first+count)
		if err != nil {
			return nil, err
		}
		if !value.Before(to) {
			break
		}
		dates = append(dates, value)
	}
	return dates, nil
}

func subscriptionOccurrenceAt(subscription domain.Subscription, billingAt time.Time) (domain.SubscriptionOccurrence, bool) {
	for _, occurrence := range subscription.Occurrences {
		if occurrence.BillingAt.Equal(billingAt) {
			return occurrence, true
		}
	}
	return domain.SubscriptionOccurrence{}, false
}

func subscriptionHasPostedOccurrence(subscription domain.Subscription, billingAt time.Time) bool {
	occurrence, ok := subscriptionOccurrenceAt(subscription, billingAt)
	return ok && occurrence.ExpenseID != ""
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
	// Every load below walks all pages: a single 100-row page silently
	// truncated the inputs to the month's totals and, worse, to
	// domain.MemberBalances, so any group past 100 expenses reported wrong
	// amounts owed.
	loadGroup := func(groupID string) error {
		groupExpenses, loadErr := listAllExpenses(ctx, s, userID, groupID)
		if loadErr != nil {
			return loadErr
		}
		groupSubscriptions, loadErr := listAllSubscriptions(ctx, s, userID, groupID)
		if loadErr != nil {
			return loadErr
		}
		group, loadErr := s.Stores.Groups.Get(ctx, groupID)
		if loadErr != nil {
			return loadErr
		}
		groupTimezone[groupID] = group.Timezone
		expenses = append(expenses, groupExpenses...)
		subscriptions = append(subscriptions, groupSubscriptions...)
		return nil
	}
	switch query.Scope {
	case "personal":
		expenses, err = listAllPersonalExpenses(ctx, s, userID)
		if err != nil {
			return result, err
		}
		subscriptions, err = listAllPersonalSubscriptions(ctx, s, userID)
		if err != nil {
			return result, err
		}
	case "group":
		if query.GroupID == "" {
			return result, domain.ErrInvalid
		}
		if err = loadGroup(query.GroupID); err != nil {
			return result, err
		}
	case "all":
		groups, loadErr := listAllGroups(ctx, s, userID)
		if loadErr != nil {
			return result, loadErr
		}
		for _, group := range groups {
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
	// Ordered here rather than relying on the load order, so the soonest
	// charge stays first no matter how the paginated list happened to sort.
	sort.SliceStable(result.Upcoming, func(i, j int) bool {
		return result.Upcoming[i].NextBilling.Before(result.Upcoming[j].NextBilling)
	})
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
		allExpenses, loadErr := listAllExpenses(ctx, s, userID, query.GroupID)
		if loadErr != nil {
			return result, loadErr
		}
		balanceExpenses := append(allExpenses, historicalSubscriptionExpenses...)
		filteredExpenses := make([]domain.Expense, 0, len(balanceExpenses))
		for _, expense := range balanceExpenses {
			if expense.IncurredOn.Before(end) {
				filteredExpenses = append(filteredExpenses, expense)
			}
		}
		allSettlements, loadErr := listAllSettlements(ctx, s, userID, query.GroupID)
		if loadErr != nil {
			return result, loadErr
		}
		filteredSettlements := make([]domain.Settlement, 0, len(allSettlements))
		for _, settlement := range allSettlements {
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

// SubscriptionPeriods walks a subscription's schedule from the start and
// reports what each period costs, which revision-scoped price changes make
// impossible to read off the subscription's current fields alone. It also
// surfaces occurrences that failed to post, which nothing else exposes to the
// user. Resolution reuses subscriptionExpenseOccurrence so these figures match
// the dashboard's, and a posted period reports its real expense so a directly
// edited charge isn't misreported as the revision's price.
func (s *Service) SubscriptionPeriods(ctx context.Context, userID, id, cursor string, limit int) (SubscriptionPeriodPage, error) {
	subscription, err := s.Stores.Subscriptions.Get(ctx, id)
	if err != nil {
		return SubscriptionPeriodPage{}, err
	}
	if subscription.GroupID == "" {
		if subscription.OwnerID != userID {
			return SubscriptionPeriodPage{}, domain.ErrForbidden
		}
	} else if err = s.role(ctx, subscription.GroupID, userID, false); err != nil {
		return SubscriptionPeriodPage{}, err
	}
	if limit < 1 || limit > 100 {
		limit = 24
	}
	s.hydrateSubscription(ctx, subscription)

	location := s.accountingLocation(ctx, userID, subscription.GroupID)
	from := subscription.StartsOn.In(location)
	if cursor != "" {
		// Nanosecond precision rather than a date, so hourly cadences that
		// bill many times a day don't skip the rest of the day's periods.
		value, parseErr := time.Parse(time.RFC3339Nano, cursor)
		if parseErr != nil {
			return SubscriptionPeriodPage{}, domain.ErrInvalid
		}
		from = value.In(location).Add(time.Nanosecond)
	}
	dates, err := domain.BillingDatesWithInterval(subscription.StartsOn.In(location), subscription.BillingCycle, subscription.BillingInterval, from, limit)
	if err != nil {
		return SubscriptionPeriodPage{}, err
	}

	periods := make([]SubscriptionPeriod, 0, len(dates))
	for _, date := range dates {
		if subscription.EndsOn != nil && date.After(*subscription.EndsOn) {
			break
		}
		resolved := subscriptionExpenseOccurrence(*subscription, date)
		period := SubscriptionPeriod{
			BillingAt: date, Status: "pending",
			AmountMinor: resolved.AmountMinor, Currency: resolved.Currency,
			BaseAmountMinor: resolved.BaseAmountMinor, BaseCurrency: resolved.BaseCurrency,
			PaidBy: resolved.PaidBy, Splits: resolved.Splits,
		}
		if occurrence, ok := subscriptionOccurrenceAt(*subscription, date); ok {
			period.Status, period.ExpenseID, period.Error = occurrence.Status, occurrence.ExpenseID, occurrence.Error
			if occurrence.ExpenseID != "" {
				// Splits live in their own collection, so the expense has to be
				// hydrated before its breakdown is readable.
				if expense, expenseErr := s.Stores.Expenses.Get(ctx, occurrence.ExpenseID); expenseErr == nil && s.hydrateExpense(ctx, expense) == nil {
					period.AmountMinor, period.Currency = expense.AmountMinor, expense.Currency
					period.BaseAmountMinor, period.BaseCurrency = expense.BaseAmountMinor, expense.BaseCurrency
					period.PaidBy, period.Splits = expense.PaidBy, expense.Splits
				}
			}
		}
		periods = append(periods, period)
	}
	result := SubscriptionPeriodPage{Periods: periods}
	if len(periods) == limit {
		result.NextCursor = periods[len(periods)-1].BillingAt.Format(time.RFC3339Nano)
	}
	return result, nil
}

// maxBackfillPeriods bounds one BackfillSubscriptionPeriods call: an hourly
// subscription backdated a couple of years would otherwise generate tens of
// thousands of real expense rows in one request.
const maxBackfillPeriods = 600

// BackfillSubscriptionPeriods posts real Expense + occurrence records for
// every period between a group subscription's StartsOn and its current
// NextBilling that never posted — which is every one of them, since
// CreateSubscription always initializes NextBilling to the first date on or
// after "now" (see Service.CreateSubscription), so a backdated StartsOn used
// to migrate an existing subscription into SubFlow otherwise never gets real
// records for its past periods: they only ever appear as on-the-fly
// synthesized data in the dashboard and SubscriptionPeriods, invisible to the
// expense list and the ledger export. Reuses postOccurrenceAt so a
// backfilled period is priced and split exactly like a regularly-posted one,
// and is idempotent against periods a previous backfill (or the cron) has
// already posted. Returns the number of periods newly created.
func (s *Service) BackfillSubscriptionPeriods(ctx context.Context, userID, id string) (int, error) {
	subscription, err := s.Stores.Subscriptions.Get(ctx, id)
	if err != nil {
		return 0, err
	}
	if subscription.GroupID == "" {
		// Personal subscriptions never post real occurrences at all (see
		// postSubscriptionOccurrence); there is nothing to backfill.
		return 0, domain.ErrInvalid
	}
	if err = s.groupPermission(ctx, userID, subscription.GroupID, "ledger.records.historical_write"); err != nil {
		s.audit(ctx, userID, subscription.GroupID, "subscription.backfilled", "subscription", subscription.ID, "failure")
		return 0, err
	}
	location := s.accountingLocation(ctx, userID, subscription.GroupID)
	startsOn := subscription.StartsOn.In(location)
	dates, err := billingDatesBetween(startsOn, subscription.BillingCycle, subscription.BillingInterval, startsOn, subscription.NextBilling)
	if err != nil {
		return 0, err
	}
	if len(dates) > maxBackfillPeriods {
		s.audit(ctx, userID, subscription.GroupID, "subscription.backfilled", "subscription", subscription.ID, "failure", encodeAuditSummary(map[string]any{"periods": len(dates), "limit": maxBackfillPeriods}, nil))
		return 0, domain.ErrInvalid
	}
	if len(dates) == 0 {
		return 0, nil
	}
	if err = s.ensureBaseRevisionCovers(ctx, subscription, startsOn); err != nil {
		return 0, err
	}
	created := 0
	for _, date := range dates {
		if subscription.EndsOn != nil && date.After(*subscription.EndsOn) {
			break
		}
		ok, postErr := s.postOccurrenceAt(ctx, subscription, date, userID)
		if postErr != nil {
			return created, postErr
		}
		if ok {
			created++
		}
	}
	s.audit(ctx, userID, subscription.GroupID, "subscription.backfilled", "subscription", subscription.ID, "success", encodeAuditSummary(map[string]any{"created": created, "starts_on": startsOn.Format("2006-01-02"), "through": subscription.NextBilling.Format("2006-01-02")}, nil))
	return created, nil
}

// ensureBaseRevisionCovers makes sure some revision resolves for "at" before
// a backfill starts posting historical periods. A freshly created
// subscription's only revision is effective from NextBilling onward (see
// Service.CreateSubscription), so a backdated StartsOn otherwise has no
// revision covering any of the periods a backfill needs to price —
// postOccurrenceAt would then hard-fail every one of them, since unlike the
// display-only fallback in subscriptionExpenseOccurrence, a real posted
// occurrence cannot leave its revision relation empty (the schema requires
// it). Rather than inventing an ephemeral revision with no row to point at,
// this persists one real "future"-scoped revision snapshotting the
// subscription's current settings effective at "at" — exactly what the
// display fallback already implies to the user before backfilling, so
// subsequent reads resolve consistently instead of relying on a per-call
// synthetic. A no-op once any revision already covers "at" (including a
// re-run, or a subscription that already had a historical edit reaching back
// that far).
func (s *Service) ensureBaseRevisionCovers(ctx context.Context, subscription *domain.Subscription, at time.Time) error {
	revisions, err := s.Stores.Subscriptions.ListRevisions(ctx, subscription.ID)
	if err != nil {
		return err
	}
	if _, ok := subscriptionRevisionAt(revisions, at); ok {
		return nil
	}
	base := subscriptionRevision(*subscription, "future", at, nil)
	return s.Stores.Subscriptions.CreateRevision(ctx, &base)
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
