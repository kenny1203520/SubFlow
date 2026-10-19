package application

import (
	"context"
	"strings"
	"subflow/internal/domain"
	"subflow/internal/ports"
	"time"
)

func incomeAuditSummary(v *domain.Income) string {
	return encodeAuditSummary(map[string]any{"title": v.Title, "amount_minor": v.AmountMinor, "currency": string(v.Currency), "received_on": v.ReceivedOn.Format("2006-01-02")}, nil)
}

func (s *Service) preparePersonalIncome(ctx context.Context, actor string, v *domain.Income) error {
	u, err := s.Stores.Users.Get(ctx, actor)
	if err != nil {
		return err
	}
	v.GroupID, v.OwnerID, v.EarnedBy = "", actor, actor
	v.Title = strings.TrimSpace(v.Title)
	if v.Currency == "" {
		v.Currency = u.DefaultCurrency
	}
	if v.Currency == "" {
		v.Currency = domain.CurrencyTWD
	}
	// Personal accounting currency belongs to the account, never the request.
	v.BaseCurrency = u.DefaultCurrency
	if v.BaseCurrency == "" {
		v.BaseCurrency = v.Currency
	}
	if v.RateMode == "" {
		v.RateMode = domain.RateAutomatic
	}
	if v.ReceivedOn.IsZero() {
		v.ReceivedOn = s.Now().UTC()
	}
	c, err := s.validateCategory(ctx, actor, "", v.CategoryID)
	if err != nil {
		return err
	}
	if c != nil && v.Category == "" {
		v.Category = c.CustomName
	}
	if v.Title == "" || v.AmountMinor < 0 || !domain.IsCurrency(v.Currency) || !domain.IsCurrency(v.BaseCurrency) {
		return domain.ErrInvalid
	}
	b, r, rt, rd, err := s.conversion(ctx, v.Currency, v.BaseCurrency, v.AmountMinor, v.RateMode, v.ExchangeRate, v.ReceivedOn)
	if err != nil {
		return err
	}
	v.BaseAmountMinor, v.RateScaled, v.ExchangeRate, v.ExchangeRateDate = b, r, rt, rd
	v.SplitMode = domain.SplitAmount
	v.Splits = []*domain.IncomeSplit{{BaseSplit: domain.BaseSplit{UserID: actor, AmountMinor: v.AmountMinor, BaseAmountMinor: v.BaseAmountMinor}}}
	return nil
}

func (s *Service) CreateIncome(ctx context.Context, actor string, v domain.Income) (*domain.Income, error) {
	if err := s.preparePersonalIncome(ctx, actor, &v); err != nil {
		return nil, s.groupIncomeFailure(ctx, actor, "", "income.created", "", "invalid_input", err)
	}
	if err := s.persistIncome(ctx, actor, &v, true); err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *Service) ListPersonalIncomes(ctx context.Context, actor string, p ports.PageRequest) (ports.Page[domain.Income], error) {
	r, err := s.Stores.Incomes.ListPersonal(ctx, actor, p)
	if err != nil {
		return r, err
	}
	for i := range r.Items {
		r.Items[i].CategoryInfo = s.hydrateCategory(ctx, r.Items[i].CategoryID)
	}
	return r, nil
}

func (s *Service) UpdateIncome(ctx context.Context, actor string, v domain.Income) (*domain.Income, error) {
	current, err := s.Stores.Incomes.Get(ctx, v.ID)
	if err != nil {
		return nil, s.groupIncomeFailure(ctx, actor, "", "income.updated", v.ID, "not_found", err)
	}
	if current.GroupID != "" || current.OwnerID != actor {
		return nil, s.groupIncomeFailure(ctx, actor, "", "income.updated", v.ID, "forbidden", domain.ErrForbidden)
	}
	if v.ReceivedOn.IsZero() {
		v.ReceivedOn = current.ReceivedOn
	}
	if v.Currency == "" {
		v.Currency = current.Currency
	}
	if v.RateMode == "" {
		v.RateMode = current.RateMode
	}
	if err = s.preparePersonalIncome(ctx, actor, &v); err != nil {
		return nil, s.groupIncomeFailure(ctx, actor, "", "income.updated", v.ID, "invalid_input", err)
	}
	if err = s.persistIncome(ctx, actor, &v, false); err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *Service) DeleteIncome(ctx context.Context, actor, id string) error {
	v, err := s.Stores.Incomes.Get(ctx, id)
	if err != nil {
		return s.groupIncomeFailure(ctx, actor, "", "income.deleted", id, "not_found", err)
	}
	if v.GroupID != "" || v.OwnerID != actor {
		return s.groupIncomeFailure(ctx, actor, "", "income.deleted", id, "forbidden", domain.ErrForbidden)
	}
	if _, err = s.Stores.Attachments.List(ctx, "income", id, time.Time{}); err != nil {
		return s.groupIncomeFailure(ctx, actor, "", "income.deleted", id, "attachments_failed", err)
	}
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if err := s.Stores.Attachments.DeleteTarget(tx, "income", id); err != nil {
			return err
		}
		if err := s.Stores.Incomes.Delete(tx, id); err != nil {
			return err
		}
		return s.audit(tx, actor, "", "income.deleted", "income", id, "success", incomeAuditSummary(v))
	})
	if err != nil {
		return s.groupIncomeFailure(ctx, actor, "", "income.deleted", id, "delete_failed", err)
	}
	return nil
}
