package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"subflow/internal/application"
	"subflow/internal/domain"
	"subflow/internal/ports"
)

type API struct{ Service *application.Service }

type envelope struct {
	Data any `json:"data"`
	Meta any `json:"meta,omitempty"`
}
type errorEnvelope struct {
	Error apiError `json:"error"`
}
type apiError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (a *API) Register(router *http.ServeMux) {}

func (a *API) RegisterRoutes(e *core.ServeEvent) {
	protected := func(route *core.RequestEvent) error { return route.Next() }
	_ = protected
	bind := apis.RequireAuth("users")
	e.Router.GET("/api/subflow/v1/groups", a.listGroups).Bind(bind)
	e.Router.GET("/api/subflow/v1/dashboard", a.workspaceDashboard).Bind(bind)
	e.Router.GET("/api/subflow/v1/subscriptions", a.listPersonalSubscriptions).Bind(bind)
	e.Router.POST("/api/subflow/v1/subscriptions", a.createPersonalSubscription).Bind(bind)
	e.Router.GET("/api/subflow/v1/expenses", a.listPersonalExpenses).Bind(bind)
	e.Router.POST("/api/subflow/v1/expenses", a.createPersonalExpense).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups", a.createGroup).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}", a.getGroup).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/groups/{groupId}", a.updateGroup).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/groups/{groupId}", a.deleteGroup).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/summary", a.dashboard).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/members", a.listMembers).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/groups/{groupId}/members/{userId}", a.removeMember).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/subscriptions", a.listSubscriptions).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/subscriptions", a.createSubscription).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/subscriptions/{id}", a.updateSubscription).Bind(bind)
	e.Router.POST("/api/subflow/v1/subscriptions/{id}/stop", a.stopSubscription).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/subscriptions/{id}/stop", a.resumeSubscription).Bind(bind)
	e.Router.GET("/api/subflow/v1/subscriptions/{id}/billing-dates", a.billingDates).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/subscriptions/{id}", a.deleteSubscription).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/expenses", a.listExpenses).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/expenses", a.createExpense).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/expenses/{id}", a.updateExpense).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/expenses/{id}", a.deleteExpense).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/settlements", a.listSettlements).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/settlements", a.createSettlement).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/settlements/{id}", a.deleteSettlement).Bind(bind)
}

func authID(e *core.RequestEvent) string {
	if e.Auth == nil {
		return ""
	}
	return e.Auth.Id
}
func groupID(e *core.RequestEvent) string { return e.Request.PathValue("groupId") }
func ok(e *core.RequestEvent, status int, data any, meta any) error {
	return e.JSON(status, envelope{Data: data, Meta: meta})
}
func noContent(e *core.RequestEvent) error {
	return e.JSON(http.StatusOK, envelope{Data: map[string]bool{"deleted": true}})
}
func fail(e *core.RequestEvent, err error) error {
	status := http.StatusInternalServerError
	code := "internal_error"
	message := "服務暫時無法處理請求"
	switch {
	case errors.Is(err, domain.ErrInvalid):
		status = http.StatusBadRequest
		code = "invalid_request"
		message = "輸入資料不正確"
	case errors.Is(err, domain.ErrForbidden):
		status = http.StatusForbidden
		code = "forbidden"
		message = "沒有權限執行此操作"
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
		code = "not_found"
		message = "找不到指定資源"
	case errors.Is(err, domain.ErrConflict):
		status = http.StatusConflict
		code = "conflict"
		message = "資料狀態衝突"
	}
	return e.JSON(status, errorEnvelope{Error: apiError{Code: code, Message: message}})
}

var sorts = map[string]map[string]bool{"groups": {"name": true, "-name": true, "created": true, "-created": true}, "members": {"created": true, "-created": true}, "subscriptions": {"name": true, "-name": true, "next_billing": true, "-next_billing": true, "created": true, "-created": true}, "expenses": {"incurred_on": true, "-incurred_on": true, "created": true, "-created": true}, "settlements": {"settled_on": true, "-settled_on": true, "created": true, "-created": true}}

func pageRequest(e *core.RequestEvent, resource string) (ports.PageRequest, error) {
	q := e.Request.URL.Query()
	p, _ := strconv.Atoi(q.Get("page"))
	pp, _ := strconv.Atoi(q.Get("perPage"))
	sort := q.Get("sort")
	if sort != "" && !sorts[resource][sort] {
		return ports.PageRequest{}, domain.ErrInvalid
	}
	return ports.PageRequest{Page: p, PerPage: pp, Sort: sort}, nil
}
func pageMeta[T any](p ports.Page[T]) map[string]int {
	return map[string]int{"page": p.Page, "perPage": p.PerPage, "totalItems": p.TotalItems, "totalPages": p.TotalPages}
}

