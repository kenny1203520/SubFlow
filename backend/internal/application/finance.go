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
	for _, expense := range expenses {
		recordStart, recordEnd := recordRange(expense.GroupID)
		if expense.IncurredOn.Before(recordStart) || !expense.IncurredOn.Before(recordEnd) {
			continue
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
	now := s.Now()
	for _, subscription := range subscriptions {
		lifecycle := domain.SubscriptionLifecycle(subscription, now)
		if lifecycle != "active" && lifecycle != "ending" {
			continue
		}
		displayCurrency, displayAmount := subscription.Currency, subscription.AmountMinor
		if query.Scope == "group" {
			displayCurrency, displayAmount = subscription.BaseCurrency, subscription.BaseAmountMinor
		}
		item := bucket(displayCurrency)
		monthly, _ := domain.MonthlyEquivalent(displayAmount, subscription.BillingCycle)
		item.MonthlySubscriptionMinor += monthly
		item.ActiveSubscriptions++
		result.MonthlySubscriptionMinor += monthly
		result.ActiveSubscriptions++
		recordStart, recordEnd := recordRange(subscription.GroupID)
		location := s.accountingLocation(ctx, userID, subscription.GroupID)
		dates, _ := domain.BillingDates(subscription.StartsOn.In(location), subscription.BillingCycle, recordStart, 4)
		for _, date := range dates {
			if !date.Before(recordEnd) {
				break
			}
			if subscription.EndsOn == nil || !date.After(*subscription.EndsOn) {
				item.ChargeCount++
				result.Upcoming = append(result.Upcoming, subscription)
			}
		}
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
		filteredExpenses := allExpenses.Items[:0]
		for _, expense := range allExpenses.Items {
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
		result.Balances = domain.MemberBalances(filteredExpenses, filteredSettlements)
	}
	return result, nil
}

func (s *Service) BillingDates(ctx context.Context, userID, id, cursor string, limit int) (BillingDatePage, error) {
	subscription, err := s.Stores.Subscriptions.Get(ctx, id)
	if err != nil {
		return BillingDatePage{}, err
	}
	if subscription.GroupID == "" {
		if subscription.OwnerID != userID {
			return BillingDatePage{}, domain.ErrForbidden
		}
	} else if err = s.role(ctx, subscription.GroupID, userID, false); err != nil {
		return BillingDatePage{}, err
	}
	location := s.accountingLocation(ctx, userID, subscription.GroupID)
	from := subscription.NextBilling.In(location)
	if cursor != "" {
		value, parseErr := time.ParseInLocation("2006-01-02", cursor, location)
		if parseErr != nil {
			return BillingDatePage{}, domain.ErrInvalid
		}
		from = value.AddDate(0, 0, 1)
	}
	dates, err := domain.BillingDates(subscription.StartsOn.In(location), subscription.BillingCycle, from, limit)
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
	if userID != value.FromUserID && userID != group.OwnerID {
		return nil, domain.ErrForbidden
	}
	value.CreatedBy = userID
	value.Currency, value.BaseCurrency = group.Currency, group.Currency
	value.BaseAmountMinor, value.RateScaled, value.ExchangeRate, value.ExchangeRateDate = value.AmountMinor, domain.ExchangeRateScale, "1", value.SettledOn
	if err = s.Stores.Settlements.Create(ctx, &value); err != nil {
		return nil, err
	}
	s.audit(ctx, userID, value.GroupID, "settlement.created", "settlement", value.ID, "success")
	return &value, nil
}
func (s *Service) DeleteSettlement(ctx context.Context, userID, id string) error {
	value, err := s.Stores.Settlements.Get(ctx, id)
	if err != nil {
		return err
	}
	group, err := s.Stores.Groups.Get(ctx, value.GroupID)
	if err != nil {
		return err
	}
	if value.CreatedBy != userID && group.OwnerID != userID {
		return domain.ErrForbidden
	}
	err = s.Stores.Settlements.Delete(ctx, id)
	if err == nil {
		s.audit(ctx, userID, value.GroupID, "settlement.deleted", "settlement", id, "success")
	}
	return err
}
