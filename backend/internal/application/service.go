package application

import (
	"context"
	"os"
	"strings"
	"time"

	"subflow/internal/adapters"
	"subflow/internal/captcha"
	"subflow/internal/domain"
	"subflow/internal/ports"
	"subflow/internal/security"
)

type Service struct {
	Stores  adapters.Stores
	Now     func() time.Time
	Rates   RateProvider
	Captcha captcha.Verifier
	Cipher  security.SettingsCipher
}

func New(stores adapters.Stores) *Service {
	return &Service{Stores: stores, Now: time.Now, Captcha: captcha.NewVerifier(), Cipher: security.NewSettingsCipher(os.Getenv("SUBFLOW_SETTINGS_ENCRYPTION_KEY"))}
}

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

func (s *Service) expenseIsHistorical(ctx context.Context, userID string, value *domain.Expense) (bool, error) {
	if value.GroupID == "" {
		return false, nil
	}
	start, _, err := s.monthRange(ctx, userID, value.GroupID, "")
	if err != nil {
		return false, err
	}
	return value.IncurredOn.Before(start), nil
}

func historicalExpenseDetails(value *domain.Expense) map[string]any {
	return map[string]any{"title": value.Title, "incurred_on": value.IncurredOn.Format("2006-01-02"), "amount_minor": value.AmountMinor}
}

func historicalExpenseChangeSummary(before, after *domain.Expense) string {
	var changes changeSet
	changes.addString("title", before.Title, after.Title)
	changes.addString("incurred_on", before.IncurredOn.Format("2006-01-02"), after.IncurredOn.Format("2006-01-02"))
	changes.addInt64("amount_minor", before.AmountMinor, after.AmountMinor)
	changes.addString("paid_by", before.PaidBy, after.PaidBy)
	changes.addString("category_id", before.CategoryID, after.CategoryID)
	return encodeAuditSummary(historicalExpenseDetails(after), changes)
}

func historicalSubscriptionDetails(value *domain.Subscription, effective time.Time) map[string]any {
	return map[string]any{"name": value.Name, "billing_at": effective.Format("2006-01-02"), "amount_minor": value.AmountMinor}
}

