package application

import (
	"context"
	"subflow/internal/domain"
	"time"
)

// IncomePatch keeps omission distinct from explicit zero/empty values.
// Ownership and accounting-derived values are never accepted from a client.
type IncomePatch struct {
	Title        *string                `json:"title"`
	Category     *string                `json:"category"`
	CategoryID   *string                `json:"categoryId"`
	AmountMinor  *int64                 `json:"amountMinor"`
	Currency     *domain.Currency       `json:"currency"`
	RateMode     *domain.RateMode       `json:"rateMode"`
	ExchangeRate *string                `json:"exchangeRate"`
	EarnedBy     *string                `json:"earnedBy"`
	ReceivedOn   *time.Time             `json:"receivedOn"`
	Notes        *string                `json:"notes"`
	SplitMode    *domain.SplitMode      `json:"splitMode"`
	Splits       *[]*domain.IncomeSplit `json:"splits"`
}

func (p IncomePatch) Apply(v domain.Income) domain.Income {
	if p.Title != nil {
		v.Title = *p.Title
	}
	if p.Category != nil {
		v.Category = *p.Category
	}
	if p.CategoryID != nil {
		v.CategoryID = *p.CategoryID
		if v.CategoryID == "" {
			v.Category = ""
		}
	}
	if p.AmountMinor != nil {
		v.AmountMinor = *p.AmountMinor
	}
	if p.Currency != nil {
		v.Currency = *p.Currency
	}
	if p.RateMode != nil {
		v.RateMode = *p.RateMode
	}
	if p.ExchangeRate != nil {
		v.ExchangeRate = *p.ExchangeRate
	}
	if p.EarnedBy != nil {
		v.EarnedBy = *p.EarnedBy
	}
	if p.ReceivedOn != nil {
		v.ReceivedOn = *p.ReceivedOn
	}
	if p.Notes != nil {
		v.Notes = *p.Notes
	}
	if p.SplitMode != nil {
		v.SplitMode = *p.SplitMode
	}
	if p.Splits != nil {
		v.Splits = *p.Splits
	}
	return v
}

func (s *Service) IncomeRequestFailure(ctx context.Context, actor, group, action, id string, cause error) error {
	return s.groupIncomeFailure(ctx, actor, group, action, id, "request_failed", cause)
}

func (s *Service) PatchIncome(ctx context.Context, actor, routeGroup, id string, patch IncomePatch) (*domain.Income, error) {
	v, err := s.Stores.Incomes.Get(ctx, id)
	if err != nil {
		return nil, s.IncomeRequestFailure(ctx, actor, routeGroup, "income.updated", id, err)
	}
	if routeGroup != "" && routeGroup != v.GroupID {
		return nil, s.IncomeRequestFailure(ctx, actor, routeGroup, "income.updated", id, domain.ErrForbidden)
	}
	if patch.ReceivedOn != nil && patch.ReceivedOn.IsZero() || patch.Title != nil && *patch.Title == "" || patch.Currency != nil && *patch.Currency == "" || patch.EarnedBy != nil && *patch.EarnedBy == "" || patch.Splits != nil && len(*patch.Splits) == 0 && v.GroupID != "" {
		return nil, s.IncomeRequestFailure(ctx, actor, v.GroupID, "income.updated", id, domain.ErrInvalid)
	}
	merged := patch.Apply(*v)
	if v.GroupID != "" {
		return s.UpdateGroupIncome(ctx, actor, merged)
	}
	return s.UpdateIncome(ctx, actor, merged)
}

func (s *Service) DeleteIncomeInScope(ctx context.Context, actor, routeGroup, id string) error {
	v, err := s.Stores.Incomes.Get(ctx, id)
	if err != nil {
		return s.IncomeRequestFailure(ctx, actor, routeGroup, "income.deleted", id, err)
	}
	if routeGroup != "" && routeGroup != v.GroupID {
		return s.IncomeRequestFailure(ctx, actor, routeGroup, "income.deleted", id, domain.ErrForbidden)
	}
	if v.GroupID != "" {
		return s.DeleteGroupIncome(ctx, actor, id)
	}
	return s.DeleteIncome(ctx, actor, id)
}

func (s *Service) incomeHistoryPermission(ctx context.Context, actor string, values ...*domain.Income) error {
	for _, v := range values {
		if v == nil || v.GroupID == "" {
			continue
		}
		start, _, err := s.monthRange(ctx, actor, v.GroupID, "")
		if err != nil {
			return err
		}
		if v.ReceivedOn.Before(start) {
			if err = s.groupPermission(ctx, actor, v.GroupID, "ledger.records.historical_write"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) persistIncome(ctx context.Context, actor string, v *domain.Income, create bool) error {
	action := "income.updated"
	if create {
		action = "income.created"
	}
	err := s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		var err error
		summary := incomeAuditSummary(v)
		if create {
			err = s.Stores.Incomes.Create(tx, v)
		} else {
			before, readErr := s.Stores.Incomes.Get(tx, v.ID)
			if readErr != nil {
				return readErr
			}
			var changes changeSet
			changes.addString("title", before.Title, v.Title)
			changes.addInt64("amount_minor", before.AmountMinor, v.AmountMinor)
			changes.addString("currency", string(before.Currency), string(v.Currency))
			changes.addString("received_on", before.ReceivedOn.Format(time.RFC3339Nano), v.ReceivedOn.Format(time.RFC3339Nano))
			changes.addString("earned_by", before.EarnedBy, v.EarnedBy)
			changes.addString("category_id", before.CategoryID, v.CategoryID)
			changes.addString("split_mode", string(before.SplitMode), string(v.SplitMode))
			changes.addAny("splits", before.Splits, v.Splits)
			summary = encodeAuditSummary(map[string]any{"currency": string(v.Currency), "notes_changed": before.Notes != v.Notes}, changes)
			err = s.Stores.Incomes.Update(tx, v)
		}
		if err != nil {
			return err
		}
		if err = s.Stores.Incomes.ReplaceSplits(tx, v.ID, v.Splits); err != nil {
			return err
		}
		return s.audit(tx, actor, v.GroupID, action, "income", v.ID, "success", summary)
	})
	if err != nil {
		return s.groupIncomeFailure(ctx, actor, v.GroupID, action, v.ID, "write_failed", err)
	}
	return nil
}
