package application

import (
	"context"
	"strings"
	"time"

	"subflow/internal/adapters"
	"subflow/internal/domain"
	"subflow/internal/ports"
)

type Service struct {
	Stores adapters.Stores
	Now    func() time.Time
}

func New(stores adapters.Stores) *Service { return &Service{Stores: stores, Now: time.Now} }

func (s *Service) role(ctx context.Context, groupID, userID string, ownerOnly bool) error {
	role, err := s.Stores.Memberships.GetRole(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if ownerOnly && !domain.CanManageGroup(role) {
		return domain.ErrForbidden
	}
	if !ownerOnly && !domain.CanManageRecords(role) {
		return domain.ErrForbidden
	}
	return nil
}

func (s *Service) CreateGroup(ctx context.Context, userID string, group domain.Group) (*domain.Group, error) {
	group.Name = strings.TrimSpace(group.Name)
	if group.Name == "" || !domain.IsCurrency(group.Currency) {
		return nil, domain.ErrInvalid
	}
	group.OwnerID = userID
	if group.Color == "" {
		group.Color = "#6d5dfc"
	}
	err := s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if err := s.Stores.Groups.Create(tx, &group); err != nil {
			return err
		}
		return s.Stores.Memberships.Create(tx, &domain.Membership{GroupID: group.ID, UserID: userID, Role: domain.RoleOwner})
	})
	return &group, err
}

func (s *Service) ListGroups(ctx context.Context, userID string, page ports.PageRequest) (ports.Page[domain.Group], error) {
	return s.Stores.Groups.ListForUser(ctx, userID, page)
}

func (s *Service) GetGroup(ctx context.Context, userID, id string) (*domain.Group, error) {
	if err := s.role(ctx, id, userID, false); err != nil {
		return nil, err
	}
	return s.Stores.Groups.Get(ctx, id)
}

