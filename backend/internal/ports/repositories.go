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
	// FindPendingByTarget finds a still-pending invitation already targeting
	// the given placeholder in the group, so a second invite can't be sent
	// to the same placeholder while one is outstanding (see
	// CollaborationService.CreateInvitationBinding).
	FindPendingByTarget(ctx context.Context, groupID, placeholderID string) (*domain.Invitation, error)
	List(context.Context, string, PageRequest) (Page[domain.Invitation], error)
	Update(context.Context, *domain.Invitation) error
	ListForEmail(context.Context, string, PageRequest) (Page[domain.Invitation], error)
}

type OwnershipTransferRepository interface {
	Create(context.Context, *domain.OwnershipTransfer) error
	Get(context.Context, string) (*domain.OwnershipTransfer, error)
	// FindPending finds a still-pending transfer for the group, so a second
	// transfer can't be started while one is outstanding.
	FindPending(ctx context.Context, groupID string) (*domain.OwnershipTransfer, error)
	Update(context.Context, *domain.OwnershipTransfer) error
}

type MemberTransferRepository interface {
	Create(context.Context, *domain.MemberTransfer) error
	Get(context.Context, string) (*domain.MemberTransfer, error)
	// FindPendingByFromUser finds a still-pending transfer for that member
	// within the group, so a second transfer can't be started for the same
	// source member while one is outstanding.
	FindPendingByFromUser(ctx context.Context, groupID, fromUserID string) (*domain.MemberTransfer, error)
	// ListPending lists every pending transfer in the group (unlike
	// OwnershipTransfer, several can be outstanding at once for different
	// member pairs), for the members UI to show what's in flight.
	ListPending(ctx context.Context, groupID string) ([]domain.MemberTransfer, error)
	Update(context.Context, *domain.MemberTransfer) error
}

type NotificationRepository interface {
	Create(context.Context, *domain.Notification) error
	Get(context.Context, string) (*domain.Notification, error)
	ListForUser(context.Context, string, PageRequest) (Page[domain.Notification], error)
	MarkRead(context.Context, string, time.Time) error
	MarkReadForResource(context.Context, string, string, time.Time) error
	// ReassignUser moves every notification.user reference from fromUserID to
	// toUserID within groupID (see Service.repointUserReferences).
	ReassignUser(ctx context.Context, groupID, fromUserID, toUserID string) error
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
	// UpdateRevision persists a revision's fields in place (used by
	// Service.repointUserReferences to rewrite paid_by/splits on an existing
	// revision snapshot; ordinary edits always create a new revision instead).
	UpdateRevision(context.Context, *domain.SubscriptionRevision) error
	CreateOccurrence(context.Context, *domain.SubscriptionOccurrence) error
	GetOccurrence(context.Context, string, time.Time) (*domain.SubscriptionOccurrence, error)
	ListOccurrences(context.Context, string) ([]domain.SubscriptionOccurrence, error)
	UpdateOccurrence(context.Context, *domain.SubscriptionOccurrence) error
	ListDue(context.Context, time.Time) ([]domain.Subscription, error)
	// ReassignUser moves every subscription.paid_by/splits and every one of
	// its revisions' paid_by/splits from fromUserID to toUserID within
	// groupID (see Service.repointUserReferences).
	ReassignUser(ctx context.Context, groupID, fromUserID, toUserID string) error
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
	// ReassignUser moves every expense.paid_by and expense_splits.user
	// reference from fromUserID to toUserID within groupID, merging into an
	// existing split row rather than violating the (expense, user) unique
	// index when both already have one on the same expense (see
	// Service.repointUserReferences).
	ReassignUser(ctx context.Context, groupID, fromUserID, toUserID string) error
}

type SettlementRepository interface {
	Create(context.Context, *domain.Settlement) error
	Get(context.Context, string) (*domain.Settlement, error)
	List(context.Context, string, PageRequest) (Page[domain.Settlement], error)
	Update(context.Context, *domain.Settlement) error
	Delete(context.Context, string) error
	// ReassignUser moves every from_user/to_user/created_by reference from
	// fromUserID to toUserID within groupID, deleting any settlement that
	// would become self-referential (from_user==to_user) as a result (see
	// Service.repointUserReferences).
	ReassignUser(ctx context.Context, groupID, fromUserID, toUserID string) error
}

type CategoryRepository interface {
	Create(context.Context, *domain.Category) error
	Get(context.Context, string) (*domain.Category, error)
	List(context.Context, string, string, bool) ([]domain.Category, error)
	Update(context.Context, *domain.Category) error
	// ReassignUser moves group-scope categories.created_by from fromUserID to
	// toUserID within groupID; personal-scope categories.owner is
	// intentionally out of the group-scoped boundary (see
	// Service.repointUserReferences).
	ReassignUser(ctx context.Context, groupID, fromUserID, toUserID string) error
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
	// inviterName is a best-effort display name for domain.Invitation.InvitedBy,
	// resolved by the caller since Mailer implementations don't have a
	// UserDirectory of their own; it may be empty.
	SendInvitation(ctx context.Context, inv domain.Invitation, group domain.Group, inviterName, url string) error
	Configured() bool
}

type Clock interface{ Now() time.Time }
