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
type OwnershipTransferRepo struct{ *Repository }
type MemberTransferRepo struct{ *Repository }
type NotificationRepo struct{ *Repository }
type SubscriptionRepo struct{ *Repository }
type ExpenseRepo struct{ *Repository }
type IncomeRepo struct{ *Repository }
type SettlementRepo struct{ *Repository }
type CategoryRepo struct{ *Repository }
type ExchangeRateRepo struct{ *Repository }
type RoleRepo struct{ *Repository }
type AuditRepo struct{ *Repository }
type ShareRepo struct{ *Repository }
type ContactRepo struct{ *Repository }
type UserRepo struct{ *Repository }
type SystemSettingsRepo struct{ *Repository }

type Stores struct {
	Groups             *Repository
	Memberships        *MembershipRepo
	Invitations        *InvitationRepo
	OwnershipTransfers *OwnershipTransferRepo
	MemberTransfers    *MemberTransferRepo
	Notifications      *NotificationRepo
	Subscriptions      *SubscriptionRepo
	Expenses           *ExpenseRepo
	Incomes            *IncomeRepo
	Settlements        *SettlementRepo
	Categories         *CategoryRepo
	ExchangeRates      *ExchangeRateRepo
	Roles              *RoleRepo
	Audits             *AuditRepo
	Shares             *ShareRepo
	Contacts           *ContactRepo
	Users              *UserRepo
	Settings           *SystemSettingsRepo
	Transactions       *Repository
}

