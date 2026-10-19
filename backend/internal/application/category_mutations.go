package application

import (
	"context"
	"errors"
	"log/slog"
	"subflow/internal/domain"
)

func (s *Service) CategoryRequestFailure(ctx context.Context, actor, action, id string, cause error) error {
	if err := s.audit(ctx, actor, "", action, "category", id, "failure", encodeAuditSummary(map[string]any{"reason": "invalid_request"}, nil)); err != nil {
		slog.Error("category request failure audit could not be persisted", "action", action, "audit_error", err)
		return errors.Join(domain.ErrAuditUnavailable, err, cause)
	}
	return cause
}

func (s *Service) CreateCategory(ctx context.Context, actor string, value domain.Category) (*domain.Category, error) {
	return s.mutateCategory(ctx, actor, value, true)
}

func (s *Service) UpdateCategory(ctx context.Context, actor string, value domain.Category) (*domain.Category, error) {
	return s.mutateCategory(ctx, actor, value, false)
}

func (s *Service) mutateCategory(ctx context.Context, actor string, value domain.Category, create bool) (*domain.Category, error) {
	action := "category.updated"
	if create {
		action = "category.created"
	}
	group := value.GroupID
	var result *domain.Category
	err := s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		var inner error
		if create {
			result, inner = s.createCategory(tx, actor, value)
		} else {
			current, readErr := s.Stores.Categories.Get(tx, value.ID)
			if readErr != nil {
				return readErr
			}
			group = current.GroupID
			result, inner = s.updateCategory(tx, actor, value)
		}
		return inner
	})
	if err != nil {
		if auditErr := s.audit(ctx, actor, group, action, "category", value.ID, "failure", encodeAuditSummary(map[string]any{"reason": "category_operation_failed"}, nil)); auditErr != nil {
			slog.Error("category failure audit could not be persisted", "action", action, "audit_error", auditErr)
			return nil, errors.Join(domain.ErrAuditUnavailable, auditErr, err)
		}
		return nil, err
	}
	return result, nil
}
