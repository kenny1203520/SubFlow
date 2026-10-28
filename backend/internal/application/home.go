package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

func (s *Service) HomeDashboard(ctx context.Context, userID string) (domain.HomeDashboard, error) {
	now := s.Now().UTC()
	userLocation := s.accountingLocation(ctx, userID, "")
	month := now.In(userLocation).Format("2006-01")
	result := domain.HomeDashboard{AsOf: now, Month: month, Receivables: []domain.CurrencyAmount{}, Payables: []domain.CurrencyAmount{}, Groups: []domain.HomeGroupPosition{}, Monthly: []domain.HomeMonthlyCashflow{}, Upcoming: []domain.HomeUpcoming{}, Reviews: []domain.SettlementReview{}, Actions: []domain.HomeAction{}}
	monthly := map[string]*domain.HomeMonthlyCashflow{}
	flow := func(currency domain.Currency, groupID, groupName, period, timezone string) *domain.HomeMonthlyCashflow {
		key := groupID + "\x00" + string(currency)
		if monthly[key] == nil {
			monthly[key] = &domain.HomeMonthlyCashflow{GroupID: groupID, GroupName: groupName, Month: period, Timezone: timezone, Currency: currency}
		}
		return monthly[key]
	}
	personal, err := s.WorkspaceDashboard(ctx, userID, DashboardQuery{Scope: "personal", Month: month})
	if err != nil {
		return result, err
	}
	for _, value := range personal.Currencies {
		item := flow(value.Currency, "", "", month, userLocation.String())
		item.IncomeReceivedMinor += value.ReceivedIncomeMinor
		item.ExpensePaidMinor += value.CashOutflowMinor
		item.BeneficialIncomeMinor += value.BeneficialIncomeMinor
		item.PersonalExpenseShareMinor += value.PersonalShareMinor
	}
	personalSubscriptions, err := listAllPersonalSubscriptions(ctx, s, userID)
	if err != nil {
		return result, err
	}
	personalUpcoming, err := homeUpcoming(ctx, s, userID, "", personalSubscriptions, now)
	if err != nil {
		return result, err
	}
	result.Upcoming = append(result.Upcoming, personalUpcoming...)
	personalIncomeUpcoming, personalIncomeActions, incomeErr := homeRecurringIncome(ctx, s, userID, "", "", now)
	if incomeErr != nil {
		return result, incomeErr
	}
	result.Upcoming = append(result.Upcoming, personalIncomeUpcoming...)
	result.Actions = append(result.Actions, personalIncomeActions...)

	groups, err := listAllGroups(ctx, s, userID)
	if err != nil {
		return result, err
	}
	positionTotals := map[domain.Currency]map[string]int64{}
	incomplete := map[string]bool{}
	for _, group := range groups {
		readable := true
		for _, permission := range []string{"ledger.expenses.read", "ledger.incomes.read", "ledger.subscriptions.read", "ledger.settlements.read"} {
			permissionErr := s.groupPermission(ctx, userID, group.ID, permission)
			if permissionErr != nil {
				if errors.Is(permissionErr, domain.ErrForbidden) {
					readable = false
					break
				}
				return result, permissionErr
			}
		}
		if !readable {
			incomplete[group.ID] = true
			result.Groups = append(result.Groups, domain.HomeGroupPosition{GroupID: group.ID, GroupName: group.Name, Timezone: group.Timezone, Currency: group.Currency})
			continue
		}
		groupMonth := now.In(s.accountingLocation(ctx, userID, group.ID)).Format("2006-01")
		groupSummary, summaryErr := s.WorkspaceDashboard(ctx, userID, DashboardQuery{Scope: "group", GroupID: group.ID, Month: groupMonth, BalanceAsOf: now})
		if summaryErr != nil {
			if errors.Is(summaryErr, domain.ErrForbidden) {
				incomplete[group.ID] = true
				continue
			}
			return result, summaryErr
		}
		if len(groupSummary.IncomeCurrencyRepairIDs) > 0 {
			incomplete[group.ID] = true
		}
		groupSubscriptions, loadErr := listAllSubscriptions(ctx, s, userID, group.ID)
		if loadErr != nil {
			return result, loadErr
		}
		groupUpcoming, upcomingErr := homeUpcoming(ctx, s, userID, group.ID, groupSubscriptions, now)
		if upcomingErr != nil {
			return result, upcomingErr
		}
		result.Upcoming = append(result.Upcoming, groupUpcoming...)
		groupIncomeUpcoming, groupIncomeActions, incomeErr := homeRecurringIncome(ctx, s, userID, group.ID, group.Name, now)
		if incomeErr != nil {
			return result, incomeErr
		}
		result.Upcoming = append(result.Upcoming, groupIncomeUpcoming...)
		result.Actions = append(result.Actions, groupIncomeActions...)
		expenses, loadErr := listAllExpenses(ctx, s, userID, group.ID)
		if loadErr != nil {
			return result, loadErr
		}
		balanceAvailable := len(groupSummary.IncomeCurrencyRepairIDs) == 0
		for _, expense := range expenses {
			if (expense.BaseCurrency != "" && expense.BaseCurrency != group.Currency) || (expense.BaseCurrency == "" && expense.Currency != group.Currency) {
				incomplete[group.ID] = true
				balanceAvailable = false
			}
		}
		if !balanceAvailable {
			incomplete[group.ID] = true
		}
		balance := int64(0)
		for _, value := range groupSummary.Balances {
			if value.UserID == userID {
				balance = value.AmountMinor
				break
			}
		}
		result.Groups = append(result.Groups, domain.HomeGroupPosition{GroupID: group.ID, GroupName: group.Name, Timezone: group.Timezone, Currency: group.Currency, BalanceMinor: balance, BalanceAvailable: balanceAvailable})
		if balanceAvailable {
			if positionTotals[group.Currency] == nil {
				positionTotals[group.Currency] = map[string]int64{"receivable": 0, "payable": 0}
			}
			if balance > 0 {
				positionTotals[group.Currency]["receivable"] += balance
			}
			if balance < 0 {
				positionTotals[group.Currency]["payable"] -= balance
			}
		}

		start, end, rangeErr := s.monthRange(ctx, userID, group.ID, groupMonth)
		if rangeErr != nil {
			return result, rangeErr
		}
		if !balanceAvailable {
			// A missing conversion makes both the shared balance and the monthly
			// group totals ambiguous. Do not show a convincing but partial zero.
			if s.Stores.SettlementReviews != nil {
				reviews, reviewErr := s.ListSettlementReviews(ctx, userID, group.ID)
				if reviewErr == nil {
					result.Reviews = append(result.Reviews, reviews...)
				} else if !errors.Is(reviewErr, domain.ErrForbidden) {
					return result, reviewErr
				}
			}
			continue
		}
		for _, expense := range expenses {
			if expense.IncurredOn.Before(start) || !expense.IncurredOn.Before(end) {
				continue
			}
			currency, amount := expense.BaseCurrency, expense.BaseAmountMinor
			if currency == "" && expense.Currency == group.Currency {
				currency, amount = expense.Currency, expense.AmountMinor
			}
			if currency == "" {
				continue
			}
			item := flow(currency, group.ID, group.Name, groupMonth, s.accountingLocation(ctx, userID, group.ID).String())
			for _, split := range expense.Splits {
				if split.UserID == userID {
					share := split.BaseAmountMinor
					if share == 0 && expense.Currency == group.Currency {
						share = split.AmountMinor
					}
					item.PersonalExpenseShareMinor += share
					break
				}
			}
			if expense.PaidBy == userID {
				item.ExpensePaidMinor += amount
			}
		}
		incomes, loadErr := listAllGroupIncomes(ctx, s, userID, group.ID)
		if loadErr != nil {
			return result, loadErr
		}
		for _, income := range incomes {
			if income.ReceivedOn.Before(start) || !income.ReceivedOn.Before(end) {
				continue
			}
			currency := income.BaseCurrency
			if currency == "" && income.Currency == group.Currency {
				currency = income.Currency
			}
			if currency == "" {
				continue
			}
			item := flow(currency, group.ID, group.Name, groupMonth, s.accountingLocation(ctx, userID, group.ID).String())
			share, received := domain.IncomeAmounts(income, userID, true)
			item.BeneficialIncomeMinor += share
			item.IncomeReceivedMinor += received
		}
		settlements, loadErr := listAllSettlements(ctx, s, userID, group.ID)
		if loadErr != nil {
			return result, loadErr
		}
		for _, settlement := range settlements {
			if !domain.SettlementIsEffective(settlement) || settlement.SettledOn.Before(start) || !settlement.SettledOn.Before(end) {
				continue
			}
			item := flow(settlement.Currency, group.ID, group.Name, groupMonth, s.accountingLocation(ctx, userID, group.ID).String())
			if settlement.ToUserID == userID {
				item.SettlementReceivedMinor += settlement.AmountMinor
			}
			if settlement.FromUserID == userID {
				item.SettlementPaidMinor += settlement.AmountMinor
			}
		}
		if s.Stores.SettlementReviews != nil {
			reviews, reviewErr := s.ListSettlementReviews(ctx, userID, group.ID)
			if reviewErr == nil {
				result.Reviews = append(result.Reviews, reviews...)
			} else if !errors.Is(reviewErr, domain.ErrForbidden) {
				return result, reviewErr
			}
		}
	}

	for currency, totals := range positionTotals {
		if totals["receivable"] > 0 {
			result.Receivables = append(result.Receivables, domain.CurrencyAmount{Currency: currency, AmountMinor: totals["receivable"]})
		}
		if totals["payable"] > 0 {
			result.Payables = append(result.Payables, domain.CurrencyAmount{Currency: currency, AmountMinor: totals["payable"]})
		}
	}
	sort.Slice(result.Receivables, func(i, j int) bool { return result.Receivables[i].Currency < result.Receivables[j].Currency })
	sort.Slice(result.Payables, func(i, j int) bool { return result.Payables[i].Currency < result.Payables[j].Currency })
	sort.Slice(result.Groups, func(i, j int) bool {
		if result.Groups[i].GroupName == result.Groups[j].GroupName {
			return result.Groups[i].GroupID < result.Groups[j].GroupID
		}
		return result.Groups[i].GroupName < result.Groups[j].GroupName
	})
	sort.Slice(result.Upcoming, func(i, j int) bool { return result.Upcoming[i].NextBilling.Before(result.Upcoming[j].NextBilling) })
	for _, value := range monthly {
		result.Monthly = append(result.Monthly, *value)
	}
	sort.Slice(result.Monthly, func(i, j int) bool {
		if result.Monthly[i].GroupName != result.Monthly[j].GroupName {
			return result.Monthly[i].GroupName < result.Monthly[j].GroupName
		}
		return result.Monthly[i].Currency < result.Monthly[j].Currency
	})
	sort.Slice(result.Reviews, func(i, j int) bool { return result.Reviews[i].CreatedAt.Before(result.Reviews[j].CreatedAt) })
	for groupID := range incomplete {
		result.IncompleteIDs = append(result.IncompleteIDs, groupID)
	}
	sort.Strings(result.IncompleteIDs)
	if s.Stores.Notifications != nil {
		for _, category := range []string{domain.NotificationCategoryTask, domain.NotificationCategoryException} {
			pending, notificationErr := s.Stores.Notifications.ListFiltered(ctx, userID, ports.NotificationQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 100, Sort: "-created"}, Category: category, Status: domain.NotificationStatusPending})
			if notificationErr != nil {
				return result, notificationErr
			}
			for _, item := range pending.Items {
				if item.Type != "settlement_review_pending" {
					result.Actions = append(result.Actions, domain.HomeAction{ID: item.ID, Type: item.Type, GroupID: item.GroupID, ResourceID: item.ResourceID, Title: item.Title, Message: item.Message, Status: item.Status, CreatedAt: item.CreatedAt})
				}
			}
		}
		sort.Slice(result.Actions, func(i, j int) bool { return result.Actions[i].CreatedAt.After(result.Actions[j].CreatedAt) })
	}
	return result, nil
}