func (a *API) listGroups(e *core.RequestEvent) error {
	p, err := pageRequest(e, "groups")
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListGroups(e.Request.Context(), authID(e), p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, pageMeta(v))
}
func (a *API) createGroup(e *core.RequestEvent) error {
	var v domain.Group
	if err := e.BindBody(&v); err != nil {
		return fail(e, domain.ErrInvalid)
	}
	created, err := a.Service.CreateGroup(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) getGroup(e *core.RequestEvent) error {
	v, err := a.Service.GetGroup(e.Request.Context(), authID(e), groupID(e))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v, nil)
}
func (a *API) updateGroup(e *core.RequestEvent) error {
	var v domain.Group
	if err := e.BindBody(&v); err != nil {
		return fail(e, domain.ErrInvalid)
	}
	v.ID = groupID(e)
	updated, err := a.Service.UpdateGroup(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}
func (a *API) deleteGroup(e *core.RequestEvent) error {
	if err := a.Service.DeleteGroup(e.Request.Context(), authID(e), groupID(e)); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) dashboard(e *core.RequestEvent) error {
	v, err := a.Service.WorkspaceDashboard(e.Request.Context(), authID(e), application.DashboardQuery{Scope: "group", GroupID: groupID(e), Month: e.Request.URL.Query().Get("month")})
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v, nil)
}
func (a *API) workspaceDashboard(e *core.RequestEvent) error {
	scope := e.Request.URL.Query().Get("scope")
	v, err := a.Service.WorkspaceDashboard(e.Request.Context(), authID(e), application.DashboardQuery{Scope: scope, GroupID: e.Request.URL.Query().Get("groupId"), Month: e.Request.URL.Query().Get("month")})
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v, nil)
}
func (a *API) billingDates(e *core.RequestEvent) error {
	limit, _ := strconv.Atoi(e.Request.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 12
	}
	value, err := a.Service.BillingDates(e.Request.Context(), authID(e), e.Request.PathValue("id"), e.Request.URL.Query().Get("cursor"), limit)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, value, nil)
}
func (a *API) listMembers(e *core.RequestEvent) error {
	p, err := pageRequest(e, "members")
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListMembers(e.Request.Context(), authID(e), groupID(e), p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, pageMeta(v))
}
func (a *API) removeMember(e *core.RequestEvent) error {
	if err := a.Service.RemoveMember(e.Request.Context(), authID(e), groupID(e), e.Request.PathValue("userId")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) listSubscriptions(e *core.RequestEvent) error {
	p, err := pageRequest(e, "subscriptions")
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListSubscriptions(e.Request.Context(), authID(e), groupID(e), p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, pageMeta(v))
}
func (a *API) listPersonalSubscriptions(e *core.RequestEvent) error {
	p, err := pageRequest(e, "subscriptions")
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListPersonalSubscriptions(e.Request.Context(), authID(e), p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, pageMeta(v))
}
func (a *API) createPersonalSubscription(e *core.RequestEvent) error {
	var v domain.Subscription
	if e.BindBody(&v) != nil {
		return fail(e, domain.ErrInvalid)
	}
	created, err := a.Service.CreateSubscription(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) createSubscription(e *core.RequestEvent) error {
	var v domain.Subscription
	if err := e.BindBody(&v); err != nil {
		return fail(e, domain.ErrInvalid)
	}
	v.GroupID = groupID(e)
	created, err := a.Service.CreateSubscription(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) updateSubscription(e *core.RequestEvent) error {
	var v domain.Subscription
	if err := e.BindBody(&v); err != nil {
		return fail(e, domain.ErrInvalid)
	}
	v.ID = e.Request.PathValue("id")
	updated, err := a.Service.UpdateSubscription(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}
func (a *API) stopSubscription(e *core.RequestEvent) error {
	var body struct {
		EndsOn string `json:"endsOn"`
	}
	if e.BindBody(&body) != nil {
		return fail(e, domain.ErrInvalid)
	}
	updated, err := a.Service.StopSubscription(e.Request.Context(), authID(e), e.Request.PathValue("id"), body.EndsOn)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}
func (a *API) resumeSubscription(e *core.RequestEvent) error {
	updated, err := a.Service.ResumeSubscription(e.Request.Context(), authID(e), e.Request.PathValue("id"))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}
func (a *API) deleteSubscription(e *core.RequestEvent) error {
	if err := a.Service.DeleteSubscription(e.Request.Context(), authID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) listExpenses(e *core.RequestEvent) error {
	p, err := pageRequest(e, "expenses")
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListExpenses(e.Request.Context(), authID(e), groupID(e), p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, pageMeta(v))
}
func (a *API) listPersonalExpenses(e *core.RequestEvent) error {
	p, err := pageRequest(e, "expenses")
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListPersonalExpenses(e.Request.Context(), authID(e), p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, pageMeta(v))
}
func (a *API) createPersonalExpense(e *core.RequestEvent) error {
	var v domain.Expense
	if e.BindBody(&v) != nil {
		return fail(e, domain.ErrInvalid)
	}
	if v.PaidBy == "" {
		v.PaidBy = authID(e)
	}
	created, err := a.Service.CreateExpense(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) createExpense(e *core.RequestEvent) error {
	var v domain.Expense
	if err := e.BindBody(&v); err != nil {
		return fail(e, domain.ErrInvalid)
	}
	v.GroupID = groupID(e)
	if v.PaidBy == "" {
		v.PaidBy = authID(e)
	}
	created, err := a.Service.CreateExpense(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) updateExpense(e *core.RequestEvent) error {
	var v domain.Expense
	if err := e.BindBody(&v); err != nil {
		return fail(e, domain.ErrInvalid)
	}
	v.ID = e.Request.PathValue("id")
	updated, err := a.Service.UpdateExpense(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}
func (a *API) deleteExpense(e *core.RequestEvent) error {
	if err := a.Service.DeleteExpense(e.Request.Context(), authID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) listSettlements(e *core.RequestEvent) error {
	page, err := pageRequest(e, "settlements")
	if err != nil {
		return fail(e, err)
	}
	values, err := a.Service.ListSettlements(e.Request.Context(), authID(e), groupID(e), page)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, values.Items, pageMeta(values))
}
func (a *API) createSettlement(e *core.RequestEvent) error {
	var value domain.Settlement
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	value.GroupID = groupID(e)
	created, err := a.Service.CreateSettlement(e.Request.Context(), authID(e), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) deleteSettlement(e *core.RequestEvent) error {
	if err := a.Service.DeleteSettlement(e.Request.Context(), authID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
