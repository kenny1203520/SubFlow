package application

import (
	"context"
	"errors"
	"sort"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

func pageCombined[T any](items []T, request ports.PageRequest) ports.Page[T] {
	if request.Page < 1 {
		request.Page = 1
	}
	if request.PerPage < 1 {
		request.PerPage = 25
	}
	total := len(items)
	pages := (total + request.PerPage - 1) / request.PerPage
	start := (request.Page - 1) * request.PerPage
	if start > total {
		start = total
	}
	end := start + request.PerPage
	if end > total {
		end = total
	}
	return ports.Page[T]{Items: items[start:end], Page: request.Page, PerPage: request.PerPage, TotalItems: total, TotalPages: pages}
}

func (s *Service) allScopeGroups(ctx context.Context, userID, permission string) ([]domain.Group, error) {
	groups, err := listAllGroups(ctx, s, userID)
	if err != nil {
		return nil, err
	}
	readable := make([]domain.Group, 0, len(groups))
	for _, group := range groups {
		if err := s.groupPermission(ctx, userID, group.ID, permission); err != nil {
			if errors.Is(err, domain.ErrForbidden) {
				continue
			}
			return nil, err
		}
		readable = append(readable, group)
	}
	return readable, nil
}

func (s *Service) ListAllScopeIncomes(ctx context.Context, userID string, request ports.PageRequest) (ports.Page[domain.Income], error) {
	incomes, err := listAllPersonalIncomes(ctx, s, userID)
	if err != nil {
		return ports.Page[domain.Income]{}, err
	}
	groups, err := s.allScopeGroups(ctx, userID, "ledger.incomes.read")
	if err != nil {
		return ports.Page[domain.Income]{}, err
	}
	for _, group := range groups {
		values, listErr := listAllGroupIncomes(ctx, s, userID, group.ID)
		if listErr != nil {
			return ports.Page[domain.Income]{}, listErr
		}
		incomes = append(incomes, values...)
	}
	sort.SliceStable(incomes, func(i, j int) bool {
		if request.Sort == "received_on" {
			return incomes[i].ReceivedOn.Before(incomes[j].ReceivedOn)
		}
		return incomes[i].ReceivedOn.After(incomes[j].ReceivedOn)
	})
	return pageCombined(incomes, request), nil
}

func (s *Service) ListAllScopeExpenses(ctx context.Context, userID string, request ports.PageRequest) (ports.Page[domain.Expense], error) {
	expenses, err := listAllPersonalExpenses(ctx, s, userID)
	if err != nil {
		return ports.Page[domain.Expense]{}, err
	}
	groups, err := s.allScopeGroups(ctx, userID, "ledger.expenses.read")
	if err != nil {
		return ports.Page[domain.Expense]{}, err
	}
	for _, group := range groups {
		values, listErr := listAllExpenses(ctx, s, userID, group.ID)
		if listErr != nil {
			return ports.Page[domain.Expense]{}, listErr
		}
		expenses = append(expenses, values...)
	}
	sort.SliceStable(expenses, func(i, j int) bool {
		if request.Sort == "incurred_on" {
			return expenses[i].IncurredOn.Before(expenses[j].IncurredOn)
		}
		return expenses[i].IncurredOn.After(expenses[j].IncurredOn)
	})
	return pageCombined(expenses, request), nil
}

func (s *Service) ListAllScopeSubscriptions(ctx context.Context, userID string, request ports.PageRequest) (ports.Page[domain.Subscription], error) {
	subscriptions, err := listAllPersonalSubscriptions(ctx, s, userID)
	if err != nil {
		return ports.Page[domain.Subscription]{}, err
	}
	groups, err := s.allScopeGroups(ctx, userID, "ledger.subscriptions.read")
	if err != nil {
		return ports.Page[domain.Subscription]{}, err
	}
	for _, group := range groups {
		values, listErr := listAllSubscriptions(ctx, s, userID, group.ID)
		if listErr != nil {
			return ports.Page[domain.Subscription]{}, listErr
		}
		subscriptions = append(subscriptions, values...)
	}
	sort.SliceStable(subscriptions, func(i, j int) bool {
		if request.Sort == "name" {
			return subscriptions[i].Name < subscriptions[j].Name
		}
		if request.Sort == "-name" {
			return subscriptions[i].Name > subscriptions[j].Name
		}
		if request.Sort == "next_billing" {
			return subscriptions[i].NextBilling.Before(subscriptions[j].NextBilling)
		}
		return subscriptions[i].NextBilling.After(subscriptions[j].NextBilling)
	})
	return pageCombined(subscriptions, request), nil
}
