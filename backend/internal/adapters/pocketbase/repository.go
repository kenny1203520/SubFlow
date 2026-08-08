package pocketbase

import (
	"context"
	"database/sql"
	"encoding/json"
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
	record.Set("role_ref", m.RoleID)
	if err := r.app(ctx).Save(record); err != nil {
		return err
	}
	m.ID = record.Id
	m.CreatedAt = record.GetDateTime("created").Time()
	return nil
}
func (r *Repository) UpdateMembershipRole(ctx context.Context, groupID, userID, roleID string) error {
	record, err := r.app(ctx).FindFirstRecordByFilter(CollectionMembers, "group={:group} && user={:user}", dbx.Params{"group": groupID, "user": userID})
	if err != nil {
		return mapError(err)
	}
	record.Set("role_ref", roleID)
	return r.app(ctx).Save(record)
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
	if err != nil {
		return ports.Page[domain.Subscription]{}, err
	}
	items := make([]domain.Subscription, len(recs))
	for i, v := range recs {
		items[i] = *subscriptionFrom(v)
	}
	count, _ := countFiltered(r.app(ctx), CollectionSubscriptions, "owner={:user} || paid_by={:user}", dbx.Params{"user": userID})
	return page(items, req, count), nil
}

func (r *Repository) ListAutomaticSubscriptions(ctx context.Context) ([]domain.Subscription, error) {
	records, err := r.app(ctx).FindRecordsByFilter(CollectionSubscriptions, "rate_mode='automatic' && `group`!=''", "next_billing", 0, 0, nil)
	if err != nil {
		return nil, mapError(err)
	}
	values := make([]domain.Subscription, len(records))
	for i, record := range records {
		values[i] = *subscriptionFrom(record)
	}
	return values, nil
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
	if err != nil {
		return ports.Page[domain.Expense]{}, err
	}
	items := make([]domain.Expense, len(recs))
	for i, v := range recs {
		items[i] = *expenseFrom(v)
	}
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

func (r *Repository) ReplaceExpenseSplits(ctx context.Context, expenseID string, values []domain.ExpenseSplit) error {
	records, err := r.app(ctx).FindRecordsByFilter(CollectionExpenseSplits, "expense={:expense}", "", 0, 0, dbx.Params{"expense": expenseID})
	if err != nil {
		return err
	}
	for _, record := range records {
		if err = r.app(ctx).Delete(record); err != nil {
			return err
		}
	}
	for i := range values {
		record, createErr := newRecord(r.app(ctx), CollectionExpenseSplits)
		if createErr != nil {
			return createErr
		}
		record.Set("expense", expenseID)
		record.Set("user", values[i].UserID)
		record.Set("amount_minor", values[i].AmountMinor)
		record.Set("base_amount_minor", values[i].BaseAmountMinor)
		record.Set("percentage_bp", values[i].PercentageBasisPoints)
		if err = r.app(ctx).Save(record); err != nil {
			return err
		}
		values[i].ID = record.Id
		values[i].ExpenseID = expenseID
	}
	return nil
}

func (r *Repository) ListExpenseSplits(ctx context.Context, expenseID string) ([]domain.ExpenseSplit, error) {
	records, err := r.app(ctx).FindRecordsByFilter(CollectionExpenseSplits, "expense={:expense}", "user", 0, 0, dbx.Params{"expense": expenseID})
	if err != nil {
		return nil, err
	}
	result := make([]domain.ExpenseSplit, len(records))
	for i, record := range records {
		result[i] = domain.ExpenseSplit{ID: record.Id, ExpenseID: expenseID, UserID: record.GetString("user"), AmountMinor: int64(record.GetFloat("amount_minor")), BaseAmountMinor: int64(record.GetFloat("base_amount_minor")), PercentageBasisPoints: int(record.GetFloat("percentage_bp"))}
	}
	return result, nil
}

func (r *Repository) CreateSettlement(ctx context.Context, v *domain.Settlement) error {
	record, err := newRecord(r.app(ctx), CollectionSettlements)
	if err != nil {
		return err
	}
	writeSettlement(record, v)
	if err = r.app(ctx).Save(record); err != nil {
		return err
	}
	v.ID = record.Id
	hydrateTimes(record, &v.CreatedAt, &v.UpdatedAt)
	return nil
}
func (r *Repository) GetSettlement(ctx context.Context, id string) (*domain.Settlement, error) {
	record, err := r.app(ctx).FindRecordById(CollectionSettlements, id)
	if err != nil {
		return nil, mapError(err)
	}
	return settlementFrom(record), nil
}
func (r *Repository) ListSettlements(ctx context.Context, groupID string, req ports.PageRequest) (ports.Page[domain.Settlement], error) {
	records, err := listRecords(r.app(ctx), CollectionSettlements, "group={:group}", req, dbx.Params{"group": groupID})
	if err != nil {
		return ports.Page[domain.Settlement]{}, err
	}
	items := make([]domain.Settlement, len(records))
	for i, record := range records {
		items[i] = *settlementFrom(record)
	}
	count, _ := countFiltered(r.app(ctx), CollectionSettlements, "group={:group}", dbx.Params{"group": groupID})
	return page(items, req, count), nil
}
func (r *Repository) DeleteSettlement(ctx context.Context, id string) error {
	record, err := r.app(ctx).FindRecordById(CollectionSettlements, id)
	if err != nil {
		return mapError(err)
	}
	return r.app(ctx).Delete(record)
}

func (r *Repository) UpdateSettlement(ctx context.Context, v *domain.Settlement) error {
	record, err := r.app(ctx).FindRecordById(CollectionSettlements, v.ID)
	if err != nil {
		return mapError(err)
	}
	writeSettlement(record, v)
	if err = r.app(ctx).Save(record); err != nil {
		return err
	}
	hydrateTimes(record, &v.CreatedAt, &v.UpdatedAt)
	return nil
}

func (r *Repository) CreateCategory(ctx context.Context, v *domain.Category) error {
	record, err := newRecord(r.app(ctx), CollectionCategories)
	if err != nil {
		return err
	}
	writeCategory(record, v)
	if err = r.app(ctx).Save(record); err != nil {
		return err
	}
	v.ID = record.Id
	hydrateTimes(record, &v.CreatedAt, &v.UpdatedAt)
	return nil
}
func (r *Repository) GetCategory(ctx context.Context, id string) (*domain.Category, error) {
	record, err := r.app(ctx).FindRecordById(CollectionCategories, id)
	if err != nil {
		return nil, mapError(err)
	}
	return categoryFrom(record), nil
}
func (r *Repository) ListCategories(ctx context.Context, ownerID, groupID string, archived bool) ([]domain.Category, error) {
	filter := "scope='system'"
	params := dbx.Params{}
	if groupID != "" {
		// PocketBase filter identifiers are not SQL identifiers. Quoting the
		// relation field with backticks makes the filter parser reject valid
		// group category requests and previously surfaced as a 500 response.
		filter += " || (scope='group' && group={:group})"
		params["group"] = groupID
	} else {
		filter += " || (scope='personal' && owner={:owner})"
		params["owner"] = ownerID
	}
	if !archived {
		filter = "(" + filter + ") && archived=false"
	}
	records, err := r.app(ctx).FindRecordsByFilter(CollectionCategories, filter, "scope,custom_name,system_key", 0, 0, params)
	if err != nil {
		return nil, mapError(err)
	}
	result := make([]domain.Category, len(records))
	for i, record := range records {
		result[i] = *categoryFrom(record)
	}
	return result, nil
}
func (r *Repository) UpdateCategory(ctx context.Context, v *domain.Category) error {
	record, err := r.app(ctx).FindRecordById(CollectionCategories, v.ID)
	if err != nil {
		return mapError(err)
	}
	writeCategory(record, v)
	if err = r.app(ctx).Save(record); err != nil {
		return err
	}
	hydrateTimes(record, &v.CreatedAt, &v.UpdatedAt)
	return nil
}

func roleCollection(scope string) string {
	if scope == "system" {
		return CollectionSystemRoles
	}
	return CollectionGroupRoles
}
func writeRole(record *core.Record, v *domain.Role) {
	record.Set("group", v.GroupID)
	record.Set("name", v.Name)
	record.Set("key", v.Key)
	record.Set("permissions", v.Permissions)
	record.Set("protected", v.Protected)
	record.Set("created_by", v.CreatedBy)
}
func roleFrom(record *core.Record, scope string) *domain.Role {
	v := &domain.Role{ID: record.Id, Scope: scope, GroupID: record.GetString("group"), Name: record.GetString("name"), Key: record.GetString("key"), Protected: record.GetBool("protected"), CreatedBy: record.GetString("created_by")}
	_ = json.Unmarshal([]byte(record.GetString("permissions")), &v.Permissions)
	hydrateTimes(record, &v.CreatedAt, &v.UpdatedAt)
	return v
}
func (r *Repository) CreateRole(ctx context.Context, v *domain.Role) error {
	record, err := newRecord(r.app(ctx), roleCollection(v.Scope))
	if err != nil {
		return err
	}
	writeRole(record, v)
	if err = r.app(ctx).Save(record); err != nil {
		return err
	}
	v.ID = record.Id
	hydrateTimes(record, &v.CreatedAt, &v.UpdatedAt)
	return nil
}
func (r *Repository) GetRoleRecord(ctx context.Context, scope, id string) (*domain.Role, error) {
	record, err := r.app(ctx).FindRecordById(roleCollection(scope), id)
	if err != nil {
		return nil, mapError(err)
	}
	return roleFrom(record, scope), nil
}
func (r *Repository) ListRoles(ctx context.Context, scope, groupID string) ([]domain.Role, error) {
	filter := ""
	params := dbx.Params{}
	if scope == "group" {
		filter = "group={:group}"
		params["group"] = groupID
	}
	records, err := r.app(ctx).FindRecordsByFilter(roleCollection(scope), filter, "key", 0, 0, params)
	if err != nil {
		return nil, mapError(err)
	}
	values := make([]domain.Role, len(records))
	for i, record := range records {
		values[i] = *roleFrom(record, scope)
	}
	return values, nil
}
func (r *Repository) UpdateRoleRecord(ctx context.Context, v *domain.Role) error {
	record, err := r.app(ctx).FindRecordById(roleCollection(v.Scope), v.ID)
	if err != nil {
		return mapError(err)
	}
	writeRole(record, v)
	if err = r.app(ctx).Save(record); err != nil {
		return err
	}
	hydrateTimes(record, &v.CreatedAt, &v.UpdatedAt)
	return nil
}
func (r *Repository) DeleteRoleRecord(ctx context.Context, scope, id string) error {
	record, err := r.app(ctx).FindRecordById(roleCollection(scope), id)
	if err != nil {
		return mapError(err)
	}
	return r.app(ctx).Delete(record)
}

func (r *Repository) CreateAudit(ctx context.Context, v *domain.AuditLog) error {
	record, err := newRecord(r.app(ctx), CollectionAuditLogs)
	if err != nil {
		return err
	}
	record.Set("actor", v.ActorID)
	record.Set("group", v.GroupID)
	record.Set("scope", v.Scope)
	record.Set("action", v.Action)
	record.Set("resource", v.Resource)
	record.Set("resource_id", v.ResourceID)
	record.Set("outcome", v.Outcome)
	record.Set("summary", v.Summary)
	record.Set("ip", v.IP)
	record.Set("user_agent", v.UserAgent)
	record.Set("hash", v.Hash)
	if err = r.app(ctx).Save(record); err != nil {
		return err
	}
	v.ID = record.Id
	v.CreatedAt = record.GetDateTime("created").Time()
	return nil
}
func (r *Repository) ListAudits(ctx context.Context, groupID string, req ports.PageRequest) (ports.Page[domain.AuditLog], error) {
	filter := ""
	params := dbx.Params{}
	if groupID != "" {
		filter = "group={:group}"
		params["group"] = groupID
	}
	records, err := listRecords(r.app(ctx), CollectionAuditLogs, filter, req, params)
	if err != nil {
		return ports.Page[domain.AuditLog]{}, err
	}
	values := make([]domain.AuditLog, len(records))
	for i, r := range records {
		values[i] = domain.AuditLog{ID: r.Id, ActorID: r.GetString("actor"), GroupID: r.GetString("group"), Scope: r.GetString("scope"), Action: r.GetString("action"), Resource: r.GetString("resource"), ResourceID: r.GetString("resource_id"), Outcome: r.GetString("outcome"), Summary: r.GetString("summary"), IP: r.GetString("ip"), UserAgent: r.GetString("user_agent"), Hash: r.GetString("hash"), CreatedAt: r.GetDateTime("created").Time()}
	}
	count, _ := countFiltered(r.app(ctx), CollectionAuditLogs, filter, params)
	return page(values, req, count), nil
}
func (r *Repository) UpsertExchangeRate(ctx context.Context, v *domain.ExchangeRate) error {
	record, err := r.app(ctx).FindFirstRecordByFilter(CollectionExchangeRates, "base_currency={:base} && quote_currency={:quote} && effective_date={:date}", dbx.Params{"base": v.BaseCurrency, "quote": v.QuoteCurrency, "date": v.EffectiveDate})
	if err != nil {
		record, err = newRecord(r.app(ctx), CollectionExchangeRates)
		if err != nil {
			return err
		}
	}
	writeExchangeRate(record, v)
	if err = r.app(ctx).Save(record); err != nil {
		return err
	}
	v.ID = record.Id
	return nil
}
func (r *Repository) LatestExchangeRate(ctx context.Context, from, to domain.Currency, date time.Time) (*domain.ExchangeRate, error) {
	if from == to {
		return &domain.ExchangeRate{BaseCurrency: from, QuoteCurrency: to, RateScaled: domain.ExchangeRateScale, Rate: "1", EffectiveDate: date, Provider: "identity", FetchedAt: time.Now()}, nil
	}
	records, err := r.app(ctx).FindRecordsByFilter(CollectionExchangeRates, "base_currency={:base} && quote_currency={:quote} && effective_date<={:date}", "-effective_date", 1, 0, dbx.Params{"base": from, "quote": to, "date": date})
	if err != nil || len(records) == 0 {
		return nil, domain.ErrRateUnavailable
	}
	return exchangeRateFrom(records[0]), nil
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
func (r *Repository) SetUserSystemRole(ctx context.Context, id, roleID string) error {
	record, err := r.app(ctx).FindRecordById("users", id)
	if err != nil {
		return mapError(err)
	}
	record.Set("system_role", roleID)
	return r.app(ctx).Save(record)
}

func (r *Repository) CreateSetupUser(ctx context.Context, input domain.SetupInput) (*domain.User, error) {
	record, err := newRecord(r.app(ctx), "users")
	if err != nil {
		return nil, err
	}
	record.Set("email", input.Email)
	record.Set("password", input.Password)
	record.Set("passwordConfirm", input.Password)
	record.Set("name", input.AdminName)
	record.Set("timezone", input.DefaultTimezone)
	record.Set("default_currency", input.DefaultCurrency)
	record.Set("verified", true)
	if err = r.app(ctx).Save(record); err != nil {
		return nil, err
	}
	return userFrom(record), nil
}

func (r *Repository) CountUsersBySystemRole(ctx context.Context, roleID string) (int, error) {
	items, err := r.app(ctx).FindRecordsByFilter("users", "system_role={:role}", "", 1, 0, dbx.Params{"role": roleID})
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func settingsFrom(record *core.Record) domain.SystemSettings {
	return domain.SystemSettings{Initialized: record.GetBool("initialized"), SiteName: record.GetString("site_name"), DefaultTimezone: record.GetString("default_timezone"), DefaultCurrency: domain.Currency(record.GetString("default_currency")), AllowRegistration: record.GetBool("allow_registration")}
}

func defaultSystemSettings() domain.SystemSettings {
	return domain.SystemSettings{SiteName: "SubFlow", DefaultTimezone: "UTC", DefaultCurrency: domain.CurrencyTWD}
}

func (r *Repository) GetSystemSettings(ctx context.Context) (domain.SystemSettings, error) {
	record, err := r.app(ctx).FindFirstRecordByFilter(CollectionSystemSettings, "key='primary'", nil)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return defaultSystemSettings(), nil
		}
		return domain.SystemSettings{}, err
	}
	value := settingsFrom(record)
	if value.SiteName == "" {
		value.SiteName = "SubFlow"
	}
	if value.DefaultTimezone == "" {
		value.DefaultTimezone = "UTC"
	}
	if value.DefaultCurrency == "" {
		value.DefaultCurrency = domain.CurrencyTWD
	}
	return value, nil
}

func (r *Repository) SaveSystemSettings(ctx context.Context, value domain.SystemSettings) error {
	record, err := r.app(ctx).FindFirstRecordByFilter(CollectionSystemSettings, "key='primary'", nil)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		record, err = newRecord(r.app(ctx), CollectionSystemSettings)
		if err != nil {
			return err
		}
		record.Set("key", "primary")
	}
	record.Set("initialized", value.Initialized)
	record.Set("site_name", value.SiteName)
	record.Set("default_timezone", value.DefaultTimezone)
	record.Set("default_currency", value.DefaultCurrency)
	record.Set("allow_registration", value.AllowRegistration)
	return r.app(ctx).Save(record)
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
	r.Set("timezone", v.Timezone)
	r.Set("color", v.Color)
	r.Set("owner", v.OwnerID)
}
func groupFrom(r *core.Record) *domain.Group {
	v := &domain.Group{ID: r.Id, Name: r.GetString("name"), Description: r.GetString("description"), Currency: domain.Currency(r.GetString("currency")), Timezone: r.GetString("timezone"), Color: r.GetString("color"), OwnerID: r.GetString("owner")}
	if v.Timezone == "" {
		v.Timezone = "UTC"
	}
	hydrateTimes(r, &v.CreatedAt, &v.UpdatedAt)
	return v
}
func membershipFrom(r *core.Record) domain.Membership {
	return domain.Membership{ID: r.Id, GroupID: r.GetString("group"), UserID: r.GetString("user"), Role: domain.MemberRole(r.GetString("role")), RoleID: r.GetString("role_ref"), CreatedAt: r.GetDateTime("created").Time()}
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
	r.Set("category_ref", v.CategoryID)
	r.Set("amount_minor", v.AmountMinor)
	r.Set("currency", v.Currency)
	r.Set("base_currency", v.BaseCurrency)
	r.Set("base_amount_minor", v.BaseAmountMinor)
	r.Set("exchange_rate_scaled", v.RateScaled)
	r.Set("exchange_rate_date", v.ExchangeRateDate)
	r.Set("rate_mode", v.RateMode)
	r.Set("billing_cycle", v.BillingCycle)
	if v.StartsOn.IsZero() {
		v.StartsOn = v.NextBilling
	}
	r.Set("starts_on", v.StartsOn)
	if v.EndsOn != nil {
		r.Set("ends_on", *v.EndsOn)
	} else {
		r.Set("ends_on", nil)
	}
	r.Set("next_billing", v.NextBilling)
	r.Set("status", v.Status)
	r.Set("notes", v.Notes)
}
func subscriptionFrom(r *core.Record) *domain.Subscription {
	v := &domain.Subscription{ID: r.Id, GroupID: r.GetString("group"), OwnerID: r.GetString("owner"), PaidBy: r.GetString("paid_by"), Name: r.GetString("name"), Category: r.GetString("category"), CategoryID: r.GetString("category_ref"), AmountMinor: int64(r.GetFloat("amount_minor")), Currency: domain.Currency(r.GetString("currency")), BaseCurrency: domain.Currency(r.GetString("base_currency")), BaseAmountMinor: int64(r.GetFloat("base_amount_minor")), RateScaled: int64(r.GetFloat("exchange_rate_scaled")), ExchangeRateDate: r.GetDateTime("exchange_rate_date").Time(), RateMode: domain.RateMode(r.GetString("rate_mode")), BillingCycle: domain.BillingCycle(r.GetString("billing_cycle")), StartsOn: r.GetDateTime("starts_on").Time(), NextBilling: r.GetDateTime("next_billing").Time(), Status: domain.SubscriptionStatus(r.GetString("status")), Notes: r.GetString("notes")}
	v.ExchangeRate = domain.FormatRate(v.RateScaled)
	if ends := r.GetDateTime("ends_on").Time(); !ends.IsZero() {
		v.EndsOn = &ends
	}
	hydrateTimes(r, &v.CreatedAt, &v.UpdatedAt)
	return v
}
func writeExpense(r *core.Record, v *domain.Expense) {
	r.Set("group", v.GroupID)
	r.Set("owner", v.OwnerID)
	r.Set("title", v.Title)
	r.Set("category", v.Category)
	r.Set("category_ref", v.CategoryID)
	r.Set("amount_minor", v.AmountMinor)
	r.Set("currency", v.Currency)
	r.Set("base_currency", v.BaseCurrency)
	r.Set("base_amount_minor", v.BaseAmountMinor)
	r.Set("exchange_rate_scaled", v.RateScaled)
	r.Set("exchange_rate_date", v.ExchangeRateDate)
	r.Set("rate_mode", v.RateMode)
	r.Set("paid_by", v.PaidBy)
	r.Set("incurred_on", v.IncurredOn)
	r.Set("split_mode", v.SplitMode)
	r.Set("notes", v.Notes)
}
func expenseFrom(r *core.Record) *domain.Expense {
	v := &domain.Expense{ID: r.Id, GroupID: r.GetString("group"), OwnerID: r.GetString("owner"), Title: r.GetString("title"), Category: r.GetString("category"), CategoryID: r.GetString("category_ref"), AmountMinor: int64(r.GetFloat("amount_minor")), Currency: domain.Currency(r.GetString("currency")), BaseCurrency: domain.Currency(r.GetString("base_currency")), BaseAmountMinor: int64(r.GetFloat("base_amount_minor")), RateScaled: int64(r.GetFloat("exchange_rate_scaled")), ExchangeRateDate: r.GetDateTime("exchange_rate_date").Time(), RateMode: domain.RateMode(r.GetString("rate_mode")), PaidBy: r.GetString("paid_by"), IncurredOn: r.GetDateTime("incurred_on").Time(), SplitMode: domain.SplitMode(r.GetString("split_mode")), Notes: r.GetString("notes")}
	v.ExchangeRate = domain.FormatRate(v.RateScaled)
	if v.Currency == "" {
		v.Currency = domain.CurrencyTWD
	}
	hydrateTimes(r, &v.CreatedAt, &v.UpdatedAt)
	return v
}
func writeSettlement(r *core.Record, v *domain.Settlement) {
	r.Set("group", v.GroupID)
	r.Set("from_user", v.FromUserID)
	r.Set("to_user", v.ToUserID)
	r.Set("created_by", v.CreatedBy)
	r.Set("amount_minor", v.AmountMinor)
	r.Set("currency", v.Currency)
	r.Set("base_currency", v.BaseCurrency)
	r.Set("base_amount_minor", v.BaseAmountMinor)
	r.Set("exchange_rate_scaled", v.RateScaled)
	r.Set("exchange_rate_date", v.ExchangeRateDate)
	r.Set("settled_on", v.SettledOn)
	r.Set("notes", v.Notes)
}
func settlementFrom(r *core.Record) *domain.Settlement {
	v := &domain.Settlement{ID: r.Id, GroupID: r.GetString("group"), FromUserID: r.GetString("from_user"), ToUserID: r.GetString("to_user"), CreatedBy: r.GetString("created_by"), AmountMinor: int64(r.GetFloat("amount_minor")), Currency: domain.Currency(r.GetString("currency")), BaseCurrency: domain.Currency(r.GetString("base_currency")), BaseAmountMinor: int64(r.GetFloat("base_amount_minor")), RateScaled: int64(r.GetFloat("exchange_rate_scaled")), ExchangeRateDate: r.GetDateTime("exchange_rate_date").Time(), SettledOn: r.GetDateTime("settled_on").Time(), Notes: r.GetString("notes")}
	v.ExchangeRate = domain.FormatRate(v.RateScaled)
	hydrateTimes(r, &v.CreatedAt, &v.UpdatedAt)
	return v
}
func userFrom(r *core.Record) *domain.User {
	return &domain.User{ID: r.Id, Email: r.Email(), Name: r.GetString("name"), Avatar: r.GetString("avatar"), Timezone: r.GetString("timezone"), DefaultCurrency: domain.Currency(r.GetString("default_currency")), SystemRoleID: r.GetString("system_role")}
}

