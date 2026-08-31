package pocketbase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

func (r *Repository) CreateIncome(ctx context.Context, v *domain.Income) error {
	rec, err := newRecord(r.app(ctx), CollectionIncomes)
	if err != nil {
		return err
	}
	writeIncome(rec, v)
	if err = r.app(ctx).Save(rec); err != nil {
		return err
	}
	v.ID = rec.Id
	hydrateTimes(rec, &v.CreatedAt, &v.UpdatedAt)
	return nil
}
func (r *Repository) GetIncome(ctx context.Context, id string) (*domain.Income, error) {
	rec, err := r.app(ctx).FindRecordById(CollectionIncomes, id)
	if err != nil {
		return nil, mapError(err)
	}
	return incomeFrom(rec), nil
}
func (r *Repository) ListPersonalIncomes(ctx context.Context, userID string, req ports.PageRequest) (ports.Page[domain.Income], error) {
	filter := "owner={:user}"
	params := dbx.Params{"user": userID}
	recs, err := listRecords(r.app(ctx), CollectionIncomes, filter, req, params)
	if err != nil {
		return ports.Page[domain.Income]{}, err
	}
	items := make([]domain.Income, len(recs))
	for i, rec := range recs {
		items[i] = *incomeFrom(rec)
	}
	count, err := countFiltered(r.app(ctx), CollectionIncomes, filter, params)
	if err != nil {
		return ports.Page[domain.Income]{}, err
	}
	return page(items, req, count), nil
}
func (r *Repository) ListIncomes(ctx context.Context, groupID string, req ports.PageRequest) (ports.Page[domain.Income], error) {
	filter := "group={:group}"
	params := dbx.Params{"group": groupID}
	recs, err := listRecords(r.app(ctx), CollectionIncomes, filter, req, params)
	if err != nil {
		return ports.Page[domain.Income]{}, err
	}
	items := make([]domain.Income, len(recs))
	for i, rec := range recs {
		items[i] = *incomeFrom(rec)
	}
	count, err := countFiltered(r.app(ctx), CollectionIncomes, filter, params)
	if err != nil {
		return ports.Page[domain.Income]{}, err
	}
	return page(items, req, count), nil
}
func (r *Repository) ListIncomesBetween(ctx context.Context, groupID string, from, to time.Time) ([]domain.Income, error) {
	filter := "group={:group} && received_on>={:from} && received_on<{:to}"
	params := dbx.Params{"group": groupID, "from": from, "to": to}
	recs, err := r.app(ctx).FindRecordsByFilter(CollectionIncomes, filter, "received_on", 0, 0, params)
	if err != nil {
		return nil, err
	}
	items := make([]domain.Income, len(recs))
	for i, rec := range recs {
		items[i] = *incomeFrom(rec)
	}
	return items, nil
}
func (r *Repository) ListGroupExpensesBetween(ctx context.Context, groupID string, from, to time.Time) ([]domain.Expense, error) {
	filter := "group={:group} && incurred_on>={:from} && incurred_on<{:to}"
	params := dbx.Params{"group": groupID, "from": from, "to": to}
	recs, err := r.app(ctx).FindRecordsByFilter(CollectionExpenses, filter, "incurred_on", 0, 0, params)
	if err != nil {
		return nil, err
	}
	items := make([]domain.Expense, len(recs))
	for i, rec := range recs {
		items[i] = *expenseFrom(rec)
		items[i].Splits, err = r.ListExpenseSplits(ctx, rec.Id)
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}
func (r *Repository) ListPersonalIncomesBetween(ctx context.Context, userID string, from, to time.Time) ([]domain.Income, error) {
	filter := "owner={:user} && received_on>={:from} && received_on<{:to}"
	params := dbx.Params{"user": userID, "from": from, "to": to}
	recs, err := r.app(ctx).FindRecordsByFilter(CollectionIncomes, filter, "received_on", 0, 0, params)
	if err != nil {
		return nil, err
	}
	items := make([]domain.Income, len(recs))
	for i, rec := range recs {
		items[i] = *incomeFrom(rec)
	}
	return items, nil
}
func (r *Repository) UpdateIncome(ctx context.Context, v *domain.Income) error {
	rec, err := r.app(ctx).FindRecordById(CollectionIncomes, v.ID)
	if err != nil {
		return mapError(err)
	}
	writeIncome(rec, v)
	if err = r.app(ctx).Save(rec); err != nil {
		return err
	}
	hydrateTimes(rec, &v.CreatedAt, &v.UpdatedAt)
	return nil
}
func (r *Repository) ReassignIncomeUser(ctx context.Context, groupID, fromUserID, toUserID string) error {
	records, err := r.app(ctx).FindRecordsByFilter(CollectionIncomes, "group={:group}", "", 0, 0, dbx.Params{"group": groupID})
	if err != nil {
		return mapError(err)
	}
	for _, record := range records {
		changed := false
		if record.GetString("earned_by") == fromUserID {
			record.Set("earned_by", toUserID)
			changed = true
		}
		var splits []*domain.IncomeSplit
		if raw := record.GetString("splits"); raw != "" {
			if err = json.Unmarshal([]byte(raw), &splits); err != nil {
				return err
			}
		}
		if len(splits) > 0 {
			merged := make([]*domain.IncomeSplit, 0, len(splits))
			byUser := make(map[string]*domain.IncomeSplit, len(splits))
			for _, split := range splits {
				if split == nil {
					continue
				}
				if split.UserID == fromUserID {
					split.UserID = toUserID
					changed = true
				}
				if existing := byUser[split.UserID]; existing != nil {
					existing.AmountMinor += split.AmountMinor
					existing.BaseAmountMinor += split.BaseAmountMinor
					existing.PercentageBasisPoints += split.PercentageBasisPoints
					changed = true
					continue
				}
				byUser[split.UserID] = split
				merged = append(merged, split)
			}
			if changed {
				record.Set("splits", merged)
			}
		}
		if changed {
			if err = r.app(ctx).Save(record); err != nil {
				return err
			}
		}
	}
	return nil
}
func (r *Repository) DeleteIncome(ctx context.Context, id string) error {
	rec, err := r.app(ctx).FindRecordById(CollectionIncomes, id)
	if err != nil {
		return mapError(err)
	}
	return r.app(ctx).Delete(rec)
}
func writeIncome(r *core.Record, v *domain.Income) {
	r.Set("group", v.GroupID)
	r.Set("owner", v.OwnerID)
	r.Set("earned_by", v.EarnedBy)
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
	r.Set("received_on", v.ReceivedOn)
	r.Set("split_mode", v.SplitMode)
	r.Set("splits", v.Splits)
	r.Set("notes", v.Notes)
}
func incomeFrom(r *core.Record) *domain.Income {
	v := &domain.Income{ID: r.Id, GroupID: r.GetString("group"), OwnerID: r.GetString("owner"), EarnedBy: r.GetString("earned_by"), Title: r.GetString("title"), Category: r.GetString("category"), CategoryID: r.GetString("category_ref"), AmountMinor: int64(r.GetFloat("amount_minor")), Currency: domain.Currency(r.GetString("currency")), BaseCurrency: domain.Currency(r.GetString("base_currency")), BaseAmountMinor: int64(r.GetFloat("base_amount_minor")), RateScaled: int64(r.GetFloat("exchange_rate_scaled")), ExchangeRateDate: r.GetDateTime("exchange_rate_date").Time(), RateMode: domain.RateMode(r.GetString("rate_mode")), SplitMode: domain.SplitMode(r.GetString("split_mode")), ReceivedOn: r.GetDateTime("received_on").Time(), Notes: r.GetString("notes")}
	_ = json.Unmarshal([]byte(r.GetString("splits")), &v.Splits)
	v.ExchangeRate = domain.FormatRate(v.RateScaled)
	if v.Currency == "" {
		v.Currency = domain.CurrencyTWD
	}
	hydrateTimes(r, &v.CreatedAt, &v.UpdatedAt)
	return v
}

func (r *Repository) ListPersonalExpensesBetween(ctx context.Context, userID string, from, to time.Time) ([]domain.Expense, error) {
	filter := `(owner={:user} || paid_by={:user}) && incurred_on>={:from} && incurred_on<{:to}`
	params := dbx.Params{"user": userID, "from": from, "to": to}
	recs, err := r.app(ctx).FindRecordsByFilter(CollectionExpenses, filter, "incurred_on", 0, 0, params)
	if err != nil {
		return nil, err
	}
	items := make([]domain.Expense, len(recs))
	for i, rec := range recs {
		items[i] = *expenseFrom(rec)
	}
	return items, nil
}
