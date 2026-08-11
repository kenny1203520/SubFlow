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

// AuditQuery keeps audit-specific filters separate from general list requests.
// Dates are already normalized to UTC by the HTTP layer.
type AuditQuery struct {
	PageRequest
	Query    string
	Action   string
	Resource string
	Outcome  string
	From     time.Time
	To       time.Time
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
	UpdateRole(context.Context, string, string, string) error
}

type RoleRepository interface {
	Create(context.Context, *domain.Role) error
	Get(context.Context, string, string) (*domain.Role, error)
	List(context.Context, string, string) ([]domain.Role, error)
	Update(context.Context, *domain.Role) error
	Delete(context.Context, string, string) error
}

type AuditRepository interface {
	Create(context.Context, *domain.AuditLog) error
	List(context.Context, string, AuditQuery) (Page[domain.AuditLog], error)
}

type InvitationRepository interface {
	Create(context.Context, *domain.Invitation) error
	Get(context.Context, string) (*domain.Invitation, error)
	GetByTokenHash(context.Context, string) (*domain.Invitation, error)
	FindPending(context.Context, string, string) (*domain.Invitation, error)
	List(context.Context, string, PageRequest) (Page[domain.Invitation], error)
	Update(context.Context, *domain.Invitation) error
	ListForEmail(context.Context, string, PageRequest) (Page[domain.Invitation], error)
}

type NotificationRepository interface {
	Create(context.Context, *domain.Notification) error
	Get(context.Context, string) (*domain.Notification, error)
	ListForUser(context.Context, string, PageRequest) (Page[domain.Notification], error)
	MarkRead(context.Context, string, time.Time) error
	MarkReadForResource(context.Context, string, string, time.Time) error
}

type SubscriptionRepository interface {
	Create(context.Context, *domain.Subscription) error
	Get(context.Context, string) (*domain.Subscription, error)
	List(context.Context, string, PageRequest) (Page[domain.Subscription], error)
	ListPersonal(context.Context, string, PageRequest) (Page[domain.Subscription], error)
	ListAutomatic(context.Context) ([]domain.Subscription, error)
	Update(context.Context, *domain.Subscription) error
	Delete(context.Context, string) error
	CreateRevision(context.Context, *domain.SubscriptionRevision) error
	ListRevisions(context.Context, string) ([]domain.SubscriptionRevision, error)
	CreateOccurrence(context.Context, *domain.SubscriptionOccurrence) error
	GetOccurrence(context.Context, string, time.Time) (*domain.SubscriptionOccurrence, error)
	ListOccurrences(context.Context, string) ([]domain.SubscriptionOccurrence, error)
	UpdateOccurrence(context.Context, *domain.SubscriptionOccurrence) error
	ListDue(context.Context, time.Time) ([]domain.Subscription, error)
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
	List(context.Context, PageRequest, string) (Page[domain.User], error)
	SetSystemRole(context.Context, string, string) error
	Create(context.Context, domain.SetupInput) (*domain.User, error)
	CountBySystemRole(context.Context, string) (int, error)
	// CreatePlaceholder, LinkPlaceholder and Delete back the "temp member"
	// feature (see application.Service.CreateTempMember).
	CreatePlaceholder(context.Context, string) (*domain.User, error)
	LinkPlaceholder(ctx context.Context, placeholderID, realUserID string) error
	Delete(context.Context, string) error
	// ListExternalAuths and UnlinkExternalAuth back the "connected accounts"
	// feature (see application.Service.ListLinkedProviders / UnlinkProvider).
	ListExternalAuths(ctx context.Context, userID string) ([]domain.LinkedProvider, error)
	UnlinkExternalAuth(ctx context.Context, userID, provider string) error
}

type SystemSettingsRepository interface {
	Get(context.Context) (domain.SystemSettings, error)
	Save(context.Context, domain.SystemSettings) error
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
