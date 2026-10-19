package ports

import (
	"context"
	"subflow/internal/domain"
	"time"
)

type RecurringIncomeRepository interface {
	Create(context.Context, *domain.RecurringIncome) error
	Get(context.Context, string) (*domain.RecurringIncome, error)
	Update(context.Context, *domain.RecurringIncome) error
	List(context.Context, string, string, PageRequest) (Page[domain.RecurringIncome], error)
	ListDue(context.Context, time.Time) ([]domain.RecurringIncome, error)
	CreateRevision(context.Context, *domain.RecurringIncomeRevision) error
	ListRevisions(context.Context, string) ([]domain.RecurringIncomeRevision, error)
	SaveOccurrence(context.Context, *domain.IncomeOccurrence) error
	GetOccurrence(context.Context, string, time.Time) (*domain.IncomeOccurrence, error)
	FindOccurrenceByIncome(context.Context, string) (*domain.IncomeOccurrence, error)
	ListOccurrences(context.Context, string, PageRequest) (Page[domain.IncomeOccurrence], error)
	ReassignUser(context.Context, string, string, string) error
}