func NewStores(app core.App) Stores {
	base := &Repository{App: app}
	return Stores{Groups: base, Memberships: &MembershipRepo{base}, Invitations: &InvitationRepo{base}, OwnershipTransfers: &OwnershipTransferRepo{base}, MemberTransfers: &MemberTransferRepo{base}, Notifications: &NotificationRepo{base}, Subscriptions: &SubscriptionRepo{base}, Expenses: &ExpenseRepo{base}, Incomes: &IncomeRepo{base}, Settlements: &SettlementRepo{base}, Categories: &CategoryRepo{base}, ExchangeRates: &ExchangeRateRepo{base}, Roles: &RoleRepo{base}, Audits: &AuditRepo{base}, Shares: &ShareRepo{base}, Contacts: &ContactRepo{base}, Users: &UserRepo{base}, Settings: &SystemSettingsRepo{base}, Transactions: base}
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
func (r *OwnershipTransferRepo) Create(ctx context.Context, v *domain.OwnershipTransfer) error {
	return r.CreateOwnershipTransfer(ctx, v)
}
func (r *OwnershipTransferRepo) Get(ctx context.Context, id string) (*domain.OwnershipTransfer, error) {
	return r.GetOwnershipTransfer(ctx, id)
}
func (r *OwnershipTransferRepo) FindPending(ctx context.Context, groupID string) (*domain.OwnershipTransfer, error) {
	return r.FindPendingOwnershipTransfer(ctx, groupID)
}
func (r *OwnershipTransferRepo) Update(ctx context.Context, v *domain.OwnershipTransfer) error {
	return r.UpdateOwnershipTransfer(ctx, v)
}
func (r *MemberTransferRepo) Create(ctx context.Context, v *domain.MemberTransfer) error {
	return r.CreateMemberTransfer(ctx, v)
}
func (r *MemberTransferRepo) Get(ctx context.Context, id string) (*domain.MemberTransfer, error) {
	return r.GetMemberTransfer(ctx, id)
}
func (r *MemberTransferRepo) FindPendingByFromUser(ctx context.Context, groupID, fromUserID string) (*domain.MemberTransfer, error) {
	return r.FindPendingMemberTransferByFromUser(ctx, groupID, fromUserID)
}
func (r *MemberTransferRepo) ListPending(ctx context.Context, groupID string) ([]domain.MemberTransfer, error) {
	return r.ListPendingMemberTransfers(ctx, groupID)
}
func (r *MemberTransferRepo) Update(ctx context.Context, v *domain.MemberTransfer) error {
	return r.UpdateMemberTransfer(ctx, v)
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
func (r *NotificationRepo) ReassignUser(ctx context.Context, groupID, fromUserID, toUserID string) error {
	return r.ReassignNotificationUser(ctx, groupID, fromUserID, toUserID)
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
func (r *SubscriptionRepo) UpdateRevision(ctx context.Context, v *domain.SubscriptionRevision) error {
	return r.UpdateSubscriptionRevision(ctx, v)
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
func (r *SubscriptionRepo) ReassignUser(ctx context.Context, groupID, fromUserID, toUserID string) error {
	return r.ReassignSubscriptionUser(ctx, groupID, fromUserID, toUserID)
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
func (r *ExpenseRepo) ListPersonalBetween(ctx context.Context, userID string, from, to time.Time) ([]domain.Expense, error) {
	return r.ListPersonalExpensesBetween(ctx, userID, from, to)
}
func (r *ExpenseRepo) ListBetween(ctx context.Context, groupID string, from, to time.Time) ([]domain.Expense, error) {
	return r.ListGroupExpensesBetween(ctx, groupID, from, to)
}
func (r *ExpenseRepo) ReplaceSplits(ctx context.Context, expenseID string, values []domain.ExpenseSplit) error {
	return r.ReplaceExpenseSplits(ctx, expenseID, values)
}
func (r *ExpenseRepo) ListSplits(ctx context.Context, expenseID string) ([]domain.ExpenseSplit, error) {
	return r.ListExpenseSplits(ctx, expenseID)
}
func (r *ExpenseRepo) ReassignUser(ctx context.Context, groupID, fromUserID, toUserID string) error {
	return r.ReassignExpenseUser(ctx, groupID, fromUserID, toUserID)
}

func (r *SettlementRepo) Create(ctx context.Context, v *domain.Settlement) error {
	return r.CreateSettlement(ctx, v)
}
func (r *SettlementRepo) Get(ctx context.Context, id string) (*domain.Settlement, error) {
	return r.GetSettlement(ctx, id)
}
func (r *SettlementRepo) List(ctx context.Context, groupID string, query ports.SettlementQuery) (ports.Page[domain.Settlement], error) {
	return r.ListSettlements(ctx, groupID, query)
}
func (r *SettlementRepo) Update(ctx context.Context, v *domain.Settlement) error {
	return r.UpdateSettlement(ctx, v)
}
func (r *SettlementRepo) Delete(ctx context.Context, id string) error {
	return r.DeleteSettlement(ctx, id)
}
func (r *SettlementRepo) ReassignUser(ctx context.Context, groupID, fromUserID, toUserID string) error {
	return r.ReassignSettlementUser(ctx, groupID, fromUserID, toUserID)
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
func (r *CategoryRepo) ReassignUser(ctx context.Context, groupID, fromUserID, toUserID string) error {
	return r.ReassignCategoryUser(ctx, groupID, fromUserID, toUserID)
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
func (r *ShareRepo) Create(ctx context.Context, v *domain.Share) error { return r.CreateShare(ctx, v) }
func (r *ShareRepo) Get(ctx context.Context, id string) (*domain.Share, error) {
	return r.GetShare(ctx, id)
}
func (r *ShareRepo) GetByTokenHash(ctx context.Context, hash string) (*domain.Share, error) {
	return r.GetShareByTokenHash(ctx, hash)
}
func (r *ShareRepo) List(ctx context.Context, groupID, ownerID string, req ports.PageRequest) (ports.Page[domain.Share], error) {
	return r.ListShares(ctx, groupID, ownerID, req)
}
func (r *ShareRepo) Update(ctx context.Context, v *domain.Share) error { return r.UpdateShare(ctx, v) }
func (r *ShareRepo) Delete(ctx context.Context, id string) error       { return r.DeleteShare(ctx, id) }
func (r *ShareRepo) ReplaceViewers(ctx context.Context, shareID string, values []domain.ShareViewer) error {
	return r.ReplaceShareViewers(ctx, shareID, values)
}
func (r *ShareRepo) ListViewers(ctx context.Context, shareID string) ([]domain.ShareViewer, error) {
	return r.ListShareViewers(ctx, shareID)
}

func (r *ContactRepo) Create(ctx context.Context, v *domain.Contact) error {
	return r.CreateContact(ctx, v)
}
func (r *ContactRepo) Get(ctx context.Context, id string) (*domain.Contact, error) {
	return r.GetContact(ctx, id)
}
func (r *ContactRepo) List(ctx context.Context, ownerID string, req ports.PageRequest) (ports.Page[domain.Contact], error) {
	return r.ListContacts(ctx, ownerID, req)
}
func (r *ContactRepo) Update(ctx context.Context, v *domain.Contact) error {
	return r.UpdateContact(ctx, v)
}
func (r *ContactRepo) Delete(ctx context.Context, id string) error { return r.DeleteContact(ctx, id) }

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
func (r *UserRepo) ListExternalAuths(ctx context.Context, userID string) ([]domain.LinkedProvider, error) {
	return r.Repository.ListUserExternalAuths(ctx, userID)
}
func (r *UserRepo) UnlinkExternalAuth(ctx context.Context, userID, provider string) error {
	return r.Repository.UnlinkUserExternalAuth(ctx, userID, provider)
}
func (r *SystemSettingsRepo) Get(ctx context.Context) (domain.SystemSettings, error) {
	return r.Repository.GetSystemSettings(ctx)
}
func (r *SystemSettingsRepo) Save(ctx context.Context, value domain.SystemSettings) error {
	return r.Repository.SaveSystemSettings(ctx, value)
}
func (r *IncomeRepo) Create(ctx context.Context, v *domain.Income) error {
	return r.CreateIncome(ctx, v)
}
func (r *IncomeRepo) Get(ctx context.Context, id string) (*domain.Income, error) {
	return r.GetIncome(ctx, id)
}
func (r *IncomeRepo) ListPersonal(ctx context.Context, userID string, req ports.PageRequest) (ports.Page[domain.Income], error) {
	return r.ListPersonalIncomes(ctx, userID, req)
}
func (r *IncomeRepo) List(ctx context.Context, groupID string, req ports.PageRequest) (ports.Page[domain.Income], error) {
	return r.ListIncomes(ctx, groupID, req)
}
func (r *IncomeRepo) ListPersonalBetween(ctx context.Context, userID string, from, to time.Time) ([]domain.Income, error) {
	return r.ListPersonalIncomesBetween(ctx, userID, from, to)
}
func (r *IncomeRepo) ListBetween(ctx context.Context, groupID string, from, to time.Time) ([]domain.Income, error) {
	return r.ListIncomesBetween(ctx, groupID, from, to)
}
func (r *IncomeRepo) Update(ctx context.Context, v *domain.Income) error {
	return r.UpdateIncome(ctx, v)
}
func (r *IncomeRepo) Delete(ctx context.Context, id string) error { return r.DeleteIncome(ctx, id) }