func historicalSubscriptionChangeSummary(before, after *domain.Subscription, effective time.Time) string {
	var changes changeSet
	changes.addString("name", before.Name, after.Name)
	changes.addInt64("amount_minor", before.AmountMinor, after.AmountMinor)
	changes.addString("paid_by", before.PaidBy, after.PaidBy)
	changes.addString("split_mode", string(before.SplitMode), string(after.SplitMode))
	changes.addString("category_id", before.CategoryID, after.CategoryID)
	return encodeAuditSummary(historicalSubscriptionDetails(after, effective), changes)
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
	if err == nil {
		s.audit(ctx, userID, group.ID, "group.created", "group", group.ID, "success", encodeAuditSummary(map[string]any{"name": group.Name, "currency": group.Currency, "timezone": group.Timezone}, nil))
	}
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
	if group.Currency != current.Currency {
		return nil, domain.ErrConflict
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
	var groupChanges changeSet
	groupChanges.addString("name", current.Name, group.Name)
	groupChanges.addString("description", current.Description, group.Description)
	groupChanges.addString("timezone", current.Timezone, group.Timezone)
	groupChanges.addString("color", current.Color, group.Color)
	s.audit(ctx, userID, group.ID, "group.updated", "group", group.ID, "success", encodeAuditSummary(nil, groupChanges))
	return &group, nil
}

func (s *Service) DeleteGroup(ctx context.Context, userID, id string) error {
	if err := s.role(ctx, id, userID, true); err != nil {
		return err
	}
	deleted, deletedErr := s.Stores.Groups.Get(ctx, id)
	err := s.Stores.Groups.Delete(ctx, id)
	if err == nil {
		details := map[string]any(nil)
		if deletedErr == nil {
			details = map[string]any{"name": deleted.Name}
		}
		s.audit(ctx, userID, id, "group.deleted", "group", id, "success", encodeAuditSummary(details, nil))
	}
	return err
}
func (s *Service) ListMembers(ctx context.Context, userID, groupID string, page ports.PageRequest) (ports.Page[domain.Membership], error) {
	if err := s.role(ctx, groupID, userID, false); err != nil {
		return ports.Page[domain.Membership]{}, err
	}
	return s.Stores.Memberships.List(ctx, groupID, page)
}
func (s *Service) RemoveMember(ctx context.Context, userID, groupID, memberID string) error {
	if err := s.groupPermission(ctx, userID, groupID, "group.members.manage"); err != nil {
		return err
	}
	role, err := s.Stores.Memberships.GetRole(ctx, groupID, memberID)
	if err != nil {
		return err
	}
	if role == domain.RoleOwner {
		return domain.ErrForbidden
	}
	err = s.Stores.Memberships.Delete(ctx, groupID, memberID)
	if err == nil {
		s.audit(ctx, userID, groupID, "member.removed", "membership", memberID, "success", encodeAuditSummary(map[string]any{"role": string(role)}, nil))
	}
	return err
}

func validSubscription(v *domain.Subscription) bool {
	return strings.TrimSpace(v.Name) != "" && v.AmountMinor >= 0 && domain.IsCurrency(v.Currency) && domain.ValidBillingCycle(v.BillingCycle, v.BillingInterval) && (v.Status == domain.SubscriptionActive || v.Status == domain.SubscriptionPaused || v.Status == domain.SubscriptionCancelled) && (!v.StartsOn.IsZero() || !v.NextBilling.IsZero())
}
func (s *Service) CreateSubscription(ctx context.Context, userID string, v domain.Subscription) (*domain.Subscription, error) {
	if v.RateMode == "" {
		v.RateMode = domain.RateAutomatic
	}
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
		v.BaseCurrency = group.Currency
	} else {
		v.GroupID = ""
		v.OwnerID = userID
		if v.PaidBy == "" {
			v.PaidBy = userID
		}
		if v.PaidBy != userID {
			return nil, domain.ErrForbidden
		}
		v.BaseCurrency = v.Currency
	}
	if v.StartsOn.IsZero() {
		v.StartsOn = v.NextBilling
	}
	location := s.accountingLocation(ctx, userID, v.GroupID)
	v.StartsOn = v.StartsOn.In(location)
	if v.Currency == "" {
		v.Currency = v.BaseCurrency
	}
	if _, err := s.validateCategory(ctx, userID, v.GroupID, v.CategoryID); err != nil {
		return nil, err
	}
	baseAmount, rate, rateText, rateDate, err := s.conversion(ctx, v.Currency, v.BaseCurrency, v.AmountMinor, v.RateMode, v.ExchangeRate, v.StartsOn)
	if err != nil {
		return nil, err
	}
	v.BaseAmountMinor, v.RateScaled, v.ExchangeRate, v.ExchangeRateDate = baseAmount, rate, rateText, rateDate
	v.BillingInterval = domain.NormalizeBillingInterval(v.BillingCycle, v.BillingInterval)
	if next, err := domain.NextBillingWithInterval(v.StartsOn, v.BillingCycle, v.BillingInterval, s.Now().In(location)); err == nil {
		v.NextBilling = next
	}
	if !validSubscription(&v) {
		return nil, domain.ErrInvalid
	}
	if v.GroupID != "" {
		members, memberErr := s.memberIDs(ctx, v.GroupID)
		if memberErr != nil {
			return nil, memberErr
		}
		if len(v.Splits) == 0 {
			v.SplitMode, v.Splits = domain.SplitAmount, []domain.ExpenseSplit{{UserID: v.PaidBy, AmountMinor: v.AmountMinor}}
		}
		v.Splits, err = domain.CanonicalSplits(v.AmountMinor, v.PaidBy, v.SplitMode, v.Splits, members)
		if err != nil {
			return nil, err
		}
		for i := range v.Splits {
			v.Splits[i].BaseAmountMinor, err = domain.ConvertMinor(v.Splits[i].AmountMinor, v.Currency, v.BaseCurrency, v.RateScaled)
			if err != nil {
				return nil, err
			}
		}
		v.Splits = domain.CanonicalBaseSplits(v.BaseAmountMinor, v.PaidBy, v.Splits)
	}
	if err := s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if createErr := s.Stores.Subscriptions.Create(tx, &v); createErr != nil {
			return createErr
		}
		if v.GroupID == "" {
			return nil
		}
		revision := subscriptionRevision(v, "future", v.NextBilling)
		return s.Stores.Subscriptions.CreateRevision(tx, &revision)
	}); err != nil {
		return nil, err
	}
	s.audit(ctx, userID, v.GroupID, "subscription.created", "subscription", v.ID, "success", encodeAuditSummary(map[string]any{"name": v.Name, "amount_minor": v.AmountMinor, "billing_cycle": v.BillingCycle, "split_mode": v.SplitMode}, nil))
	if v.GroupID != "" {
		s.audit(ctx, userID, v.GroupID, "subscription.version_created", "subscription", v.ID, "success", encodeAuditSummary(map[string]any{"scope": "future", "effective_billing_at": v.NextBilling.Format("2006-01-02")}, nil))
	}
	return &v, nil
}
func (s *Service) ListPersonalSubscriptions(ctx context.Context, userID string, page ports.PageRequest) (ports.Page[domain.Subscription], error) {
	result, err := s.Stores.Subscriptions.ListPersonal(ctx, userID, page)
	if err == nil {
		for i := range result.Items {
			result.Items[i].LifecycleStatus = domain.SubscriptionLifecycle(result.Items[i], s.Now())
			result.Items[i].CategoryInfo = s.hydrateCategory(ctx, result.Items[i].CategoryID)
			s.hydrateSubscription(ctx, &result.Items[i])
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
			amount, _ := domain.MonthlyEquivalentWithInterval(item.AmountMinor, item.BillingCycle, item.BillingInterval)
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
	dates, dateErr := domain.BillingDatesWithInterval(v.StartsOn.In(location), v.BillingCycle, v.BillingInterval, v.NextBilling.In(location), 1200)
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
	s.audit(ctx, userID, v.GroupID, "subscription.stop_scheduled", "subscription", v.ID, "success", encodeAuditSummary(map[string]any{"ends_on": value.Format("2006-01-02")}, nil))
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
	s.audit(ctx, userID, v.GroupID, "subscription.stop_cancelled", "subscription", v.ID, "success", encodeAuditSummary(map[string]any{"name": v.Name}, nil))
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
			result.Items[i].CategoryInfo = s.hydrateCategory(ctx, result.Items[i].CategoryID)
			s.hydrateSubscription(ctx, &result.Items[i])
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
	if v.PaidBy == "" {
		v.PaidBy = current.PaidBy
	}
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
	v.BillingInterval = domain.NormalizeBillingInterval(v.BillingCycle, v.BillingInterval)
	if next, nextErr := domain.NextBillingWithInterval(v.StartsOn, v.BillingCycle, v.BillingInterval, s.Now().In(location)); nextErr == nil {
		v.NextBilling = next
	} else {
		return nil, nextErr
	}
	v.EndsOn = current.EndsOn
	if v.BaseCurrency == "" {
		v.BaseCurrency = current.BaseCurrency
	}
	if v.Currency == "" {
		v.Currency = current.Currency
	}
	if v.RateMode == "" {
		v.RateMode = current.RateMode
		if v.RateMode == "" {
			v.RateMode = domain.RateAutomatic
		}
	}
	if v.CategoryID == "" {
		v.CategoryID = current.CategoryID
	}
	if _, err = s.validateCategory(ctx, userID, v.GroupID, v.CategoryID); err != nil {
		return nil, err
	}
	baseAmount, rate, rateText, rateDate, conversionErr := s.conversion(ctx, v.Currency, v.BaseCurrency, v.AmountMinor, v.RateMode, v.ExchangeRate, v.StartsOn)
	if conversionErr != nil {
		return nil, conversionErr
	}
	v.BaseAmountMinor, v.RateScaled, v.ExchangeRate, v.ExchangeRateDate = baseAmount, rate, rateText, rateDate
	if !validSubscription(&v) {
		return nil, domain.ErrInvalid
	}
	if v.GroupID != "" {
		members, memberErr := s.memberIDs(ctx, v.GroupID)
		if memberErr != nil {
			return nil, memberErr
		}
		if len(v.Splits) == 0 {
			v.SplitMode, v.Splits = current.SplitMode, current.Splits
		}
		if len(v.Splits) == 0 {
			v.SplitMode, v.Splits = domain.SplitAmount, []domain.ExpenseSplit{{UserID: v.PaidBy, AmountMinor: v.AmountMinor}}
		}
		v.Splits, err = domain.CanonicalSplits(v.AmountMinor, v.PaidBy, v.SplitMode, v.Splits, members)
		if err != nil {
			return nil, err
		}
		for i := range v.Splits {
			v.Splits[i].BaseAmountMinor, err = domain.ConvertMinor(v.Splits[i].AmountMinor, v.Currency, v.BaseCurrency, v.RateScaled)
			if err != nil {
				return nil, err
			}
		}
		v.Splits = domain.CanonicalBaseSplits(v.BaseAmountMinor, v.PaidBy, v.Splits)
	}
	scope := v.RevisionScope
	if scope != "one_off" {
		scope = "future"
	}
	effective := v.EffectiveBillingAt
	historical := false
	if v.GroupID != "" {
		if effective.IsZero() {
			effective = current.NextBilling
		}
		historical = effective.Before(current.NextBilling)
		if historical {
			if err = s.groupPermission(ctx, userID, current.GroupID, "ledger.records.historical_write"); err != nil {
				s.audit(ctx, userID, current.GroupID, "subscription.updated", "subscription", current.ID, "failure", encodeAuditSummary(historicalSubscriptionDetails(current, effective), nil))
				return nil, err
			}
			// Reopening a closed period must not rewrite what the subscription
			// bills from now on, so a historical edit only ever revises that one
			// occurrence.
			scope = "one_off"
		}
		if !subscriptionBillingDateAllowed(*current, effective, location, historical) {
			return nil, domain.ErrInvalid
		}
		if scope == "one_off" && (v.BillingCycle != current.BillingCycle || v.BillingInterval != current.BillingInterval || !v.StartsOn.Equal(current.StartsOn) || v.Status != current.Status) {
			return nil, domain.ErrInvalid
		}
	}
	if err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		// A one-off revision intentionally leaves the subscription's current and
		// future defaults untouched. The revision resolver applies it only to its
		// exact, not-yet-posted billing occurrence.
		if v.GroupID == "" || scope == "future" {
			if updateErr := s.Stores.Subscriptions.Update(tx, &v); updateErr != nil {
				return updateErr
			}
		}
		if v.GroupID == "" {
			return nil
		}
		if occurrence, occurrenceErr := s.Stores.Subscriptions.GetOccurrence(tx, v.ID, effective); occurrenceErr == nil && occurrence.ExpenseID != "" {
			return domain.ErrConflict
		}
		revision := subscriptionRevision(v, scope, effective)
		return s.Stores.Subscriptions.CreateRevision(tx, &revision)
	}); err != nil {
		return nil, err
	}
	s.audit(ctx, userID, v.GroupID, "subscription.updated", "subscription", v.ID, "success", historicalSubscriptionChangeSummary(current, &v, effective))
	if v.GroupID != "" {
		s.audit(ctx, userID, v.GroupID, "subscription.version_created", "subscription", v.ID, "success", encodeAuditSummary(map[string]any{"scope": scope, "effective_billing_at": effective.Format("2006-01-02")}, nil))
	}
	return &v, nil
}
func subscriptionRevision(v domain.Subscription, scope string, effective time.Time) domain.SubscriptionRevision {
	return domain.SubscriptionRevision{SubscriptionID: v.ID, Scope: scope, EffectiveBillingAt: effective, Name: v.Name, Category: v.Category, CategoryID: v.CategoryID, AmountMinor: v.AmountMinor, Currency: v.Currency, BaseCurrency: v.BaseCurrency, BaseAmountMinor: v.BaseAmountMinor, ExchangeRate: v.ExchangeRate, RateScaled: v.RateScaled, ExchangeRateDate: v.ExchangeRateDate, RateMode: v.RateMode, PaidBy: v.PaidBy, SplitMode: v.SplitMode, Splits: append([]domain.ExpenseSplit(nil), v.Splits...), Notes: v.Notes}
}
func (s *Service) hydrateSubscription(ctx context.Context, v *domain.Subscription) {
	if v.GroupID == "" {
		return
	}
	values, err := s.Stores.Subscriptions.ListRevisions(ctx, v.ID)
	if err != nil {
		return
	}
	v.Revisions = values
	if occurrences, occurrenceErr := s.Stores.Subscriptions.ListOccurrences(ctx, v.ID); occurrenceErr == nil {
		v.Occurrences = occurrences
	}
	if revision, ok := subscriptionRevisionAt(values, v.NextBilling); ok {
		applySubscriptionRevision(v, revision)
	}
}
func applySubscriptionRevision(v *domain.Subscription, revision domain.SubscriptionRevision) {
	v.Name = revision.Name
	v.Category = revision.Category
	v.CategoryID = revision.CategoryID
	v.AmountMinor = revision.AmountMinor
	v.Currency = revision.Currency
	v.BaseCurrency = revision.BaseCurrency
	v.BaseAmountMinor = revision.BaseAmountMinor
	v.ExchangeRate = revision.ExchangeRate
	v.RateScaled = revision.RateScaled
	v.ExchangeRateDate = revision.ExchangeRateDate
	v.RateMode = revision.RateMode
	v.PaidBy = revision.PaidBy
	v.SplitMode = revision.SplitMode
	// Keep the subscription's own splits when a revision carries none, so a
	// revision without split data cannot silently drop the participants.
	if len(revision.Splits) > 0 {
		v.Splits = append([]domain.ExpenseSplit(nil), revision.Splits...)
	}
	v.Notes = revision.Notes
}

