package application

import (
	"context"
	"errors"
	"log/slog"
	"subflow/internal/domain"
	"subflow/internal/ports"
)

func readAllRecords[T any](fetch func(ports.PageRequest) (ports.Page[T], error), sort string) (ports.Page[T], error) {
	result := ports.Page[T]{}
	for page := 1; ; page++ {
		values, err := fetch(ports.PageRequest{Page: page, PerPage: 100, Sort: sort + ",id"})
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, values.Items...)
		if page >= values.TotalPages || len(values.Items) == 0 {
			return result, nil
		}
	}
}

func (s *Service) CurrencyRequestFailure(ctx context.Context, actor, group, action string, cause error) error {
	if err := s.audit(ctx, actor, group, action, "group", group, "failure", encodeAuditSummary(map[string]any{"reason": "currency_operation_failed"}, nil)); err != nil {
		slog.Error("currency failure audit could not be persisted", "action", action, "audit_error", err)
		return errors.Join(domain.ErrAuditUnavailable, err, cause)
	}
	return cause
}

func (s *Service) PreviewGroupCurrency(ctx context.Context, actor, group string, target domain.Currency) (*domain.CurrencyChangePreview, error) {
	value, err := s.previewGroupCurrency(ctx, actor, group, target)
	if err == nil {
		err = s.audit(ctx, actor, group, "group.currency_previewed", "group", group, "success", encodeAuditSummary(map[string]any{"target": target, "affected": value.Affected, "missing": len(value.Missing)}, nil))
	}
	if err != nil {
		return nil, s.CurrencyRequestFailure(ctx, actor, group, "group.currency_previewed", err)
	}
	return value, nil
}

func (s *Service) ChangeGroupCurrency(ctx context.Context, actor, group string, target domain.Currency) (*domain.Group, error) {
	var result *domain.Group
	err := s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		var err error
		result, err = s.changeGroupCurrency(tx, actor, group, target)
		if err != nil {
			return err
		}
		return s.audit(tx, actor, group, "group.currency_changed", "group", group, "success", encodeAuditSummary(map[string]any{"currency": target}, nil))
	})
	if err != nil {
		return nil, s.CurrencyRequestFailure(ctx, actor, group, "group.currency_changed", err)
	}
	return result, nil
}
