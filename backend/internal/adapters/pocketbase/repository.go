package pocketbase

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

type Repository struct{ App core.App }

type txKey struct{}

func (r *Repository) app(ctx context.Context) core.App {
	if app, ok := ctx.Value(txKey{}).(core.App); ok {
		return app
	}
	return r.App
}

func (r *Repository) Within(ctx context.Context, fn func(context.Context) error) error {
	return r.app(ctx).RunInTransaction(func(tx core.App) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

func (r *Repository) Create(ctx context.Context, group *domain.Group) error {
	return r.CreateGroup(ctx, group)
}

func (r *Repository) CreateGroup(ctx context.Context, group *domain.Group) error {
	record, err := newRecord(r.app(ctx), CollectionGroups)
	if err != nil {
		return err
	}
	writeGroup(record, group)
	if err := r.app(ctx).Save(record); err != nil {
		return err
	}
	group.ID = record.Id
	hydrateTimes(record, &group.CreatedAt, &group.UpdatedAt)
	return nil
}

func (r *Repository) Get(ctx context.Context, id string) (*domain.Group, error) {
	return r.GetGroup(ctx, id)
}
func (r *Repository) GetGroup(ctx context.Context, id string) (*domain.Group, error) {
	record, err := r.app(ctx).FindRecordById(CollectionGroups, id)
	if err != nil {
		return nil, mapError(err)
	}
	return groupFrom(record), nil
}
func (r *Repository) ListForUser(ctx context.Context, userID string, req ports.PageRequest) (ports.Page[domain.Group], error) {
	members, err := r.app(ctx).FindRecordsByFilter(CollectionMembers, "user={:user}", "-created", 0, 0, dbx.Params{"user": userID})
	if err != nil {
		return ports.Page[domain.Group]{}, err
	}
	ids := make([]string, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.GetString("group"))
	}
	items := make([]domain.Group, 0, len(ids))
	for _, id := range ids {
		if g, e := r.GetGroup(ctx, id); e == nil {
			items = append(items, *g)
		}
	}
	return slicePage(items, req), nil
}
func (r *Repository) Update(ctx context.Context, group *domain.Group) error {
	return r.UpdateGroup(ctx, group)
}
func (r *Repository) UpdateGroup(ctx context.Context, group *domain.Group) error {
	record, err := r.app(ctx).FindRecordById(CollectionGroups, group.ID)
	if err != nil {
		return mapError(err)
	}
	writeGroup(record, group)
	if err := r.app(ctx).Save(record); err != nil {
		return err
	}
	hydrateTimes(record, &group.CreatedAt, &group.UpdatedAt)
	return nil
}
func (r *Repository) Delete(ctx context.Context, id string) error { return r.DeleteGroup(ctx, id) }
func (r *Repository) DeleteGroup(ctx context.Context, id string) error {
	record, err := r.app(ctx).FindRecordById(CollectionGroups, id)
	if err != nil {
		return mapError(err)
	}
	return r.app(ctx).Delete(record)
}

func (r *Repository) CreateMembership(ctx context.Context, m *domain.Membership) error {
	record, err := newRecord(r.app(ctx), CollectionMembers)
	if err != nil {
		return err
	}
	record.Set("group", m.GroupID)
	record.Set("user", m.UserID)
	record.Set("role", m.Role)
	if err := r.app(ctx).Save(record); err != nil {
		return err
	}
	m.ID = record.Id
	m.CreatedAt = record.GetDateTime("created").Time()
	return nil
}
func (r *Repository) GetRole(ctx context.Context, groupID, userID string) (domain.MemberRole, error) {
	record, err := r.app(ctx).FindFirstRecordByFilter(CollectionMembers, "group={:group} && user={:user}", dbx.Params{"group": groupID, "user": userID})
	if err != nil {
		return "", mapError(err)
	}
	return domain.MemberRole(record.GetString("role")), nil
}
func (r *Repository) ListMemberships(ctx context.Context, groupID string, req ports.PageRequest) (ports.Page[domain.Membership], error) {
	records, err := listRecords(r.app(ctx), CollectionMembers, "group={:group}", req, dbx.Params{"group": groupID})
	if err != nil {
		return ports.Page[domain.Membership]{}, err
	}
	items := make([]domain.Membership, 0, len(records))
	for _, v := range records {
		m := membershipFrom(v)
		if u, e := r.GetUser(ctx, m.UserID); e == nil {
			m.User = u
		}
		items = append(items, m)
	}
	count, _ := countFiltered(r.app(ctx), CollectionMembers, "group={:group}", dbx.Params{"group": groupID})
	return page(items, req, count), nil
}
func (r *Repository) DeleteMembership(ctx context.Context, groupID, userID string) error {
	rec, err := r.app(ctx).FindFirstRecordByFilter(CollectionMembers, "group={:group} && user={:user}", dbx.Params{"group": groupID, "user": userID})
	if err != nil {
		return mapError(err)
	}
	return r.app(ctx).Delete(rec)
}

func (r *Repository) CreateInvitation(ctx context.Context, v *domain.Invitation) error {
	rec, err := newRecord(r.app(ctx), CollectionInvitations)
	if err != nil {
		return err
	}
	writeInvitation(rec, v)
	if err = r.app(ctx).Save(rec); err != nil {
		return err
	}
	v.ID = rec.Id
	hydrateTimes(rec, &v.CreatedAt, &v.UpdatedAt)
	return nil
}
func (r *Repository) GetInvitation(ctx context.Context, id string) (*domain.Invitation, error) {
	rec, err := r.app(ctx).FindRecordById(CollectionInvitations, id)
	if err != nil {
		return nil, mapError(err)
	}
	return invitationFrom(rec), nil
}
func (r *Repository) GetByTokenHash(ctx context.Context, hash string) (*domain.Invitation, error) {
	rec, err := r.app(ctx).FindFirstRecordByFilter(CollectionInvitations, "token_hash={:hash}", dbx.Params{"hash": hash})
	if err != nil {
		return nil, mapError(err)
	}
	return invitationFrom(rec), nil
}
func (r *Repository) FindPending(ctx context.Context, groupID, email string) (*domain.Invitation, error) {
	rec, err := r.app(ctx).FindFirstRecordByFilter(CollectionInvitations, "group={:group} && email={:email} && status='pending'", dbx.Params{"group": groupID, "email": email})
	if err != nil {
		return nil, mapError(err)
	}
	return invitationFrom(rec), nil
}
func (r *Repository) ListInvitations(ctx context.Context, groupID string, req ports.PageRequest) (ports.Page[domain.Invitation], error) {
	recs, err := listRecords(r.app(ctx), CollectionInvitations, "group={:group}", req, dbx.Params{"group": groupID})
	if err != nil {
		return ports.Page[domain.Invitation]{}, err
	}
	items := make([]domain.Invitation, len(recs))
	for i, v := range recs {
		items[i] = *invitationFrom(v)
	}
	count, _ := countFiltered(r.app(ctx), CollectionInvitations, "group={:group}", dbx.Params{"group": groupID})
	return page(items, req, count), nil
}
func (r *Repository) UpdateInvitation(ctx context.Context, v *domain.Invitation) error {
	rec, err := r.app(ctx).FindRecordById(CollectionInvitations, v.ID)
	if err != nil {
		return mapError(err)
	}
	writeInvitation(rec, v)
	if err = r.app(ctx).Save(rec); err != nil {
		return err
	}
	hydrateTimes(rec, &v.CreatedAt, &v.UpdatedAt)
	return nil
}

func (r *Repository) CreateSubscription(ctx context.Context, v *domain.Subscription) error {
	rec, err := newRecord(r.app(ctx), CollectionSubscriptions)
	if err != nil {
		return err
	}
	writeSubscription(rec, v)
	if err = r.app(ctx).Save(rec); err != nil {
		return err
	}
	v.ID = rec.Id
	hydrateTimes(rec, &v.CreatedAt, &v.UpdatedAt)
	return nil
}
func (r *Repository) GetSubscription(ctx context.Context, id string) (*domain.Subscription, error) {
	rec, err := r.app(ctx).FindRecordById(CollectionSubscriptions, id)
	if err != nil {
		return nil, mapError(err)
	}
	return subscriptionFrom(rec), nil
}
func (r *Repository) ListSubscriptions(ctx context.Context, groupID string, req ports.PageRequest) (ports.Page[domain.Subscription], error) {
	recs, err := listRecords(r.app(ctx), CollectionSubscriptions, "group={:group}", req, dbx.Params{"group": groupID})
	if err != nil {
		return ports.Page[domain.Subscription]{}, err
	}
	items := make([]domain.Subscription, len(recs))
	for i, v := range recs {
		items[i] = *subscriptionFrom(v)
	}
	count, _ := countFiltered(r.app(ctx), CollectionSubscriptions, "group={:group}", dbx.Params{"group": groupID})
	return page(items, req, count), nil
}
func (r *Repository) ListPersonalSubscriptions(ctx context.Context, userID string, req ports.PageRequest) (ports.Page[domain.Subscription], error) {
	recs, err := listRecords(r.app(ctx), CollectionSubscriptions, "owner={:user} || paid_by={:user}", req, dbx.Params{"user": userID})
	if err != nil { return ports.Page[domain.Subscription]{}, err }
	items := make([]domain.Subscription, len(recs))
	for i, v := range recs { items[i] = *subscriptionFrom(v) }
	count, _ := countFiltered(r.app(ctx), CollectionSubscriptions, "owner={:user} || paid_by={:user}", dbx.Params{"user": userID})
	return page(items, req, count), nil
}
func (r *Repository) UpdateSubscription(ctx context.Context, v *domain.Subscription) error {
	rec, err := r.app(ctx).FindRecordById(CollectionSubscriptions, v.ID)
	if err != nil {
		return mapError(err)
	}
	writeSubscription(rec, v)
	if err = r.app(ctx).Save(rec); err != nil {
		return err
	}
	hydrateTimes(rec, &v.CreatedAt, &v.UpdatedAt)
	return nil
}
func (r *Repository) DeleteSubscription(ctx context.Context, id string) error {
	rec, err := r.app(ctx).FindRecordById(CollectionSubscriptions, id)
	if err != nil {
		return mapError(err)
	}
	return r.app(ctx).Delete(rec)
}

func (r *Repository) CreateExpense(ctx context.Context, v *domain.Expense) error {
	rec, err := newRecord(r.app(ctx), CollectionExpenses)
	if err != nil {
		return err
	}
	writeExpense(rec, v)
	if err = r.app(ctx).Save(rec); err != nil {
		return err
	}
	v.ID = rec.Id
	hydrateTimes(rec, &v.CreatedAt, &v.UpdatedAt)
	return nil
}
func (r *Repository) GetExpense(ctx context.Context, id string) (*domain.Expense, error) {
	rec, err := r.app(ctx).FindRecordById(CollectionExpenses, id)
	if err != nil {
		return nil, mapError(err)
	}
	return expenseFrom(rec), nil
}
func (r *Repository) ListExpenses(ctx context.Context, groupID string, req ports.PageRequest) (ports.Page[domain.Expense], error) {
	recs, err := listRecords(r.app(ctx), CollectionExpenses, "group={:group}", req, dbx.Params{"group": groupID})
	if err != nil {
		return ports.Page[domain.Expense]{}, err
	}
	items := make([]domain.Expense, len(recs))
	for i, v := range recs {
		items[i] = *expenseFrom(v)
	}
	count, _ := countFiltered(r.app(ctx), CollectionExpenses, "group={:group}", dbx.Params{"group": groupID})
	return page(items, req, count), nil
}
func (r *Repository) ListPersonalExpenses(ctx context.Context, userID string, req ports.PageRequest) (ports.Page[domain.Expense], error) {
	recs, err := listRecords(r.app(ctx), CollectionExpenses, "owner={:user} || paid_by={:user}", req, dbx.Params{"user": userID})
	if err != nil { return ports.Page[domain.Expense]{}, err }
	items := make([]domain.Expense, len(recs))
	for i, v := range recs { items[i] = *expenseFrom(v) }
	count, _ := countFiltered(r.app(ctx), CollectionExpenses, "owner={:user} || paid_by={:user}", dbx.Params{"user": userID})
	return page(items, req, count), nil
}
func (r *Repository) UpdateExpense(ctx context.Context, v *domain.Expense) error {
	rec, err := r.app(ctx).FindRecordById(CollectionExpenses, v.ID)
	if err != nil {
		return mapError(err)
	}
	writeExpense(rec, v)
	if err = r.app(ctx).Save(rec); err != nil {
		return err
	}
	hydrateTimes(rec, &v.CreatedAt, &v.UpdatedAt)
	return nil
}
func (r *Repository) DeleteExpense(ctx context.Context, id string) error {
	rec, err := r.app(ctx).FindRecordById(CollectionExpenses, id)
	if err != nil {
		return mapError(err)
	}
	return r.app(ctx).Delete(rec)
}

func (r *Repository) GetUser(ctx context.Context, id string) (*domain.User, error) {
	rec, err := r.app(ctx).FindRecordById("users", id)
	if err != nil {
		return nil, mapError(err)
	}
	return userFrom(rec), nil
}
func (r *Repository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	rec, err := r.app(ctx).FindAuthRecordByEmail("users", email)
	if err != nil {
		return nil, mapError(err)
	}
	return userFrom(rec), nil
}

func newRecord(app core.App, name string) (*core.Record, error) {
	c, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		return nil, err
	}
	return core.NewRecord(c), nil
}
func mapError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}
func normalizePage(req ports.PageRequest) ports.PageRequest {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 {
		req.PerPage = 20
	}
	if req.PerPage > 100 {
		req.PerPage = 100
	}
	return req
}
func listRecords(app core.App, c, filter string, req ports.PageRequest, params dbx.Params) ([]*core.Record, error) {
	req = normalizePage(req)
	sort := req.Sort
	if sort == "" {
		sort = "-created"
	}
	return app.FindRecordsByFilter(c, filter, sort, req.PerPage, (req.Page-1)*req.PerPage, params)
}
func countFiltered(app core.App, c, filter string, params dbx.Params) (int, error) {
	records, err := app.FindRecordsByFilter(c, filter, "", 0, 0, params)
	return len(records), err
}
func page[T any](items []T, req ports.PageRequest, total int) ports.Page[T] {
	req = normalizePage(req)
	return ports.Page[T]{Items: items, Page: req.Page, PerPage: req.PerPage, TotalItems: total, TotalPages: int(math.Ceil(float64(total) / float64(req.PerPage)))}
}
func slicePage[T any](all []T, req ports.PageRequest) ports.Page[T] {
	req = normalizePage(req)
	start := (req.Page - 1) * req.PerPage
	if start > len(all) {
		start = len(all)
	}
	end := start + req.PerPage
	if end > len(all) {
		end = len(all)
	}
	return page(all[start:end], req, len(all))
}
func hydrateTimes(r *core.Record, created, updated *time.Time) {
	*created = r.GetDateTime("created").Time()
	*updated = r.GetDateTime("updated").Time()
}
func writeGroup(r *core.Record, v *domain.Group) {
	r.Set("name", v.Name)
	r.Set("description", v.Description)
	r.Set("currency", v.Currency)
	r.Set("color", v.Color)
	r.Set("owner", v.OwnerID)
}
func groupFrom(r *core.Record) *domain.Group {
	v := &domain.Group{ID: r.Id, Name: r.GetString("name"), Description: r.GetString("description"), Currency: domain.Currency(r.GetString("currency")), Color: r.GetString("color"), OwnerID: r.GetString("owner")}
	hydrateTimes(r, &v.CreatedAt, &v.UpdatedAt)
	return v
}
func membershipFrom(r *core.Record) domain.Membership {
	return domain.Membership{ID: r.Id, GroupID: r.GetString("group"), UserID: r.GetString("user"), Role: domain.MemberRole(r.GetString("role")), CreatedAt: r.GetDateTime("created").Time()}
}
func writeInvitation(r *core.Record, v *domain.Invitation) {
	r.Set("group", v.GroupID)
	r.Set("email", v.Email)
	r.Set("token_hash", v.TokenHash)
	r.Set("status", v.Status)
	r.Set("invited_by", v.InvitedBy)
	r.Set("accepted_by", v.AcceptedBy)
	r.Set("expires_at", v.ExpiresAt)
}
func invitationFrom(r *core.Record) *domain.Invitation {
	v := &domain.Invitation{ID: r.Id, GroupID: r.GetString("group"), Email: r.GetString("email"), TokenHash: r.GetString("token_hash"), Status: domain.InvitationStatus(r.GetString("status")), InvitedBy: r.GetString("invited_by"), AcceptedBy: r.GetString("accepted_by"), ExpiresAt: r.GetDateTime("expires_at").Time()}
	hydrateTimes(r, &v.CreatedAt, &v.UpdatedAt)
	return v
}
func writeSubscription(r *core.Record, v *domain.Subscription) {
	r.Set("group", v.GroupID)
	r.Set("owner", v.OwnerID)
	r.Set("paid_by", v.PaidBy)
	r.Set("name", v.Name)
	r.Set("category", v.Category)
	r.Set("amount_minor", v.AmountMinor)
	r.Set("currency", v.Currency)
	r.Set("billing_cycle", v.BillingCycle)
	if v.StartsOn.IsZero() { v.StartsOn = v.NextBilling }
	r.Set("starts_on", v.StartsOn)
	if v.EndsOn != nil { r.Set("ends_on", *v.EndsOn) }
	r.Set("next_billing", v.NextBilling)
	r.Set("status", v.Status)
	r.Set("notes", v.Notes)
}
func subscriptionFrom(r *core.Record) *domain.Subscription {
	v := &domain.Subscription{ID: r.Id, GroupID: r.GetString("group"), OwnerID: r.GetString("owner"), PaidBy: r.GetString("paid_by"), Name: r.GetString("name"), Category: r.GetString("category"), AmountMinor: int64(r.GetFloat("amount_minor")), Currency: domain.Currency(r.GetString("currency")), BillingCycle: domain.BillingCycle(r.GetString("billing_cycle")), StartsOn: r.GetDateTime("starts_on").Time(), NextBilling: r.GetDateTime("next_billing").Time(), Status: domain.SubscriptionStatus(r.GetString("status")), Notes: r.GetString("notes")}
	if ends := r.GetDateTime("ends_on").Time(); !ends.IsZero() { v.EndsOn = &ends }
	hydrateTimes(r, &v.CreatedAt, &v.UpdatedAt)
	return v
}
func writeExpense(r *core.Record, v *domain.Expense) {
	r.Set("group", v.GroupID)
	r.Set("owner", v.OwnerID)
	r.Set("title", v.Title)
	r.Set("category", v.Category)
	r.Set("amount_minor", v.AmountMinor)
	r.Set("paid_by", v.PaidBy)
	r.Set("incurred_on", v.IncurredOn)
	r.Set("split_mode", v.SplitMode)
	r.Set("notes", v.Notes)
}
func expenseFrom(r *core.Record) *domain.Expense {
	v := &domain.Expense{ID: r.Id, GroupID: r.GetString("group"), OwnerID: r.GetString("owner"), Title: r.GetString("title"), Category: r.GetString("category"), AmountMinor: int64(r.GetFloat("amount_minor")), PaidBy: r.GetString("paid_by"), IncurredOn: r.GetDateTime("incurred_on").Time(), SplitMode: domain.SplitMode(r.GetString("split_mode")), Notes: r.GetString("notes")}
	hydrateTimes(r, &v.CreatedAt, &v.UpdatedAt)
	return v
}
func userFrom(r *core.Record) *domain.User {
	return &domain.User{ID: r.Id, Email: r.Email(), Name: r.GetString("name"), Avatar: r.GetString("avatar"), Timezone: r.GetString("timezone")}
}
