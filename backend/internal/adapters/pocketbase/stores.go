package pocketbase

import (
	"context"

	"github.com/pocketbase/pocketbase/core"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

type MembershipRepo struct{ *Repository }
type InvitationRepo struct{ *Repository }
type SubscriptionRepo struct{ *Repository }
type ExpenseRepo struct{ *Repository }
type UserRepo struct{ *Repository }

type Stores struct {
	Groups        *Repository
	Memberships   *MembershipRepo
	Invitations   *InvitationRepo
	Subscriptions *SubscriptionRepo
	Expenses      *ExpenseRepo
	Users         *UserRepo
	Transactions  *Repository
}

func NewStores(app core.App) Stores {
	base := &Repository{App: app}
	return Stores{base, &MembershipRepo{base}, &InvitationRepo{base}, &SubscriptionRepo{base}, &ExpenseRepo{base}, &UserRepo{base}, base}
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

func (r *UserRepo) Get(ctx context.Context, id string) (*domain.User, error) {
	return r.GetUser(ctx, id)
}
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.Repository.FindByEmail(ctx, email)
}
