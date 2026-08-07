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

func (s *Service) monthRange(ctx context.Context, userID, month string) (time.Time, time.Time, error) {
	location := time.Local
	if user, err := s.Stores.Users.Get(ctx, userID); err == nil && user.Timezone != "" {
		if value, loadErr := time.LoadLocation(user.Timezone); loadErr == nil {
			location = value
		}
	}
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
	start, end, err := s.monthRange(ctx, userID, query.Month)
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	result := domain.DashboardSummary{Month: start.Format("2006-01")}
	var expenses []domain.Expense
	var subscriptions []domain.Subscription
	groupCurrency := map[string]domain.Currency{}
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
		groupCurrency[groupID] = group.Currency
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
	for _, expense := range expenses {
		if expense.IncurredOn.Before(start) || !expense.IncurredOn.Before(end) {
			continue
		}
		item := bucket(expense.Currency)
		share := int64(0)
		for _, split := range expense.Splits {
			if split.UserID == userID {
				share = split.AmountMinor
				break
			}
		}
		if expense.GroupID == "" {
			share = expense.AmountMinor
		}
		if query.Scope == "group" {
			item.CashOutflowMinor += expense.AmountMinor
		} else if expense.PaidBy == userID {
			item.CashOutflowMinor += expense.AmountMinor
		}
		item.PersonalShareMinor += share
		if expense.PaidBy == userID && expense.AmountMinor > share {
			item.ReimbursableMinor += expense.AmountMinor - share
		}
		result.MonthExpenseMinor += expense.AmountMinor
	}
	now := s.Now()
	for _, subscription := range subscriptions {
		lifecycle := domain.SubscriptionLifecycle(subscription, now)
		if lifecycle != "active" && lifecycle != "ending" {
			continue
		}
		item := bucket(subscription.Currency)
		monthly, _ := domain.MonthlyEquivalent(subscription.AmountMinor, subscription.BillingCycle)
		item.MonthlySubscriptionMinor += monthly
		item.ActiveSubscriptions++
		result.MonthlySubscriptionMinor += monthly
		result.ActiveSubscriptions++
		dates, _ := domain.BillingDates(subscription.StartsOn, subscription.BillingCycle, start, 4)
		for _, date := range dates {
			if !date.Before(end) {
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
	from := subscription.NextBilling
	if cursor != "" {
		value, parseErr := time.Parse("2006-01-02", cursor)
		if parseErr != nil {
			return BillingDatePage{}, domain.ErrInvalid
		}
		from = value.AddDate(0, 0, 1)
	}
	dates, err := domain.BillingDates(subscription.StartsOn, subscription.BillingCycle, from, limit)
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
	if err = s.Stores.Settlements.Create(ctx, &value); err != nil {
		return nil, err
	}
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
	return s.Stores.Settlements.Delete(ctx, id)
}
