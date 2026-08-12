package application

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

type RateProvider interface {
	Quote(context.Context, domain.Currency, domain.Currency, time.Time) (*domain.ExchangeRate, error)
	// QuoteAll returns every quote currency the provider knows for a single
	// base currency in one call, letting RefreshReferenceRates pre-warm broad
	// cache coverage without one HTTP round trip per currency pair.
	QuoteAll(context.Context, domain.Currency, time.Time) (map[domain.Currency]*domain.ExchangeRate, error)
}

func (s *Service) Currencies() []domain.CurrencyInfo { return domain.ActiveCurrencies() }

func (s *Service) QuoteRate(ctx context.Context, from, to domain.Currency, date time.Time) (*domain.ExchangeRate, error) {
	if !domain.IsCurrency(from) || !domain.IsCurrency(to) || date.IsZero() {
		return nil, domain.ErrInvalid
	}
	if from == to {
		return &domain.ExchangeRate{BaseCurrency: from, QuoteCurrency: to, RateScaled: domain.ExchangeRateScale, Rate: "1", EffectiveDate: date, Provider: "identity", FetchedAt: s.Now()}, nil
	}
	cached, cachedErr := s.Stores.ExchangeRates.LatestOnOrBefore(ctx, from, to, date)
	if cachedErr == nil && cached.EffectiveDate.Format("2006-01-02") == date.Format("2006-01-02") {
		rate := cached
		rate.Rate = domain.FormatRate(rate.RateScaled)
		return rate, nil
	}
	if s.Rates == nil {
		if cachedErr == nil {
			cached.Rate = domain.FormatRate(cached.RateScaled)
			cached.Stale = true
			return cached, nil
		}
		return nil, domain.ErrRateUnavailable
	}
	rate, err := s.Rates.Quote(ctx, from, to, date)
	if err != nil {
		if cachedErr == nil {
			cached.Rate = domain.FormatRate(cached.RateScaled)
			cached.Stale = true
			return cached, nil
		}
		return nil, err
	}
	if err = s.Stores.ExchangeRates.Upsert(ctx, rate); err != nil {
		return nil, err
	}
	rate.Rate = domain.FormatRate(rate.RateScaled)
	return rate, nil
}

// referenceRateBases are pre-warmed as base currencies each refresh cycle.
// Because the upstream feed returns every quote currency for a single base
// in one call (see OpenERAPIProvider.QuoteAll), warming a handful of common
// bases yields broad, largely bidirectional cache coverage (TWD-quoted,
// USD-quoted, etc.) instead of only the 4 fixed TWD pairs this used to warm,
// without making one HTTP call per currency pair.
var referenceRateBases = []domain.Currency{domain.CurrencyUSD, domain.CurrencyEUR, domain.CurrencyJPY, "GBP", domain.CurrencyTWD}

