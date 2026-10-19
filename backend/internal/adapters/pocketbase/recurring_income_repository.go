package pocketbase

import (
	"context"
	"encoding/json"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"subflow/internal/domain"
	"subflow/internal/ports"
	"time"
)

func setJSON(record *core.Record, key string, value any) {
	b, _ := json.Marshal(value)
	var out any
	_ = json.Unmarshal(b, &out)
	record.Set(key, out)
}
func readJSON(record *core.Record, key string, target any) {
	b, _ := json.Marshal(record.Get(key))
	_ = json.Unmarshal(b, target)
}
func recurringIncomeFrom(record *core.Record) *domain.RecurringIncome {
	v := &domain.RecurringIncome{ID: record.Id, OwnerID: record.GetString("owner"), GroupID: record.GetString("group"), Timezone: record.GetString("timezone"), Cycle: domain.BillingCycle(record.GetString("cycle")), Interval: int(record.GetInt("interval")), StartsOn: record.GetDateTime("starts_on").Time(), NextOn: record.GetDateTime("next_on").Time(), PostingMode: record.GetString("posting_mode"), Status: record.GetString("status")}
	if !record.GetDateTime("ends_on").IsZero() {
		x := record.GetDateTime("ends_on").Time()
		v.EndsOn = &x
	}
	readJSON(record, "template", &v.Template)
	hydrateTimes(record, &v.CreatedAt, &v.UpdatedAt)
	return v
}
func writeRecurringIncome(record *core.Record, v *domain.RecurringIncome) {
	record.Set("owner", v.OwnerID)
	record.Set("group", v.GroupID)
	setJSON(record, "template", v.Template)
	record.Set("timezone", v.Timezone)
	record.Set("cycle", v.Cycle)
	record.Set("interval", v.Interval)
	record.Set("starts_on", v.StartsOn)
	if v.EndsOn == nil {
		record.Set("ends_on", time.Time{})
	} else {
		record.Set("ends_on", *v.EndsOn)
	}
	record.Set("next_on", v.NextOn)
	record.Set("posting_mode", v.PostingMode)
	record.Set("status", v.Status)
}
func (r *Repository) CreateRecurringIncome(ctx context.Context, v *domain.RecurringIncome) error {
	record, err := newRecord(r.app(ctx), CollectionRecurringIncomes)
	if err != nil {
		return err
	}
	writeRecurringIncome(record, v)
	if err = r.app(ctx).Save(record); err != nil {
		return err
	}
	v.ID = record.Id
	hydrateTimes(record, &v.CreatedAt, &v.UpdatedAt)
	return nil
}
func (r *Repository) GetRecurringIncome(ctx context.Context, id string) (*domain.RecurringIncome, error) {
	record, err := r.app(ctx).FindRecordById(CollectionRecurringIncomes, id)
	if err != nil {
		return nil, mapError(err)
	}
	return recurringIncomeFrom(record), nil
}
func (r *Repository) UpdateRecurringIncome(ctx context.Context, v *domain.RecurringIncome) error {
	record, err := r.app(ctx).FindRecordById(CollectionRecurringIncomes, v.ID)
	if err != nil {
		return mapError(err)
	}
	writeRecurringIncome(record, v)
	if err = r.app(ctx).Save(record); err != nil {
		return err
	}
	hydrateTimes(record, &v.CreatedAt, &v.UpdatedAt)
	return nil
}
func (r *Repository) ListRecurringIncomes(ctx context.Context, owner, group string, request ports.PageRequest) (ports.Page[domain.RecurringIncome], error) {
	filter := "owner={:owner}"
	params := dbx.Params{"owner": owner}
	if group != "" {
		filter = "group={:group}"
		params = dbx.Params{"group": group}
	}
	records, err := listRecords(r.app(ctx), CollectionRecurringIncomes, filter, request, params)
	if err != nil {
		return ports.Page[domain.RecurringIncome]{}, err
	}
	count, err := countFiltered(r.app(ctx), CollectionRecurringIncomes, filter, params)
	if err != nil {
		return ports.Page[domain.RecurringIncome]{}, err
	}
	values := make([]domain.RecurringIncome, len(records))
	for i, x := range records {
		values[i] = *recurringIncomeFrom(x)
	}
	return page(values, request, count), nil
}
func (r *Repository) ListDueRecurringIncomes(ctx context.Context, before time.Time) ([]domain.RecurringIncome, error) {
	bound, err := types.ParseDateTime(before)
	if err != nil {
		return nil, err
	}
	records, err := r.app(ctx).FindRecordsByFilter(CollectionRecurringIncomes, "status='active' && next_on<={:before}", "next_on", 0, 0, dbx.Params{"before": bound})
	if err != nil {
		return nil, mapError(err)
	}
	values := make([]domain.RecurringIncome, len(records))
	for i, x := range records {
		values[i] = *recurringIncomeFrom(x)
	}
	return values, nil
}
func (r *Repository) CreateRecurringIncomeRevision(ctx context.Context, v *domain.RecurringIncomeRevision) error {
	record, err := newRecord(r.app(ctx), CollectionRecurringIncomeRevisions)
	if err != nil {
		return err
	}
	record.Set("recurring_income", v.RecurringIncomeID)
	setJSON(record, "snapshot", v)
	record.Set("effective_on", v.EffectiveOn)
	if v.Through == nil {
		record.Set("through", time.Time{})
	} else {
		record.Set("through", *v.Through)
	}
	record.Set("scope", v.Scope)
	if err = r.app(ctx).Save(record); err != nil {
		return err
	}
	v.ID = record.Id
	hydrateTimes(record, &v.CreatedAt, new(time.Time))
	return nil
}
func (r *Repository) ListRecurringIncomeRevisions(ctx context.Context, id string) ([]domain.RecurringIncomeRevision, error) {
	records, err := r.app(ctx).FindRecordsByFilter(CollectionRecurringIncomeRevisions, "recurring_income={:id}", "effective_on,created", 0, 0, dbx.Params{"id": id})
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]domain.RecurringIncomeRevision, len(records))
	for i, x := range records {
		readJSON(x, "snapshot", &out[i])
		out[i].ID = x.Id
		out[i].RecurringIncomeID = x.GetString("recurring_income")
		out[i].EffectiveOn = x.GetDateTime("effective_on").Time()
		out[i].Scope = x.GetString("scope")
		if !x.GetDateTime("through").IsZero() {
			v := x.GetDateTime("through").Time()
			out[i].Through = &v
		}
		hydrateTimes(x, &out[i].CreatedAt, new(time.Time))
	}
	return out, nil
}
func (r *Repository) SaveIncomeOccurrence(ctx context.Context, v *domain.IncomeOccurrence) error {
	app := r.app(ctx)
	var record *core.Record
	var err error
	if v.ID != "" {
		record, err = app.FindRecordById(CollectionIncomeOccurrences, v.ID)
	} else {
		record, err = newRecord(app, CollectionIncomeOccurrences)
	}
	if err != nil {
		return err
	}
	record.Set("recurring_income", v.RecurringIncomeID)
	record.Set("revision", v.RevisionID)
	record.Set("income", v.IncomeID)
	record.Set("scheduled_on", v.ScheduledOn)
	record.Set("status", v.Status)
	record.Set("posting_mode", v.PostingMode)
	setJSON(record, "snapshot", v.Template)
	record.Set("manual_override", v.ManualOverride)
	record.Set("reason", v.Reason)
	if err = app.Save(record); err != nil {
		return mapError(err)
	}
	v.ID = record.Id
	hydrateTimes(record, &v.CreatedAt, &v.UpdatedAt)
	return nil
}
func (r *Repository) GetIncomeOccurrence(ctx context.Context, id string, on time.Time) (*domain.IncomeOccurrence, error) {
	date, err := types.ParseDateTime(on)
	if err != nil {
		return nil, err
	}
	record, err := r.app(ctx).FindFirstRecordByFilter(CollectionIncomeOccurrences, "recurring_income={:id} && scheduled_on={:on}", dbx.Params{"id": id, "on": date})
	if err != nil {
		return nil, mapError(err)
	}
	return incomeOccurrenceFrom(record), nil
}
func (r *Repository) FindIncomeOccurrenceByIncome(ctx context.Context, id string) (*domain.IncomeOccurrence, error) {
	record, err := r.app(ctx).FindFirstRecordByFilter(CollectionIncomeOccurrences, "income={:id}", dbx.Params{"id": id})
	if err != nil {
		return nil, mapError(err)
	}
	return incomeOccurrenceFrom(record), nil
}
func incomeOccurrenceFrom(record *core.Record) *domain.IncomeOccurrence {
	v := &domain.IncomeOccurrence{ID: record.Id, RecurringIncomeID: record.GetString("recurring_income"), RevisionID: record.GetString("revision"), IncomeID: record.GetString("income"), ScheduledOn: record.GetDateTime("scheduled_on").Time(), Status: record.GetString("status"), PostingMode:record.GetString("posting_mode"), ManualOverride: record.GetBool("manual_override"), Reason: record.GetString("reason")}
	readJSON(record, "snapshot", &v.Template)
	hydrateTimes(record, &v.CreatedAt, &v.UpdatedAt)
	return v
}
func (r *Repository) ListIncomeOccurrences(ctx context.Context, id string, request ports.PageRequest) (ports.Page[domain.IncomeOccurrence], error) {
	records, err := listRecords(r.app(ctx), CollectionIncomeOccurrences, "recurring_income={:id}", request, dbx.Params{"id": id})
	if err != nil {
		return ports.Page[domain.IncomeOccurrence]{}, err
	}
	count, err := countFiltered(r.app(ctx), CollectionIncomeOccurrences, "recurring_income={:id}", dbx.Params{"id": id})
	if err != nil {
		return ports.Page[domain.IncomeOccurrence]{}, err
	}
	out := make([]domain.IncomeOccurrence, len(records))
	for i, x := range records {
		out[i] = *incomeOccurrenceFrom(x)
	}
	return page(out, request, count), nil
}
func (r *Repository) ReassignRecurringIncomeUser(ctx context.Context, group, from, to string) error {
	records, err := r.app(ctx).FindRecordsByFilter(CollectionRecurringIncomes, "group={:group}", "", 0, 0, dbx.Params{"group": group})
	if err != nil {
		return err
	}
	for _, record := range records {
		value := recurringIncomeFrom(record)
		if value.Template.EarnedBy == from {
			value.Template.EarnedBy = to
		}
		for _, split := range value.Template.Splits {
			if split.UserID == from {
				split.UserID = to
			}
		}
		if err = r.UpdateRecurringIncome(ctx, value); err != nil {
			return err
		}
	}
	return nil
}
