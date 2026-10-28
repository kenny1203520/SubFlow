package httpapi

import (
	"ganigeorgiev.com/pocketbase/pocketbase/core"
	"net/http"
	"strings"
	"subflow/internal/domain"
	"time"
)

func recurringDate(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, domain.ErrInvalid
	}
	v, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		v, err = time.Parse("2006-01-02", raw)
	}
	if err != nil {
		return time.Time{}, domain.ErrInvalid
	}
	return v, nil
}
func (a *API) recurringGroup(e *core.RequestEvent) string {
	return strings.TrimSpace(e.Request.PathValue("groupId"))
}
func (a *API) recurringRequestFailure(e *core.RequestEvent, action string, cause error) error {
	return a.Service.RecurringIncomeRequestFailure(e.Request.Context(), authID(e), &domain.RecurringIncome{GroupID: a.recurringGroup(e)}, action, cause)
}
func (a *API) authorizeRecurringRoute(e *core.RequestEvent, action string) error {
	group := a.recurringGroup(e)
	if group == "" {
		return nil
	}
	id := e.Request.PathValue("id")
	v, err := a.Service.Stores.RecurringIncomes.Get(e.Request.Context(), id)
	if err != nil {
		return a.Service.RecurringIncomeRequestFailure(e.Request.Context(), authID(e), nil, action, err)
	}
	if v.GroupID != group {
		return a.Service.RecurringIncomeRequestFailure(e.Request.Context(), authID(e), v, action, domain.ErrForbidden)
	}
	if err = a.Service.AuthorizeRecurringIncome(e.Request.Context(), authID(e), v, action); err != nil {
		return a.Service.RecurringIncomeRequestFailure(e.Request.Context(), authID(e), v, action, err)
	}
	return nil
}
func (a *API) listRecurringIncomes(e *core.RequestEvent) error {
	group := a.recurringGroup(e)
	p, err := pageRequest(e, "incomes")
	if err != nil {
		return fail(e, err)
	}
	values, err := a.Service.ListRecurringIncomes(e.Request.Context(), authID(e), group, p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, values.Items, values)
}
func (a *API) createRecurringIncome(e *core.RequestEvent) error {
	var value domain.RecurringIncome
	if e.BindBody(&value) != nil {
		return fail(e, a.recurringRequestFailure(e, "created", domain.ErrInvalid))
	}
	if group := a.recurringGroup(e); group != "" {
		value.GroupID = group
	}
	created, err := a.Service.CreateRecurringIncome(e.Request.Context(), authID(e), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) updateRecurringIncome(e *core.RequestEvent) error {
	if err := a.authorizeRecurringRoute(e, "ledger.incomes.write"); err != nil {
		return fail(e, err)
	}
	var value domain.RecurringIncome
	if e.BindBody(&value) != nil {
		return fail(e, a.recurringRequestFailure(e, "updated", domain.ErrInvalid))
	}
	if group := a.recurringGroup(e); group != "" {
		value.GroupID = group
	}
	updated, err := a.Service.UpdateRecurringIncome(e.Request.Context(), authID(e), e.Request.PathValue("id"), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}
func (a *API) changeRecurringIncomeStatus(e *core.RequestEvent) error {
	if err := a.authorizeRecurringRoute(e, "ledger.incomes.write"); err != nil {
		return fail(e, err)
	}
	var body struct {
		Status string `json:"status"`
		EndsOn string `json:"endsOn"`
	}
	if e.BindBody(&body) != nil {
		return fail(e, a.recurringRequestFailure(e, "status_changed", domain.ErrInvalid))
	}
	var end *time.Time
	if body.EndsOn != "" {
		date, err := recurringDate(body.EndsOn)
		if err != nil {
			return fail(e, a.recurringRequestFailure(e, "status_changed", err))
		}
		end = &date
	}
	if err := a.Service.SetRecurringIncomeStatus(e.Request.Context(), authID(e), e.Request.PathValue("id"), body.Status, end); err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, map[string]bool{"ok": true}, nil)
}
func (a *API) listIncomeOccurrences(e *core.RequestEvent) error {
	if err := a.authorizeRecurringRoute(e, "ledger.incomes.read"); err != nil {
		return fail(e, err)
	}
	p, err := pageRequest(e, "incomes")
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListIncomeOccurrences(e.Request.Context(), authID(e), e.Request.PathValue("id"), p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, v)
}
func (a *API) confirmIncomeOccurrence(e *core.RequestEvent) error {
	if err := a.authorizeRecurringRoute(e, "ledger.incomes.write"); err != nil {
		return fail(e, err)
	}
	var body struct {
		ScheduledOn string        `json:"scheduledOn"`
		Income      domain.Income `json:"income"`
	}
	if e.BindBody(&body) != nil {
		return fail(e, a.recurringRequestFailure(e, "income_occurrence.confirmed", domain.ErrInvalid))
	}
	date, err := recurringDate(body.ScheduledOn)
	if err != nil {
		return fail(e, a.recurringRequestFailure(e, "income_occurrence.confirmed", err))
	}
	value, err := a.Service.ConfirmIncomeOccurrence(e.Request.Context(), authID(e), e.Request.PathValue("id"), date, body.Income)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, value, nil)
}
func (a *API) skipIncomeOccurrence(e *core.RequestEvent) error {
	if err := a.authorizeRecurringRoute(e, "ledger.incomes.write"); err != nil {
		return fail(e, err)
	}
	var body struct {
		ScheduledOn string `json:"scheduledOn"`
		Reason      string `json:"reason"`
	}
	if e.BindBody(&body) != nil {
		return fail(e, a.recurringRequestFailure(e, "income_occurrence.skipped", domain.ErrInvalid))
	}
	date, err := recurringDate(body.ScheduledOn)
	if err != nil {
		return fail(e, a.recurringRequestFailure(e, "income_occurrence.skipped", err))
	}
	if err = a.Service.SkipIncomeOccurrence(e.Request.Context(), authID(e), e.Request.PathValue("id"), date, body.Reason); err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, map[string]bool{"ok": true}, nil)
}
func (a *API) retryIncomeOccurrence(e *core.RequestEvent) error {
	if err := a.authorizeRecurringRoute(e, "ledger.incomes.write"); err != nil {
		return fail(e, err)
	}
	var body struct {
		ScheduledOn string `json:"scheduledOn"`
	}
	if e.BindBody(&body) != nil {
		return fail(e, a.recurringRequestFailure(e, "income_occurrence.retried", domain.ErrInvalid))
	}
	date, err := recurringDate(body.ScheduledOn)
	if err != nil {
		return fail(e, a.recurringRequestFailure(e, "income_occurrence.retried", err))
	}
	if err = a.Service.RetryIncomeOccurrence(e.Request.Context(), authID(e), e.Request.PathValue("id"), date); err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, map[string]bool{"ok": true}, nil)
}
func (a *API) previewRecurringIncomeBackfill(e *core.RequestEvent) error {
	if err := a.authorizeRecurringRoute(e, "ledger.incomes.write"); err != nil {
		return fail(e, err)
	}
	through, err := recurringDate(e.Request.URL.Query().Get("through"))
	if err != nil {
		return fail(e, a.recurringRequestFailure(e, "recurring_income.backfill_previewed", err))
	}
	preview, err := a.Service.PreviewRecurringIncomeBackfill(e.Request.Context(), authID(e), e.Request.PathValue("id"), through)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, preview, nil)
}
func (a *API) backfillRecurringIncome(e *core.RequestEvent) error {
	if err := a.authorizeRecurringRoute(e, "ledger.incomes.write"); err != nil {
		return fail(e, err)
	}
	var body struct {
		Through string `json:"through"`
	}
	if e.BindBody(&body) != nil {
		return fail(e, a.recurringRequestFailure(e, "recurring_income.backfilled", domain.ErrInvalid))
	}
	through, err := recurringDate(body.Through)
	if err != nil {
		return fail(e, a.recurringRequestFailure(e, "recurring_income.backfilled", err))
	}
	if err = a.Service.BackfillRecurringIncome(e.Request.Context(), authID(e), e.Request.PathValue("id"), through); err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, map[string]bool{"ok": true}, nil)
}