// allowPast is granted only to callers holding ledger.records.historical_write,
// letting them revise a billing date that has already gone by. The date must
// still land exactly on the subscription's schedule either way.
func subscriptionBillingDateAllowed(subscription domain.Subscription, effective time.Time, location *time.Location, allowPast bool) bool {
	if !allowPast && effective.Before(subscription.NextBilling) {
		return false
	}
	start, target := subscription.StartsOn.In(location), effective.In(location)
	for index := 0; index < 12000; index++ {
		value, err := domain.BillingDateWithInterval(start, subscription.BillingCycle, subscription.BillingInterval, index)
		if err != nil {
			return false
		}
		if value.Equal(target) {
			return true
		}
		if value.After(target) {
			return false
		}
	}
	return false
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
	err = s.Stores.Subscriptions.Delete(ctx, id)
	if err == nil {
		s.audit(ctx, userID, v.GroupID, "subscription.deleted", "subscription", id, "success", encodeAuditSummary(map[string]any{"name": v.Name, "amount_minor": v.AmountMinor}, nil))
	}
	return err
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
	v.CategoryInfo = s.hydrateCategory(ctx, v.CategoryID)
	return nil
}
func (s *Service) CreateExpense(ctx context.Context, userID string, v domain.Expense) (*domain.Expense, error) {
	if v.RateMode == "" {
		v.RateMode = domain.RateAutomatic
	}
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
		v.BaseCurrency = group.Currency
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
		v.BaseCurrency = v.Currency
		v.SplitMode = domain.SplitAmount
		v.Splits = []domain.ExpenseSplit{{UserID: userID, AmountMinor: v.AmountMinor}}
	}
	if _, err := s.validateCategory(ctx, userID, v.GroupID, v.CategoryID); err != nil {
		return nil, err
	}
	baseAmount, rate, rateText, rateDate, err := s.conversion(ctx, v.Currency, v.BaseCurrency, v.AmountMinor, v.RateMode, v.ExchangeRate, v.IncurredOn)
	if err != nil {
		return nil, err
	}
	v.BaseAmountMinor, v.RateScaled, v.ExchangeRate, v.ExchangeRateDate = baseAmount, rate, rateText, rateDate
	for i := range v.Splits {
		v.Splits[i].BaseAmountMinor, err = domain.ConvertMinor(v.Splits[i].AmountMinor, v.Currency, v.BaseCurrency, rate)
		if err != nil {
			return nil, err
		}
	}
	v.Splits = domain.CanonicalBaseSplits(v.BaseAmountMinor, v.PaidBy, v.Splits)
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
	s.audit(ctx, userID, v.GroupID, "expense.created", "expense", v.ID, "success", encodeAuditSummary(map[string]any{"title": v.Title, "amount_minor": v.AmountMinor, "incurred_on": v.IncurredOn.Format("2006-01-02"), "split_mode": v.SplitMode}, nil))
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
	historicalUpdate := false
	v.GroupID = current.GroupID
	v.OwnerID = current.OwnerID
	// Mirrors the same fallback UpdateSubscription applies to StartsOn: an
	// omitted date keeps the exact stored instant instead of forcing the
	// caller to resend one, which a timezone-lossy client could shift by a
	// day relative to what is actually stored.
	if v.IncurredOn.IsZero() {
		v.IncurredOn = current.IncurredOn
	}
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
		currentHistorical, historicalErr := s.expenseIsHistorical(ctx, userID, current)
		if historicalErr != nil {
			return nil, historicalErr
		}
		requestedHistorical := false
		if !v.IncurredOn.IsZero() {
			requestedHistorical, historicalErr = s.expenseIsHistorical(ctx, userID, &v)
			if historicalErr != nil {
				return nil, historicalErr
			}
		}
		historicalUpdate = currentHistorical || requestedHistorical
		if historicalUpdate {
			if err = s.groupPermission(ctx, userID, current.GroupID, "ledger.records.historical_write"); err != nil {
				s.audit(ctx, userID, current.GroupID, "expense.updated", "expense", current.ID, "failure", encodeAuditSummary(map[string]any{"current": historicalExpenseDetails(current), "requested": historicalExpenseDetails(&v)}, nil))
				return nil, err
			}
		}
		members, memberErr := s.memberIDs(ctx, current.GroupID)
		if memberErr != nil {
			return nil, memberErr
		}
		group, groupErr := s.Stores.Groups.Get(ctx, current.GroupID)
		if groupErr != nil {
			return nil, groupErr
		}
		v.BaseCurrency = group.Currency
		if len(v.Splits) == 0 {
			v.Splits = current.Splits
		}
		v.Splits, err = domain.CanonicalSplits(v.AmountMinor, v.PaidBy, v.SplitMode, v.Splits, members)
		if err != nil {
			return nil, err
		}
	}
	if v.Currency == "" {
		v.Currency = current.Currency
	}
	if v.BaseCurrency == "" {
		v.BaseCurrency = current.BaseCurrency
	}
	if v.RateMode == "" {
		v.RateMode = current.RateMode
		if v.RateMode == "" {
			v.RateMode = domain.RateAutomatic
		}
	}
	if v.CategoryID == "" {
		v.CategoryID = current.CategoryID
	}
	if _, err = s.validateCategory(ctx, userID, v.GroupID, v.CategoryID); err != nil {
		return nil, err
	}
	baseAmount, rate, rateText, rateDate, conversionErr := s.conversion(ctx, v.Currency, v.BaseCurrency, v.AmountMinor, v.RateMode, v.ExchangeRate, v.IncurredOn)
	if conversionErr != nil {
		return nil, conversionErr
	}
	v.BaseAmountMinor, v.RateScaled, v.ExchangeRate, v.ExchangeRateDate = baseAmount, rate, rateText, rateDate
	for i := range v.Splits {
		v.Splits[i].BaseAmountMinor, err = domain.ConvertMinor(v.Splits[i].AmountMinor, v.Currency, v.BaseCurrency, rate)
		if err != nil {
			return nil, err
		}
	}
	v.Splits = domain.CanonicalBaseSplits(v.BaseAmountMinor, v.PaidBy, v.Splits)
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
	s.audit(ctx, userID, v.GroupID, "expense.updated", "expense", v.ID, "success", historicalExpenseChangeSummary(current, &v))
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
	historical, historicalErr := s.expenseIsHistorical(ctx, userID, v)
	if historicalErr != nil {
		return historicalErr
	}
	if historical {
		if err = s.groupPermission(ctx, userID, v.GroupID, "ledger.records.historical_write"); err != nil {
			s.audit(ctx, userID, v.GroupID, "expense.deleted", "expense", id, "failure", encodeAuditSummary(historicalExpenseDetails(v), nil))
			return err
		}
	}
	err = s.Stores.Expenses.Delete(ctx, id)
	if err == nil {
		s.audit(ctx, userID, v.GroupID, "expense.deleted", "expense", id, "success", encodeAuditSummary(historicalExpenseDetails(v), nil))
	}
	return err
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
			m, _ := domain.MonthlyEquivalentWithInterval(v.AmountMinor, v.BillingCycle, v.BillingInterval)
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