func writeCategory(r *core.Record, v *domain.Category) {
	r.Set("scope", v.Scope)
	r.Set("owner", v.OwnerID)
	r.Set("group", v.GroupID)
	r.Set("system_key", v.SystemKey)
	r.Set("custom_name", v.CustomName)
	r.Set("icon_key", v.IconKey)
	r.Set("created_by", v.CreatedBy)
	r.Set("archived", v.Archived)
}
func categoryFrom(r *core.Record) *domain.Category {
	v := &domain.Category{ID: r.Id, Scope: r.GetString("scope"), OwnerID: r.GetString("owner"), GroupID: r.GetString("group"), SystemKey: r.GetString("system_key"), CustomName: r.GetString("custom_name"), IconKey: r.GetString("icon_key"), CreatedBy: r.GetString("created_by"), Archived: r.GetBool("archived")}
	hydrateTimes(r, &v.CreatedAt, &v.UpdatedAt)
	return v
}
func writeExchangeRate(r *core.Record, v *domain.ExchangeRate) {
	r.Set("base_currency", v.BaseCurrency)
	r.Set("quote_currency", v.QuoteCurrency)
	r.Set("rate_scaled", v.RateScaled)
	r.Set("effective_date", v.EffectiveDate)
	r.Set("provider", v.Provider)
	r.Set("fetched_at", v.FetchedAt)
}
func exchangeRateFrom(r *core.Record) *domain.ExchangeRate {
	v := &domain.ExchangeRate{ID: r.Id, BaseCurrency: domain.Currency(r.GetString("base_currency")), QuoteCurrency: domain.Currency(r.GetString("quote_currency")), RateScaled: int64(r.GetFloat("rate_scaled")), EffectiveDate: r.GetDateTime("effective_date").Time(), Provider: r.GetString("provider"), FetchedAt: r.GetDateTime("fetched_at").Time()}
	v.Rate = domain.FormatRate(v.RateScaled)
	return v
}