func homeUpcoming(ctx context.Context, s *Service, userID, groupID string, subscriptions []domain.Subscription, now time.Time) ([]domain.HomeUpcoming, error) {
	result := make([]domain.HomeUpcoming, 0, len(subscriptions))
	location := s.accountingLocation(ctx, userID, groupID)
	localNow := now.In(location)
	from := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location)
	to := from.AddDate(0, 0, 31)
	for _, subscription := range subscriptions {
		lifecycle := domain.SubscriptionLifecycle(subscription, now)
		if lifecycle != "active" && lifecycle != "ending" {
			continue
		}
		candidateFrom := subscription.NextBilling.In(location)
		if candidateFrom.Before(from) {
			candidateFrom = from
		}
		for attempt := 0; attempt < 32; attempt++ {
			dates, err := domain.BillingDatesWithInterval(subscription.StartsOn.In(location), subscription.BillingCycle, subscription.BillingInterval, candidateFrom, 1)
			if err != nil {
				return nil, err
			}
			if len(dates) == 0 || !dates[0].Before(to) || (subscription.EndsOn != nil && dates[0].After(subscription.EndsOn.In(location))) {
				break
			}
			if !subscriptionHasPostedOccurrence(subscription, dates[0]) {
				priced := subscriptionExpenseOccurrence(subscription, dates[0])
				result = append(result, domain.HomeUpcoming{ID: subscription.ID, GroupID: subscription.GroupID, Kind: "subscription", Name: priced.Title, NextBilling: dates[0], AmountMinor: priced.AmountMinor, Currency: priced.Currency, Timezone: location.String()})
				break
			}
			candidateFrom = dates[0].Add(time.Nanosecond)
		}
	}
	return result, nil
}