func (s *Service) UpdateGroup(ctx context.Context, userID string, group domain.Group) (*domain.Group, error) {
	if err := s.role(ctx, group.ID, userID, true); err != nil {
		return nil, err
	}
	current, err := s.Stores.Groups.Get(ctx, group.ID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(group.Name) == "" || !domain.IsCurrency(group.Currency) {
		return nil, domain.ErrInvalid
	}
	group.OwnerID = current.OwnerID
	if err = s.Stores.Groups.Update(ctx, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *Service) DeleteGroup(ctx context.Context, userID, id string) error {
	if err := s.role(ctx, id, userID, true); err != nil {
		return err
	}
	return s.Stores.Groups.Delete(ctx, id)
}
func (s *Service) ListMembers(ctx context.Context, userID, groupID string, page ports.PageRequest) (ports.Page[domain.Membership], error) {
	if err := s.role(ctx, groupID, userID, false); err != nil {
		return ports.Page[domain.Membership]{}, err
	}
	return s.Stores.Memberships.List(ctx, groupID, page)
}
func (s *Service) RemoveMember(ctx context.Context, userID, groupID, memberID string) error {
	if err := s.role(ctx, groupID, userID, true); err != nil {
		return err
	}
	role, err := s.Stores.Memberships.GetRole(ctx, groupID, memberID)
	if err != nil {
		return err
	}
	if role == domain.RoleOwner {
		return domain.ErrForbidden
	}
	return s.Stores.Memberships.Delete(ctx, groupID, memberID)
}

func validSubscription(v *domain.Subscription) bool {
	return strings.TrimSpace(v.Name) != "" && v.AmountMinor >= 0 && domain.IsCurrency(v.Currency) && (v.BillingCycle == domain.BillingMonthly || v.BillingCycle == domain.BillingQuarterly || v.BillingCycle == domain.BillingYearly) && (v.Status == domain.SubscriptionActive || v.Status == domain.SubscriptionPaused || v.Status == domain.SubscriptionCancelled) && !v.NextBilling.IsZero()
}
func (s *Service) CreateSubscription(ctx context.Context, userID string, v domain.Subscription) (*domain.Subscription, error) {
	if err := s.role(ctx, v.GroupID, userID, false); err != nil {
		return nil, err
	}
	if !validSubscription(&v) {
		return nil, domain.ErrInvalid
	}
	if err := s.Stores.Subscriptions.Create(ctx, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *Service) ListSubscriptions(ctx context.Context, userID, groupID string, page ports.PageRequest) (ports.Page[domain.Subscription], error) {
	if err := s.role(ctx, groupID, userID, false); err != nil {
		return ports.Page[domain.Subscription]{}, err
	}
	return s.Stores.Subscriptions.List(ctx, groupID, page)
}
func (s *Service) UpdateSubscription(ctx context.Context, userID string, v domain.Subscription) (*domain.Subscription, error) {
	current, err := s.Stores.Subscriptions.Get(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	if v.GroupID == "" {
		v.GroupID = current.GroupID
	}
	if v.GroupID != current.GroupID {
		return nil, domain.ErrForbidden
	}
	if err = s.role(ctx, current.GroupID, userID, false); err != nil {
		return nil, err
	}
	if !validSubscription(&v) {
		return nil, domain.ErrInvalid
	}
	if err = s.Stores.Subscriptions.Update(ctx, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *Service) DeleteSubscription(ctx context.Context, userID, id string) error {
	v, err := s.Stores.Subscriptions.Get(ctx, id)
	if err != nil {
		return err
	}
	if err = s.role(ctx, v.GroupID, userID, false); err != nil {
		return err
	}
	return s.Stores.Subscriptions.Delete(ctx, id)
}

func validExpense(v *domain.Expense) bool {
	return strings.TrimSpace(v.Title) != "" && v.AmountMinor >= 0 && !v.IncurredOn.IsZero() && v.PaidBy != ""
}
func (s *Service) CreateExpense(ctx context.Context, userID string, v domain.Expense) (*domain.Expense, error) {
	if err := s.role(ctx, v.GroupID, userID, false); err != nil {
		return nil, err
	}
	if _, err := s.Stores.Memberships.GetRole(ctx, v.GroupID, v.PaidBy); err != nil {
		return nil, domain.ErrInvalid
	}
	if !validExpense(&v) {
		return nil, domain.ErrInvalid
	}
	if err := s.Stores.Expenses.Create(ctx, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *Service) ListExpenses(ctx context.Context, userID, groupID string, page ports.PageRequest) (ports.Page[domain.Expense], error) {
	if err := s.role(ctx, groupID, userID, false); err != nil {
		return ports.Page[domain.Expense]{}, err
	}
	return s.Stores.Expenses.List(ctx, groupID, page)
}
func (s *Service) UpdateExpense(ctx context.Context, userID string, v domain.Expense) (*domain.Expense, error) {
	current, err := s.Stores.Expenses.Get(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	if v.GroupID == "" {
		v.GroupID = current.GroupID
	}
	if v.GroupID != current.GroupID {
		return nil, domain.ErrForbidden
	}
	if err = s.role(ctx, current.GroupID, userID, false); err != nil {
		return nil, err
	}
	if !validExpense(&v) {
		return nil, domain.ErrInvalid
	}
	if err = s.Stores.Expenses.Update(ctx, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *Service) DeleteExpense(ctx context.Context, userID, id string) error {
	v, err := s.Stores.Expenses.Get(ctx, id)
	if err != nil {
		return err
	}
	if err = s.role(ctx, v.GroupID, userID, false); err != nil {
		return err
	}
	return s.Stores.Expenses.Delete(ctx, id)
}

func (s *Service) Dashboard(ctx context.Context, userID, groupID string) (domain.DashboardSummary, error) {
	if err := s.role(ctx, groupID, userID, false); err != nil {
		return domain.DashboardSummary{}, err
	}
	subs, err := s.Stores.Subscriptions.List(ctx, groupID, ports.PageRequest{Page: 1, PerPage: 100, Sort: "next_billing"})
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	expenses, err := s.Stores.Expenses.List(ctx, groupID, ports.PageRequest{Page: 1, PerPage: 100, Sort: "-incurred_on"})
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	now := s.Now()
	result := domain.DashboardSummary{}
	for _, v := range subs.Items {
		if v.Status == domain.SubscriptionActive {
			result.ActiveSubscriptions++
			m, _ := domain.MonthlyEquivalent(v.AmountMinor, v.BillingCycle)
			result.MonthlySubscriptionMinor += m
			if !v.NextBilling.Before(now) && v.NextBilling.Before(now.AddDate(0, 1, 0)) {
				result.Upcoming = append(result.Upcoming, v)
			}
		}
	}
	for _, v := range expenses.Items {
		if v.IncurredOn.Year() == now.Year() && v.IncurredOn.Month() == now.Month() {
			result.MonthExpenseMinor += v.AmountMinor
		}
	}
	return result, nil
}
