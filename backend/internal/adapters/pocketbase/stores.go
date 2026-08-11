package pocketbase

import (
	"context"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

type MembershipRepo struct{ *Repository }
type InvitationRepo struct{ *Repository }
type NotificationRepo struct{ *Repository }
type SubscriptionRepo struct{ *Repository }
type ExpenseRepo struct{ *Repository }
type SettlementRepo struct{ *Repository }
type CategoryRepo struct{ *Repository }
type ExchangeRateRepo struct{ *Repository }
type RoleRepo struct{ *Repository }
type AuditRepo struct{ *Repository }
type UserRepo struct{ *Repository }
type SystemSettingsRepo struct{ *Repository }

type Stores struct {
	Groups        *Repository
	Memberships   *MembershipRepo
	Invitations   *InvitationRepo
	Notifications *NotificationRepo
	Subscriptions *SubscriptionRepo
	Expenses      *ExpenseRepo
	Settlements   *SettlementRepo
	Categories    *CategoryRepo
	ExchangeRates *ExchangeRateRepo
	Roles         *RoleRepo
	Audits        *AuditRepo
	Users         *UserRepo
	Settings      *SystemSettingsRepo
	Transactions  *Repository
}

func NewStores(app core.App) Stores {
	base := &Repository{App: app}
	return Stores{base, &MembershipRepo{base}, &InvitationRepo{base}, &NotificationRepo{base}, &SubscriptionRepo{base}, &ExpenseRepo{base}, &SettlementRepo{base}, &CategoryRepo{base}, &ExchangeRateRepo{base}, &RoleRepo{base}, &AuditRepo{base}, &UserRepo{base}, &SystemSettingsRepo{base}, base}
}

func (r *MembershipRepo) Create(ctx context.Context, v *domain.Membership) error {
	return r.CreateMembership(ctx, v)
}
func (r *MembershipRepo) List(ctx context.Context, groupID string, req ports.PageRequest) (ports.Page[domain.Membership], error) {
	return r.ListMemberships(ctx, groupID, req)
}
func (r *MembershipRepo) Delete(ctx context.Context, groupID, userID string) error {
	return r.DeleteMembership(ctx, groupID, userID)
}
func (r *MembershipRepo) UpdateRole(ctx context.Context, groupID, userID, roleID string) error {
	return r.UpdateMembershipRole(ctx, groupID, userID, roleID)
}

func (r *InvitationRepo) Create(ctx context.Context, v *domain.Invitation) error {
	return r.CreateInvitation(ctx, v)
}
func (r *InvitationRepo) Get(ctx context.Context, id string) (*domain.Invitation, error) {
	return r.GetInvitation(ctx, id)
}
func (r *InvitationRepo) List(ctx context.Context, groupID string, req ports.PageRequest) (ports.Page[domain.Invitation], error) {
	return r.ListInvitations(ctx, groupID, req)
}
func (r *InvitationRepo) Update(ctx context.Context, v *domain.Invitation) error {
	return r.UpdateInvitation(ctx, v)
}
func (r *InvitationRepo) ListForEmail(ctx context.Context, email string, req ports.PageRequest) (ports.Page[domain.Invitation], error) {
	return r.ListInvitationsForEmail(ctx, email, req)
}
func (r *NotificationRepo) Create(ctx context.Context, v *domain.Notification) error {
	return r.CreateNotification(ctx, v)
}
func (r *NotificationRepo) Get(ctx context.Context, id string) (*domain.Notification, error) {
	return r.GetNotification(ctx, id)
}
func (r *NotificationRepo) ListForUser(ctx context.Context, id string, req ports.PageRequest) (ports.Page[domain.Notification], error) {
	return r.ListNotifications(ctx, id, req)
}
func (r *NotificationRepo) MarkRead(ctx context.Context, id string, when time.Time) error {
	return r.MarkNotificationRead(ctx, id, when)
}
func (r *NotificationRepo) MarkReadForResource(ctx context.Context, userID, resourceID string, when time.Time) error {
	return r.MarkNotificationsReadForResource(ctx, userID, resourceID, when)
}

func (r *SubscriptionRepo) Create(ctx context.Context, v *domain.Subscription) error {
	return r.CreateSubscription(ctx, v)
}
func (r *SubscriptionRepo) Get(ctx context.Context, id string) (*domain.Subscription, error) {
	return r.GetSubscription(ctx, id)
}
func (r *SubscriptionRepo) List(ctx context.Context, groupID string, req ports.PageRequest) (ports.Page[domain.Subscription], error) {
	return r.ListSubscriptions(ctx, groupID, req)
}
func (r *SubscriptionRepo) Update(ctx context.Context, v *domain.Subscription) error {
	return r.UpdateSubscription(ctx, v)
}
func (r *SubscriptionRepo) Delete(ctx context.Context, id string) error {
	return r.DeleteSubscription(ctx, id)
}
func (r *SubscriptionRepo) ListPersonal(ctx context.Context, userID string, req ports.PageRequest) (ports.Page[domain.Subscription], error) {
	return r.ListPersonalSubscriptions(ctx, userID, req)
}
func (r *SubscriptionRepo) ListAutomatic(ctx context.Context) ([]domain.Subscription, error) {
	return r.ListAutomaticSubscriptions(ctx)
}
func (r *SubscriptionRepo) CreateRevision(ctx context.Context, v *domain.SubscriptionRevision) error {
	return r.CreateSubscriptionRevision(ctx, v)
}
func (r *SubscriptionRepo) ListRevisions(ctx context.Context, subscriptionID string) ([]domain.SubscriptionRevision, error) {
	return r.ListSubscriptionRevisions(ctx, subscriptionID)
}
func (r *SubscriptionRepo) CreateOccurrence(ctx context.Context, v *domain.SubscriptionOccurrence) error {
	return r.CreateSubscriptionOccurrence(ctx, v)
}
func (r *SubscriptionRepo) GetOccurrence(ctx context.Context, subscriptionID string, billingAt time.Time) (*domain.SubscriptionOccurrence, error) {
	return r.GetSubscriptionOccurrence(ctx, subscriptionID, billingAt)
}
func (r *SubscriptionRepo) ListOccurrences(ctx context.Context, subscriptionID string) ([]domain.SubscriptionOccurrence, error) {
	return r.ListSubscriptionOccurrences(ctx, subscriptionID)
}
func (r *SubscriptionRepo) UpdateOccurrence(ctx context.Context, v *domain.SubscriptionOccurrence) error {
	return r.UpdateSubscriptionOccurrence(ctx, v)
}
func (r *SubscriptionRepo) ListDue(ctx context.Context, before time.Time) ([]domain.Subscription, error) {
	return r.ListDueSubscriptions(ctx, before)
}

func (r *ExpenseRepo) Create(ctx context.Context, v *domain.Expense) error {
	return r.CreateExpense(ctx, v)
}
func (r *ExpenseRepo) Get(ctx context.Context, id string) (*domain.Expense, error) {
	return r.GetExpense(ctx, id)
}
func (r *ExpenseRepo) List(ctx context.Context, groupID string, req ports.PageRequest) (ports.Page[domain.Expense], error) {
	return r.ListExpenses(ctx, groupID, req)
}
func (r *ExpenseRepo) Update(ctx context.Context, v *domain.Expense) error {
	return r.UpdateExpense(ctx, v)
}
func (r *ExpenseRepo) Delete(ctx context.Context, id string) error { return r.DeleteExpense(ctx, id) }
func (r *ExpenseRepo) ListPersonal(ctx context.Context, userID string, req ports.PageRequest) (ports.Page[domain.Expense], error) {
	return r.ListPersonalExpenses(ctx, userID, req)
}
func (r *ExpenseRepo) ReplaceSplits(ctx context.Context, expenseID string, values []domain.ExpenseSplit) error {
	return r.ReplaceExpenseSplits(ctx, expenseID, values)
}
func (r *ExpenseRepo) ListSplits(ctx context.Context, expenseID string) ([]domain.ExpenseSplit, error) {
	return r.ListExpenseSplits(ctx, expenseID)
}

func (r *SettlementRepo) Create(ctx context.Context, v *domain.Settlement) error {
	return r.CreateSettlement(ctx, v)
}
func (r *SettlementRepo) Get(ctx context.Context, id string) (*domain.Settlement, error) {
	return r.GetSettlement(ctx, id)
}
func (r *SettlementRepo) List(ctx context.Context, groupID string, req ports.PageRequest) (ports.Page[domain.Settlement], error) {
	return r.ListSettlements(ctx, groupID, req)
}
func (r *SettlementRepo) Update(ctx context.Context, v *domain.Settlement) error {
	return r.UpdateSettlement(ctx, v)
}
func (r *SettlementRepo) Delete(ctx context.Context, id string) error {
	return r.DeleteSettlement(ctx, id)
}

func (r *CategoryRepo) Create(ctx context.Context, v *domain.Category) error {
	return r.CreateCategory(ctx, v)
}
func (r *CategoryRepo) Get(ctx context.Context, id string) (*domain.Category, error) {
	return r.GetCategory(ctx, id)
}
func (r *CategoryRepo) List(ctx context.Context, ownerID, groupID string, archived bool) ([]domain.Category, error) {
	return r.ListCategories(ctx, ownerID, groupID, archived)
}
func (r *CategoryRepo) Update(ctx context.Context, v *domain.Category) error {
	return r.UpdateCategory(ctx, v)
}
func (r *ExchangeRateRepo) Upsert(ctx context.Context, v *domain.ExchangeRate) error {
	return r.UpsertExchangeRate(ctx, v)
}
func (r *ExchangeRateRepo) LatestOnOrBefore(ctx context.Context, from, to domain.Currency, date time.Time) (*domain.ExchangeRate, error) {
	return r.LatestExchangeRate(ctx, from, to, date)
}
func (r *RoleRepo) Create(ctx context.Context, v *domain.Role) error { return r.CreateRole(ctx, v) }
func (r *RoleRepo) Get(ctx context.Context, scope, id string) (*domain.Role, error) {
	return r.GetRoleRecord(ctx, scope, id)
}
func (r *RoleRepo) List(ctx context.Context, scope, groupID string) ([]domain.Role, error) {
	return r.ListRoles(ctx, scope, groupID)
}
func (r *RoleRepo) Update(ctx context.Context, v *domain.Role) error {
	return r.UpdateRoleRecord(ctx, v)
}
func (r *RoleRepo) Delete(ctx context.Context, scope, id string) error {
	return r.DeleteRoleRecord(ctx, scope, id)
}
func (r *AuditRepo) Create(ctx context.Context, v *domain.AuditLog) error {
	return r.CreateAudit(ctx, v)
}
func (r *AuditRepo) List(ctx context.Context, groupID string, query ports.AuditQuery) (ports.Page[domain.AuditLog], error) {
	return r.ListAudits(ctx, groupID, query)
}

func (r *UserRepo) Get(ctx context.Context, id string) (*domain.User, error) {
	return r.GetUser(ctx, id)
}
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.Repository.FindByEmail(ctx, email)
}
func (r *UserRepo) List(ctx context.Context, req ports.PageRequest, query string) (ports.Page[domain.User], error) {
	return r.Repository.ListUsers(ctx, req, query)
}
func (r *UserRepo) SetSystemRole(ctx context.Context, id, roleID string) error {
	return r.Repository.SetUserSystemRole(ctx, id, roleID)
}
func (r *UserRepo) Create(ctx context.Context, input domain.SetupInput) (*domain.User, error) {
	return r.Repository.CreateSetupUser(ctx, input)
}
func (r *UserRepo) CountBySystemRole(ctx context.Context, roleID string) (int, error) {
	return r.Repository.CountUsersBySystemRole(ctx, roleID)
}
func (r *UserRepo) CreatePlaceholder(ctx context.Context, name string) (*domain.User, error) {
	return r.Repository.CreatePlaceholder(ctx, name)
}
func (r *UserRepo) LinkPlaceholder(ctx context.Context, placeholderID, realUserID string) error {
	return r.Repository.LinkPlaceholder(ctx, placeholderID, realUserID)
}
func (r *UserRepo) Delete(ctx context.Context, id string) error {
	return r.Repository.DeleteUser(ctx, id)
}
func (r *SystemSettingsRepo) Get(ctx context.Context) (domain.SystemSettings, error) {
	return r.Repository.GetSystemSettings(ctx)
}
func (r *SystemSettingsRepo) Save(ctx context.Context, value domain.SystemSettings) error {
	return r.Repository.SaveSystemSettings(ctx, value)
}