func homeRecurringIncome(ctx context.Context, s *Service, userID, groupID, groupName string, now time.Time) ([]domain.HomeUpcoming, []domain.HomeAction, error) {
	plans := make([]domain.RecurringIncome, 0)
	for page := 1; ; page++ {
		result, err := s.ListRecurringIncomes(ctx, userID, groupID, ports.PageRequest{Page: page, PerPage: exportPageSize, Sort: "next_on"})
		if err != nil {
			return nil, nil, err
		}
		plans = append(plans, result.Items...)
		if page >= result.TotalPages || len(result.Items) == 0 {
			break
		}
	}
	canConfirm := groupID == ""
	if groupID != "" {
		permissionErr := s.groupPermission(ctx, userID, groupID, "ledger.incomes.write")
		if permissionErr != nil && !errors.Is(permissionErr, domain.ErrForbidden) {
			return nil, nil, permissionErr
		}
		canConfirm = permissionErr == nil
	}
	upcoming := make([]domain.HomeUpcoming, 0, len(plans))
	actions := make([]domain.HomeAction, 0)
	for _, plan := range plans {
		if plan.Status != "active" {
			continue
		}
		location := s.accountingLocation(ctx, userID, groupID)
		if plan.Timezone != "" {
			if saved, err := time.LoadLocation(plan.Timezone); err == nil {
				location = saved
			}
		}
		localNow := now.In(location)
		today := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location)
		through := today.AddDate(0, 0, 31)
		occurrences := make([]domain.IncomeOccurrence, 0)
		for page := 1; ; page++ {
			result, err := s.ListIncomeOccurrences(ctx, userID, plan.ID, ports.PageRequest{Page: page, PerPage: exportPageSize, Sort: "-scheduled_on"})
			if err != nil {
				return nil, nil, err
			}
			occurrences = append(occurrences, result.Items...)
			if page >= result.TotalPages || len(result.Items) == 0 {
				break
			}
		}
		byDate := make(map[string]domain.IncomeOccurrence, len(occurrences))
		for _, occurrence := range occurrences {
			key := occurrence.ScheduledOn.In(location).Format("2006-01-02T15:04:05")
			byDate[key] = occurrence
			if canConfirm && occurrence.Status == "pending" && occurrence.PostingMode == "manual" && !occurrence.ScheduledOn.In(location).After(localNow) {
				share, received := domain.IncomeAmounts(occurrence.Template, userID, false)
				amount := received
				if amount == 0 {
					amount = share
				}
				actions = append(actions, domain.HomeAction{ID: "income-occurrence:" + occurrence.ID, Type: "recurring_income_manual_pending", GroupID: groupID, ResourceID: occurrence.RecurringIncomeID, Title: occurrence.Template.Title, Message: groupName, Status: occurrence.Status, CreatedAt: occurrence.CreatedAt, ScheduledOn: occurrence.ScheduledOn, AmountMinor: amount, Currency: occurrence.Template.Currency})
			}
		}

		candidate := plan.NextOn.In(location)
		if candidate.Before(today) {
			candidate = today
		}
		for attempt := 0; attempt < 32; attempt++ {
			dates, err := domain.BillingDatesWithInterval(plan.StartsOn.In(location), plan.Cycle, plan.Interval, candidate, 1)
			if err != nil {
				return nil, nil, err
			}
			if len(dates) == 0 || !dates[0].Before(through) || (plan.EndsOn != nil && dates[0].After(plan.EndsOn.In(location))) {
				break
			}
			dateKey := dates[0].In(location).Format("2006-01-02T15:04:05")
			if _, exists := byDate[dateKey]; exists {
				candidate = dates[0].Add(time.Nanosecond)
				continue
			}
			share, received := domain.IncomeAmounts(plan.Template, userID, false)
			amount := received
			if amount == 0 {
				amount = share
			}
			if amount > 0 {
				upcoming = append(upcoming, domain.HomeUpcoming{ID: plan.ID + ":" + dateKey, GroupID: groupID, Kind: "income", Name: plan.Template.Title, NextBilling: dates[0], AmountMinor: amount, Currency: plan.Template.Currency, Timezone: location.String()})
			}
			break
		}
	}
	return upcoming, actions, nil
}