func (s *Service) RefreshReferenceRates(ctx context.Context) error {
	if s.Rates == nil {
		return nil
	}
	today := s.Now().UTC()
	for _, base := range referenceRateBases {
		rates, err := s.Rates.QuoteAll(ctx, base, today)
		if err != nil {
			if errors.Is(err, domain.ErrRateUnavailable) {
				s.audit(ctx, "", "", "exchange_rate.refresh_failed", "exchange_rate", string(base), "failure", encodeAuditSummary(map[string]any{"base": base}, nil))
				continue
			}
			return err
		}
		for _, rate := range rates {
			if err = s.Stores.ExchangeRates.Upsert(ctx, rate); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) RefreshAutomaticSubscriptions(ctx context.Context) error {
	values, err := s.Stores.Subscriptions.ListAutomatic(ctx)
	if err != nil {
		return err
	}
	for i := range values {
		v := &values[i]
		baseAmount, rate, text, effective, quoteErr := s.conversion(ctx, v.Currency, v.BaseCurrency, v.AmountMinor, domain.RateAutomatic, "", s.Now())
		if quoteErr != nil {
			continue
		}
		v.BaseAmountMinor, v.RateScaled, v.ExchangeRate, v.ExchangeRateDate = baseAmount, rate, text, effective
		if updateErr := s.Stores.Subscriptions.Update(ctx, v); updateErr != nil {
			return updateErr
		}
	}
	return nil
}

// PostDueSubscriptions turns every due group subscription into the immutable
// expense that represents that billing occurrence.  The occurrence record is
// created in the same transaction as the expense and has a unique
// (subscription, billing_at) index, so repeated scheduler runs are safe.
func (s *Service) PostDueSubscriptions(ctx context.Context) error {
	values, err := s.Stores.Subscriptions.ListDue(ctx, s.Now())
	if err != nil {
		return err
	}
	for i := range values {
		if err := s.postSubscriptionOccurrence(ctx, &values[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) postSubscriptionOccurrence(ctx context.Context, subscription *domain.Subscription) error {
	if subscription.GroupID == "" || domain.SubscriptionLifecycle(*subscription, s.Now()) == "ended" {
		return nil
	}
	revisions, err := s.Stores.Subscriptions.ListRevisions(ctx, subscription.ID)
	if err != nil {
		return err
	}
	revision, ok := subscriptionRevisionAt(revisions, subscription.NextBilling)
	if !ok {
		return s.recordSubscriptionOccurrenceFailure(ctx, subscription, "subscription_version_missing")
	}
	members, err := s.memberIDs(ctx, subscription.GroupID)
	if err != nil {
		return err
	}
	splits, err := domain.CanonicalSplits(revision.AmountMinor, revision.PaidBy, revision.SplitMode, revision.Splits, members)
	if err != nil {
		return s.recordSubscriptionOccurrenceFailure(ctx, subscription, "subscription_split_invalid")
	}
	for i := range splits {
		splits[i].BaseAmountMinor, err = domain.ConvertMinor(splits[i].AmountMinor, revision.Currency, revision.BaseCurrency, revision.RateScaled)
		if err != nil {
			return s.recordSubscriptionOccurrenceFailure(ctx, subscription, "subscription_conversion_invalid")
		}
	}
	splits = domain.CanonicalBaseSplits(revision.BaseAmountMinor, revision.PaidBy, splits)
	billingAt := subscription.NextBilling
	return s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if _, findErr := s.Stores.Subscriptions.GetOccurrence(tx, subscription.ID, billingAt); findErr == nil {
			return nil
		} else if !errors.Is(findErr, domain.ErrNotFound) {
			return findErr
		}
		expense := domain.Expense{
			GroupID: subscription.GroupID, SubscriptionID: subscription.ID,
			Title: revision.Name, Category: revision.Category, CategoryID: revision.CategoryID,
			AmountMinor: revision.AmountMinor, Currency: revision.Currency,
			BaseCurrency: revision.BaseCurrency, BaseAmountMinor: revision.BaseAmountMinor,
			RateScaled: revision.RateScaled, ExchangeRate: revision.ExchangeRate,
			ExchangeRateDate: revision.ExchangeRateDate, RateMode: revision.RateMode,
			PaidBy: revision.PaidBy, IncurredOn: billingAt, SplitMode: revision.SplitMode,
			Notes: revision.Notes, Splits: splits,
		}
		if createErr := s.Stores.Expenses.Create(tx, &expense); createErr != nil {
			return createErr
		}
		if splitErr := s.Stores.Expenses.ReplaceSplits(tx, expense.ID, splits); splitErr != nil {
			return splitErr
		}
		occurrence := domain.SubscriptionOccurrence{SubscriptionID: subscription.ID, RevisionID: revision.ID, ExpenseID: expense.ID, BillingAt: billingAt, Status: "posted"}
		if createErr := s.Stores.Subscriptions.CreateOccurrence(tx, &occurrence); createErr != nil {
			return createErr
		}
		next, nextErr := domain.NextBillingWithInterval(subscription.StartsOn, subscription.BillingCycle, subscription.BillingInterval, billingAt.Add(time.Nanosecond))
		if nextErr != nil {
			return nextErr
		}
		subscription.NextBilling = next
		if updateErr := s.Stores.Subscriptions.Update(tx, subscription); updateErr != nil {
			return updateErr
		}
		s.audit(tx, "", subscription.GroupID, "subscription.occurrence_posted", "subscription", subscription.ID, "success", encodeAuditSummary(map[string]any{"billing_at": billingAt.Format("2006-01-02"), "amount_minor": expense.AmountMinor, "expense_id": expense.ID}, nil))
		return nil
	})
}

func subscriptionRevisionAt(values []domain.SubscriptionRevision, billingAt time.Time) (domain.SubscriptionRevision, bool) {
	var selected domain.SubscriptionRevision
	found := false
	for _, value := range values {
		if value.EffectiveBillingAt.After(billingAt) {
			continue
		}
		if value.Scope == "one_off" && !value.EffectiveBillingAt.Equal(billingAt) {
			continue
		}
		if !found || value.EffectiveBillingAt.After(selected.EffectiveBillingAt) || (value.EffectiveBillingAt.Equal(selected.EffectiveBillingAt) && value.Scope == "one_off") {
			selected, found = value, true
		}
	}
	return selected, found
}

func (s *Service) recordSubscriptionOccurrenceFailure(ctx context.Context, subscription *domain.Subscription, reason string) error {
	occurrence := domain.SubscriptionOccurrence{SubscriptionID: subscription.ID, BillingAt: subscription.NextBilling, Status: "failed", Error: reason}
	if err := s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if _, findErr := s.Stores.Subscriptions.GetOccurrence(tx, subscription.ID, subscription.NextBilling); findErr == nil {
			return nil
		} else if !errors.Is(findErr, domain.ErrNotFound) {
			return findErr
		}
		// A failed record has no revision relation. Persist a copy of the latest
		// version so the failure remains diagnosable without mutating history.
		revisions, listErr := s.Stores.Subscriptions.ListRevisions(tx, subscription.ID)
		if listErr != nil {
			return listErr
		}
		if revision, ok := subscriptionRevisionAt(revisions, subscription.NextBilling); ok {
			occurrence.RevisionID = revision.ID
		}
		if occurrence.RevisionID == "" {
			return domain.ErrInvalid
		}
		return s.Stores.Subscriptions.CreateOccurrence(tx, &occurrence)
	}); err != nil {
		return err
	}
	s.audit(ctx, "", subscription.GroupID, "subscription.occurrence_failed", "subscription", subscription.ID, "failure", encodeAuditSummary(map[string]any{"billing_at": subscription.NextBilling.Format("2006-01-02"), "reason": reason}, nil))
	return nil
}

func (s *Service) conversion(ctx context.Context, currency, base domain.Currency, amount int64, mode domain.RateMode, supplied string, date time.Time) (int64, int64, string, time.Time, error) {
	if base == "" {
		base = currency
	}
	if currency == base {
		return amount, domain.ExchangeRateScale, "1", date, nil
	}
	if mode == "" {
		mode = domain.RateAutomatic
	}
	if mode == domain.RateManual {
		rate, err := domain.ParseRate(supplied)
		if err != nil {
			return 0, 0, "", time.Time{}, err
		}
		converted, err := domain.ConvertMinor(amount, currency, base, rate)
		return converted, rate, domain.FormatRate(rate), date, err
	}
	quote, err := s.QuoteRate(ctx, currency, base, date)
	if err != nil {
		return 0, 0, "", time.Time{}, err
	}
	converted, err := domain.ConvertMinor(amount, currency, base, quote.RateScaled)
	return converted, quote.RateScaled, domain.FormatRate(quote.RateScaled), quote.EffectiveDate, err
}

func (s *Service) ListCategories(ctx context.Context, userID, scope, groupID string, archived bool) ([]domain.Category, error) {
	if scope != "personal" && scope != "group" {
		return nil, domain.ErrInvalid
	}
	if scope == "group" {
		if groupID == "" || s.role(ctx, groupID, userID, false) != nil {
			return nil, domain.ErrForbidden
		}
	} else {
		groupID = ""
	}
	return s.Stores.Categories.List(ctx, userID, groupID, archived)
}

func (s *Service) CreateCategory(ctx context.Context, userID string, value domain.Category) (*domain.Category, error) {
	value.CustomName = strings.TrimSpace(value.CustomName)
	value.SystemKey = ""
	value.CreatedBy = userID
	if value.CustomName == "" || (value.Scope != "personal" && value.Scope != "group") {
		return nil, domain.ErrInvalid
	}
	if value.Scope == "personal" {
		value.OwnerID, value.GroupID = userID, ""
	} else {
		value.OwnerID = ""
		if value.GroupID == "" || s.role(ctx, value.GroupID, userID, false) != nil {
			return nil, domain.ErrForbidden
		}
	}
	values, err := s.Stores.Categories.List(ctx, userID, value.GroupID, true)
	if err != nil {
		return nil, err
	}
	for _, current := range values {
		if current.Scope == value.Scope && strings.EqualFold(strings.TrimSpace(current.CustomName), value.CustomName) {
			return nil, domain.ErrConflict
		}
	}
	if err = s.Stores.Categories.Create(ctx, &value); err != nil {
		return nil, err
	}
	s.audit(ctx, userID, value.GroupID, "category.created", "category", value.ID, "success", encodeAuditSummary(map[string]any{"name": value.CustomName, "scope": value.Scope, "icon": value.IconKey}, nil))
	return &value, nil
}

func (s *Service) UpdateCategory(ctx context.Context, userID string, value domain.Category) (*domain.Category, error) {
	current, err := s.Stores.Categories.Get(ctx, value.ID)
	if err != nil {
		return nil, err
	}
	if current.Scope == "system" {
		return nil, domain.ErrForbidden
	}
	if current.Scope == "personal" {
		if current.OwnerID != userID {
			return nil, domain.ErrForbidden
		}
	} else {
		group, groupErr := s.Stores.Groups.Get(ctx, current.GroupID)
		if groupErr != nil {
			return nil, groupErr
		}
		if current.CreatedBy != userID && group.OwnerID != userID {
			return nil, domain.ErrForbidden
		}
	}
	before := *current
	if name := strings.TrimSpace(value.CustomName); name != "" {
		current.CustomName = name
	}
	current.Archived = value.Archived
	if value.IconKey != "" {
		current.IconKey = value.IconKey
	}
	if err = s.Stores.Categories.Update(ctx, current); err != nil {
		return nil, err
	}
	var categoryChanges changeSet
	categoryChanges.addString("name", before.CustomName, current.CustomName)
	categoryChanges.addBool("archived", before.Archived, current.Archived)
	categoryChanges.addString("icon", before.IconKey, current.IconKey)
	s.audit(ctx, userID, current.GroupID, "category.updated", "category", current.ID, "success", encodeAuditSummary(nil, categoryChanges))
	return current, nil
}

func (s *Service) validateCategory(ctx context.Context, userID, groupID, id string) (*domain.Category, error) {
	if id == "" {
		return nil, nil
	}
	category, err := s.Stores.Categories.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if category.Archived {
		return nil, domain.ErrInvalid
	}
	if category.Scope == "system" {
		return category, nil
	}
	if groupID == "" && category.Scope == "personal" && category.OwnerID == userID {
		return category, nil
	}
	if groupID != "" && category.Scope == "group" && category.GroupID == groupID {
		return category, nil
	}
	return nil, domain.ErrForbidden
}

func (s *Service) hydrateCategory(ctx context.Context, id string) *domain.Category {
	if id == "" {
		return nil
	}
	value, _ := s.Stores.Categories.Get(ctx, id)
	return value
}

func (s *Service) PreviewGroupCurrency(ctx context.Context, userID, groupID string, target domain.Currency) (*domain.CurrencyChangePreview, error) {
	if err := s.role(ctx, groupID, userID, true); err != nil {
		return nil, err
	}
	group, err := s.Stores.Groups.Get(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if !domain.IsCurrency(target) {
		return nil, domain.ErrInvalid
	}
	preview := &domain.CurrencyChangePreview{From: group.Currency, To: target}
	expenses, err := s.Stores.Expenses.List(ctx, groupID, pageAll("incurred_on"))
	if err != nil {
		return nil, err
	}
	subs, err := s.Stores.Subscriptions.List(ctx, groupID, pageAll("next_billing"))
	if err != nil {
		return nil, err
	}
	settlements, err := s.Stores.Settlements.List(ctx, groupID, pageAll("settled_on"))
	if err != nil {
		return nil, err
	}
	preview.Affected = len(expenses.Items) + len(subs.Items) + len(settlements.Items)
	check := func(resource, id string, from domain.Currency, date time.Time) {
		if from == target {
			return
		}
		if _, quoteErr := s.QuoteRate(ctx, from, target, date); quoteErr != nil {
			preview.Missing = append(preview.Missing, domain.CurrencyChangeMissing{Resource: resource, ID: id, From: from, To: target, Date: date.Format("2006-01-02")})
		}
	}
	for _, v := range expenses.Items {
		check("expense", v.ID, v.Currency, v.IncurredOn)
	}
	for _, v := range subs.Items {
		check("subscription", v.ID, v.Currency, v.StartsOn)
	}
	for _, v := range settlements.Items {
		check("settlement", v.ID, v.Currency, v.SettledOn)
	}
	sort.Slice(preview.Missing, func(i, j int) bool { return preview.Missing[i].Date < preview.Missing[j].Date })
	return preview, nil
}

func (s *Service) ChangeGroupCurrency(ctx context.Context, userID, groupID string, target domain.Currency) (*domain.Group, error) {
	preview, err := s.PreviewGroupCurrency(ctx, userID, groupID, target)
	if err != nil {
		return nil, err
	}
	if len(preview.Missing) > 0 {
		return nil, domain.ErrRateUnavailable
	}
	group, err := s.Stores.Groups.Get(ctx, groupID)
	if err != nil {
		return nil, err
	}
	expenses, err := s.Stores.Expenses.List(ctx, groupID, pageAll("incurred_on"))
	if err != nil {
		return nil, err
	}
	subs, err := s.Stores.Subscriptions.List(ctx, groupID, pageAll("next_billing"))
	if err != nil {
		return nil, err
	}
	settlements, err := s.Stores.Settlements.List(ctx, groupID, pageAll("settled_on"))
	if err != nil {
		return nil, err
	}
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		for i := range expenses.Items {
			v := &expenses.Items[i]
			if hydrateErr := s.hydrateExpense(tx, v); hydrateErr != nil {
				return hydrateErr
			}
			quote, quoteErr := s.QuoteRate(tx, v.Currency, target, v.IncurredOn)
			if quoteErr != nil {
				return quoteErr
			}
			v.BaseCurrency, v.RateScaled, v.ExchangeRate, v.ExchangeRateDate = target, quote.RateScaled, domain.FormatRate(quote.RateScaled), quote.EffectiveDate
			v.BaseAmountMinor, quoteErr = domain.ConvertMinor(v.AmountMinor, v.Currency, target, quote.RateScaled)
			if quoteErr != nil {
				return quoteErr
			}
			for j := range v.Splits {
				v.Splits[j].BaseAmountMinor, quoteErr = domain.ConvertMinor(v.Splits[j].AmountMinor, v.Currency, target, quote.RateScaled)
				if quoteErr != nil {
					return quoteErr
				}
			}
			v.Splits = domain.CanonicalBaseSplits(v.BaseAmountMinor, v.PaidBy, v.Splits)
			if quoteErr = s.Stores.Expenses.Update(tx, v); quoteErr != nil {
				return quoteErr
			}
			if quoteErr = s.Stores.Expenses.ReplaceSplits(tx, v.ID, v.Splits); quoteErr != nil {
				return quoteErr
			}
		}
		for i := range subs.Items {
			v := &subs.Items[i]
			quote, quoteErr := s.QuoteRate(tx, v.Currency, target, v.StartsOn)
			if quoteErr != nil {
				return quoteErr
			}
			v.BaseCurrency, v.RateScaled, v.ExchangeRate, v.ExchangeRateDate = target, quote.RateScaled, domain.FormatRate(quote.RateScaled), quote.EffectiveDate
			v.BaseAmountMinor, quoteErr = domain.ConvertMinor(v.AmountMinor, v.Currency, target, quote.RateScaled)
			if quoteErr != nil {
				return quoteErr
			}
			if quoteErr = s.Stores.Subscriptions.Update(tx, v); quoteErr != nil {
				return quoteErr
			}
		}
		for i := range settlements.Items {
			v := &settlements.Items[i]
			quote, quoteErr := s.QuoteRate(tx, v.Currency, target, v.SettledOn)
			if quoteErr != nil {
				return quoteErr
			}
			v.BaseCurrency, v.RateScaled, v.ExchangeRate, v.ExchangeRateDate = target, quote.RateScaled, domain.FormatRate(quote.RateScaled), quote.EffectiveDate
			v.BaseAmountMinor, quoteErr = domain.ConvertMinor(v.AmountMinor, v.Currency, target, quote.RateScaled)
			if quoteErr != nil {
				return quoteErr
			}
			if quoteErr = s.Stores.Settlements.Update(tx, v); quoteErr != nil {
				return quoteErr
			}
		}
		group.Currency = target
		return s.Stores.Groups.Update(tx, group)
	})
	if err != nil {
		return nil, err
	}
	return group, nil
}

func pageAll(sort string) ports.PageRequest {
	return ports.PageRequest{Page: 1, PerPage: 10000, Sort: sort}
}
