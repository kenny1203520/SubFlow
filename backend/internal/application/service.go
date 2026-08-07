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
	if group.Timezone == "" {
		if user, err := s.Stores.Users.Get(ctx, userID); err == nil {
			group.Timezone = user.Timezone
		}
		if group.Timezone == "" {
			group.Timezone = "UTC"
		}
	}
	if _, err := time.LoadLocation(group.Timezone); err != nil {
		return nil, domain.ErrInvalid
	}
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
	if group.Timezone == "" {
		group.Timezone = current.Timezone
		if group.Timezone == "" {
			group.Timezone = "UTC"
		}
	}
	if _, err = time.LoadLocation(group.Timezone); err != nil {
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
	return strings.TrimSpace(v.Name) != "" && v.AmountMinor >= 0 && domain.IsCurrency(v.Currency) && (v.BillingCycle == domain.BillingMonthly || v.BillingCycle == domain.BillingQuarterly || v.BillingCycle == domain.BillingYearly) && (v.Status == domain.SubscriptionActive || v.Status == domain.SubscriptionPaused || v.Status == domain.SubscriptionCancelled) && (!v.StartsOn.IsZero() || !v.NextBilling.IsZero())
}
func (s *Service) CreateSubscription(ctx context.Context, userID string, v domain.Subscription) (*domain.Subscription, error) {
	if v.GroupID != "" {
		v.OwnerID = ""
		if err := s.role(ctx, v.GroupID, userID, false); err != nil {
			return nil, err
		}
		if v.PaidBy == "" {
			v.PaidBy = userID
		}
		if _, err := s.Stores.Memberships.GetRole(ctx, v.GroupID, v.PaidBy); err != nil {
			return nil, domain.ErrInvalid
		}
		group, err := s.Stores.Groups.Get(ctx, v.GroupID)
		if err != nil {
			return nil, err
		}
		v.Currency = group.Currency
	} else {
		v.GroupID = ""
		v.OwnerID = userID
		if v.PaidBy == "" {
			v.PaidBy = userID
		}
		if v.PaidBy != userID {
			return nil, domain.ErrForbidden
		}
	}
	if v.StartsOn.IsZero() {
		v.StartsOn = v.NextBilling
	}
	location := s.accountingLocation(ctx, userID, v.GroupID)
	v.StartsOn = v.StartsOn.In(location)
	if next, err := domain.NextBilling(v.StartsOn, v.BillingCycle, s.Now().In(location)); err == nil {
		v.NextBilling = next
	}
	if !validSubscription(&v) {
		return nil, domain.ErrInvalid
	}
	if err := s.Stores.Subscriptions.Create(ctx, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *Service) ListPersonalSubscriptions(ctx context.Context, userID string, page ports.PageRequest) (ports.Page[domain.Subscription], error) {
	result, err := s.Stores.Subscriptions.ListPersonal(ctx, userID, page)
	if err == nil {
		for i := range result.Items {
			result.Items[i].LifecycleStatus = domain.SubscriptionLifecycle(result.Items[i], s.Now())
		}
	}
	return result, err
}

func (s *Service) PersonalDashboard(ctx context.Context, userID string, all bool) (domain.DashboardSummary, error) {
	subs, err := s.ListPersonalSubscriptions(ctx, userID, ports.PageRequest{Page: 1, PerPage: 100, Sort: "next_billing"})
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	expenses, err := s.ListPersonalExpenses(ctx, userID, ports.PageRequest{Page: 1, PerPage: 100, Sort: "-incurred_on"})
	if err != nil {
		return domain.DashboardSummary{}, err
	}
	now := s.Now()
	result := domain.DashboardSummary{}
	totals := map[domain.Currency]*domain.CurrencyDashboard{}
	for _, item := range subs.Items {
		item.LifecycleStatus = domain.SubscriptionLifecycle(item, now)
		if item.LifecycleStatus == "active" || item.LifecycleStatus == "ending" {
			amount, _ := domain.MonthlyEquivalent(item.AmountMinor, item.BillingCycle)
			bucket := totals[item.Currency]
			if bucket == nil {
				bucket = &domain.CurrencyDashboard{Currency: item.Currency}
				totals[item.Currency] = bucket
			}
			bucket.MonthlySubscriptionMinor += amount
			bucket.ActiveSubscriptions++
			result.ActiveSubscriptions++
			result.MonthlySubscriptionMinor += amount
			if !item.NextBilling.Before(now) && item.NextBilling.Before(now.AddDate(0, 1, 0)) {
				result.Upcoming = append(result.Upcoming, item)
			}
		}
	}
	for _, item := range expenses.Items {
		if item.IncurredOn.Year() == now.Year() && item.IncurredOn.Month() == now.Month() {
			bucket := totals[domain.CurrencyTWD]
			if bucket == nil {
				bucket = &domain.CurrencyDashboard{Currency: domain.CurrencyTWD}
				totals[domain.CurrencyTWD] = bucket
			}
			bucket.CashOutflowMinor += item.AmountMinor
			bucket.PersonalShareMinor += item.AmountMinor
			result.MonthExpenseMinor += item.AmountMinor
		}
	}
	for _, bucket := range totals {
		result.Currencies = append(result.Currencies, *bucket)
	}
	return result, nil
}

func (s *Service) StopSubscription(ctx context.Context, userID, id, endsOn string) (*domain.Subscription, error) {
	v, err := s.Stores.Subscriptions.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if v.GroupID == "" {
		if v.OwnerID != userID {
			return nil, domain.ErrForbidden
		}
	} else if err = s.role(ctx, v.GroupID, userID, false); err != nil {
		return nil, err
	}
	location := s.accountingLocation(ctx, userID, v.GroupID)
	value, err := time.ParseInLocation("2006-01-02", endsOn, location)
	if err != nil {
		value, err = time.Parse(time.RFC3339, endsOn)
		if err != nil {
			return nil, domain.ErrInvalid
		}
	}
	dates, dateErr := domain.BillingDates(v.StartsOn.In(location), v.BillingCycle, v.NextBilling.In(location), 1200)
	if dateErr != nil {
		return nil, dateErr
	}
	matched := false
	for _, current := range dates {
		if current.Format("2006-01-02") == value.Format("2006-01-02") {
			matched = true
			break
		}
		if current.After(value) {
			break
		}
	}
	if !matched || value.Before(v.NextBilling) {
		return nil, domain.ErrInvalid
	}
	v.EndsOn = &value
	if err = s.Stores.Subscriptions.Update(ctx, v); err != nil {
		return nil, err
	}
	v.LifecycleStatus = domain.SubscriptionLifecycle(*v, s.Now())
	return v, nil
}
func (s *Service) ResumeSubscription(ctx context.Context, userID, id string) (*domain.Subscription, error) {
	v, err := s.Stores.Subscriptions.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if v.GroupID == "" {
		if v.OwnerID != userID {
			return nil, domain.ErrForbidden
		}
	} else if err = s.role(ctx, v.GroupID, userID, false); err != nil {
		return nil, err
	}
	v.EndsOn = nil
	if err = s.Stores.Subscriptions.Update(ctx, v); err != nil {
		return nil, err
	}
	v.LifecycleStatus = domain.SubscriptionLifecycle(*v, s.Now())
	return v, nil
}
func (s *Service) ListSubscriptions(ctx context.Context, userID, groupID string, page ports.PageRequest) (ports.Page[domain.Subscription], error) {
	if err := s.role(ctx, groupID, userID, false); err != nil {
		return ports.Page[domain.Subscription]{}, err
	}
	result, err := s.Stores.Subscriptions.List(ctx, groupID, page)
	if err == nil {
		for i := range result.Items {
			result.Items[i].LifecycleStatus = domain.SubscriptionLifecycle(result.Items[i], s.Now())
		}
	}
	return result, err
}
func (s *Service) UpdateSubscription(ctx context.Context, userID string, v domain.Subscription) (*domain.Subscription, error) {
	current, err := s.Stores.Subscriptions.Get(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	v.GroupID = current.GroupID
	v.OwnerID = current.OwnerID
	v.PaidBy = current.PaidBy
	if current.GroupID == "" {
		if current.OwnerID != userID {
			return nil, domain.ErrForbidden
		}
	} else if err = s.role(ctx, current.GroupID, userID, false); err != nil {
		return nil, err
	}
	if v.StartsOn.IsZero() {
		v.StartsOn = current.StartsOn
	}
	location := s.accountingLocation(ctx, userID, current.GroupID)
	v.StartsOn = v.StartsOn.In(location)
	if next, nextErr := domain.NextBilling(v.StartsOn, v.BillingCycle, s.Now().In(location)); nextErr == nil {
		v.NextBilling = next
	} else {
		return nil, nextErr
	}
	v.EndsOn = current.EndsOn
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
	if v.GroupID == "" {
		if v.OwnerID != userID {
			return domain.ErrForbidden
		}
	} else if err = s.role(ctx, v.GroupID, userID, false); err != nil {
		return err
	}
	return s.Stores.Subscriptions.Delete(ctx, id)
}

func validExpense(v *domain.Expense) bool {
	return strings.TrimSpace(v.Title) != "" && v.AmountMinor >= 0 && domain.IsCurrency(v.Currency) && !v.IncurredOn.IsZero() && v.PaidBy != ""
}
func (s *Service) memberIDs(ctx context.Context, groupID string) ([]string, error) {
	page, err := s.Stores.Memberships.List(ctx, groupID, ports.PageRequest{Page: 1, PerPage: 100})
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(page.Items))
	for i, item := range page.Items {
		ids[i] = item.UserID
	}
	return ids, nil
}
func (s *Service) hydrateExpense(ctx context.Context, v *domain.Expense) error {
	splits, err := s.Stores.Expenses.ListSplits(ctx, v.ID)
	if err != nil {
		return err
	}
	v.Splits = splits
	return nil
}
func (s *Service) CreateExpense(ctx context.Context, userID string, v domain.Expense) (*domain.Expense, error) {
	if v.PaidBy == "" {
		v.PaidBy = userID
	}
	if v.GroupID != "" {
		v.OwnerID = ""
		if err := s.role(ctx, v.GroupID, userID, false); err != nil {
			return nil, err
		}
		members, err := s.memberIDs(ctx, v.GroupID)
		if err != nil {
			return nil, err
		}
		group, err := s.Stores.Groups.Get(ctx, v.GroupID)
		if err != nil {
			return nil, err
		}
		v.Currency = group.Currency
		if len(v.Splits) == 0 {
			v.Splits = []domain.ExpenseSplit{{UserID: v.PaidBy, AmountMinor: v.AmountMinor}}
			v.SplitMode = domain.SplitAmount
		}
		v.Splits, err = domain.CanonicalSplits(v.AmountMinor, v.PaidBy, v.SplitMode, v.Splits, members)
		if err != nil {
			return nil, err
		}
	} else {
		v.GroupID = ""
		v.OwnerID = userID
		if v.PaidBy != userID {
			return nil, domain.ErrForbidden
		}
		if v.Currency == "" {
			v.Currency = domain.CurrencyTWD
		}
		v.SplitMode = domain.SplitAmount
		v.Splits = []domain.ExpenseSplit{{UserID: userID, AmountMinor: v.AmountMinor}}
	}
	if !validExpense(&v) {
		return nil, domain.ErrInvalid
	}
	if err := s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if err := s.Stores.Expenses.Create(tx, &v); err != nil {
			return err
		}
		return s.Stores.Expenses.ReplaceSplits(tx, v.ID, v.Splits)
	}); err != nil {
		return nil, err
	}
	for i := range v.Splits {
		v.Splits[i].ExpenseID = v.ID
	}
	return &v, nil
}
func (s *Service) ListPersonalExpenses(ctx context.Context, userID string, page ports.PageRequest) (ports.Page[domain.Expense], error) {
	result, err := s.Stores.Expenses.ListPersonal(ctx, userID, page)
	if err != nil {
		return result, err
	}
	for i := range result.Items {
		if err = s.hydrateExpense(ctx, &result.Items[i]); err != nil {
			return result, err
		}
	}
	return result, nil
}
func (s *Service) ListExpenses(ctx context.Context, userID, groupID string, page ports.PageRequest) (ports.Page[domain.Expense], error) {
	if err := s.role(ctx, groupID, userID, false); err != nil {
		return ports.Page[domain.Expense]{}, err
	}
	result, err := s.Stores.Expenses.List(ctx, groupID, page)
	if err != nil {
		return result, err
	}
	for i := range result.Items {
		if err = s.hydrateExpense(ctx, &result.Items[i]); err != nil {
			return result, err
		}
	}
	return result, nil
}
func (s *Service) UpdateExpense(ctx context.Context, userID string, v domain.Expense) (*domain.Expense, error) {
	current, err := s.Stores.Expenses.Get(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	v.GroupID = current.GroupID
	v.OwnerID = current.OwnerID
	if current.GroupID == "" {
		if current.OwnerID != userID {
			return nil, domain.ErrForbidden
		}
		v.PaidBy = userID
		if v.Currency == "" {
			v.Currency = current.Currency
		}
		v.SplitMode = domain.SplitAmount
		v.Splits = []domain.ExpenseSplit{{UserID: userID, AmountMinor: v.AmountMinor}}
	} else {
		if err = s.role(ctx, current.GroupID, userID, false); err != nil {
			return nil, err
		}
		members, memberErr := s.memberIDs(ctx, current.GroupID)
		if memberErr != nil {
			return nil, memberErr
		}
		group, groupErr := s.Stores.Groups.Get(ctx, current.GroupID)
		if groupErr != nil {
			return nil, groupErr
		}
		v.Currency = group.Currency
		if len(v.Splits) == 0 {
			v.Splits = current.Splits
		}
		v.Splits, err = domain.CanonicalSplits(v.AmountMinor, v.PaidBy, v.SplitMode, v.Splits, members)
		if err != nil {
			return nil, err
		}
	}
	if !validExpense(&v) {
		return nil, domain.ErrInvalid
	}
	if err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if updateErr := s.Stores.Expenses.Update(tx, &v); updateErr != nil {
			return updateErr
		}
		return s.Stores.Expenses.ReplaceSplits(tx, v.ID, v.Splits)
	}); err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *Service) DeleteExpense(ctx context.Context, userID, id string) error {
	v, err := s.Stores.Expenses.Get(ctx, id)
	if err != nil {
		return err
	}
	if v.GroupID == "" {
		if v.OwnerID != userID {
			return domain.ErrForbidden
		}
	} else if err = s.role(ctx, v.GroupID, userID, false); err != nil {
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