func (s *Service) SettlementSuggestions(ctx context.Context, userID, groupID string) ([]domain.SettlementSuggestion, error) {
	if err := s.groupPermission(ctx, userID, groupID, "ledger.settlements.read"); err != nil {
		return nil, err
	}
	group, err := s.Stores.Groups.Get(ctx, groupID)
	if err != nil {
		return nil, err
	}
	now := s.Now().UTC()
	month := now.In(accountingLocation(ctx, s, userID, groupID)).Format("2006-01")
	summary, err := s.WorkspaceDashboard(ctx, userID, DashboardQuery{Scope: "group", GroupID: groupID, Month: month, BalanceAsOf: now})
	if err != nil {
		return nil, err
	}
	balances := map[string]int64{}
	for _, item := range summary.Balances {
		balances[item.UserID] = item.AmountMinor
	}
	if s.Stores.SettlementReviews != nil {
		pending, listErr := s.Stores.SettlementReviews.ListPending(ctx, groupID)
		if listErr != nil {
			return nil, listErr
		}
		for _, item := range pending {
			switch item.Action {
			case "create":
				amount := item.Proposal.BaseAmountMinor
				if amount == 0 {
					amount = item.Proposal.AmountMinor
				}
				balances[item.Proposal.FromUserID] += amount
				balances[item.Proposal.ToUserID] -= amount
			case "change", "void":
				balances[item.Proposal.FromUserID] = 0
				balances[item.Proposal.ToUserID] = 0
			}
		}
	}
	version := settlementBalanceVersion(groupID, group.Currency, balances)
	type memberAmount struct {
		id     string
		amount int64
	}
	creditors, debtors := []memberAmount{}, []memberAmount{}
	for id, amount := range balances {
		if amount > 0 {
			creditors = append(creditors, memberAmount{id, amount})
		}
		if amount < 0 {
			debtors = append(debtors, memberAmount{id, -amount})
		}
	}
	sort.Slice(creditors, func(i, j int) bool {
		if creditors[i].amount == creditors[j].amount {
			return creditors[i].id < creditors[j].id
		}
		return creditors[i].amount > creditors[j].amount
	})
	sort.Slice(debtors, func(i, j int) bool {
		if debtors[i].amount == debtors[j].amount {
			return debtors[i].id < debtors[j].id
		}
		return debtors[i].amount > debtors[j].amount
	})
	result := []domain.SettlementSuggestion{}
	for di, ci := 0, 0; di < len(debtors) && ci < len(creditors); {
		amount := debtors[di].amount
		if creditors[ci].amount < amount {
			amount = creditors[ci].amount
		}
		if amount > 0 {
			result = append(result, domain.SettlementSuggestion{GroupID: groupID, FromUserID: debtors[di].id, ToUserID: creditors[ci].id, AmountMinor: amount, Currency: group.Currency, Label: "simplified", Version: version})
		}
		debtors[di].amount -= amount
		creditors[ci].amount -= amount
		if debtors[di].amount == 0 {
			di++
		}
		if creditors[ci].amount == 0 {
			ci++
		}
	}
	return result, nil
}

func settlementBalanceVersion(groupID string, currency domain.Currency, balances map[string]int64) string {
	ids := make([]string, 0, len(balances))
	for id := range balances {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var payload strings.Builder
	_, _ = fmt.Fprintf(&payload, "%s|%s", groupID, currency)
	for _, id := range ids {
		_, _ = fmt.Fprintf(&payload, "|%s:%d", id, balances[id])
	}
	sum := sha256.Sum256([]byte(payload.String()))
	return hex.EncodeToString(sum[:16])
}

func accountingLocation(ctx context.Context, s *Service, userID, groupID string) *time.Location {
	return s.accountingLocation(ctx, userID, groupID)
}
