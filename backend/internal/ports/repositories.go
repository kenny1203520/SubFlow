package ports

import (
	"context"
	"time"

	"subflow/internal/domain"
)

type PageRequest struct {
	Page    int
	PerPage int
	Sort    string
}

type Page[T any] struct {
	Items      []T `json:"items"`
	Page       int `json:"page"`
	PerPage    int `json:"perPage"`
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
}

type GroupRepository interface {
	Create(context.Context, *domain.Group) error
	Get(context.Context, string) (*domain.Group, error)
	ListForUser(context.Context, string, PageRequest) (Page[domain.Group], error)
	Update(context.Context, *domain.Group) error
	Delete(context.Context, string) error
}

type MembershipRepository interface {
	Create(context.Context, *domain.Membership) error
	GetRole(context.Context, string, string) (domain.MemberRole, error)
	List(context.Context, string, PageRequest) (Page[domain.Membership], error)
	Delete(context.Context, string, string) error
}

type InvitationRepository interface {
	Create(context.Context, *domain.Invitation) error
	Get(context.Context, string) (*domain.Invitation, error)
	GetByTokenHash(context.Context, string) (*domain.Invitation, error)
	FindPending(context.Context, string, string) (*domain.Invitation, error)
	List(context.Context, string, PageRequest) (Page[domain.Invitation], error)
	Update(context.Context, *domain.Invitation) error
}

type SubscriptionRepository interface {
	Create(context.Context, *domain.Subscription) error
	Get(context.Context, string) (*domain.Subscription, error)
	List(context.Context, string, PageRequest) (Page[domain.Subscription], error)
	ListPersonal(context.Context, string, PageRequest) (Page[domain.Subscription], error)
	ListAutomatic(context.Context) ([]domain.Subscription, error)
	Update(context.Context, *domain.Subscription) error
	Delete(context.Context, string) error
}

type ExpenseRepository interface {
	Create(context.Context, *domain.Expense) error
	Get(context.Context, string) (*domain.Expense, error)
	List(context.Context, string, PageRequest) (Page[domain.Expense], error)
	ListPersonal(context.Context, string, PageRequest) (Page[domain.Expense], error)
	Update(context.Context, *domain.Expense) error
	Delete(context.Context, string) error
	ReplaceSplits(context.Context, string, []domain.ExpenseSplit) error
	ListSplits(context.Context, string) ([]domain.ExpenseSplit, error)
}

type SettlementRepository interface {
	Create(context.Context, *domain.Settlement) error
	Get(context.Context, string) (*domain.Settlement, error)
	List(context.Context, string, PageRequest) (Page[domain.Settlement], error)
	Update(context.Context, *domain.Settlement) error
	Delete(context.Context, string) error
}

type CategoryRepository interface {
	Create(context.Context, *domain.Category) error
	Get(context.Context, string) (*domain.Category, error)
	List(context.Context, string, string, bool) ([]domain.Category, error)
	Update(context.Context, *domain.Category) error
}

type ExchangeRateRepository interface {
	Upsert(context.Context, *domain.ExchangeRate) error
	LatestOnOrBefore(context.Context, domain.Currency, domain.Currency, time.Time) (*domain.ExchangeRate, error)
}

type UserDirectory interface {
	Get(context.Context, string) (*domain.User, error)
	FindByEmail(context.Context, string) (*domain.User, error)
}

type TransactionManager interface {
	Within(context.Context, func(context.Context) error) error
}

type EventPublisher interface {
	Publish(context.Context, domain.Event) error
	Subscribe(context.Context, string) (<-chan domain.Event, func())
	SubscribeWorkspace(context.Context, string, []string) (<-chan domain.Event, func())
}

type Mailer interface {
	SendInvitation(context.Context, domain.Invitation, domain.Group, string) error
	Configured() bool
}

type Clock interface{ Now() time.Time }
