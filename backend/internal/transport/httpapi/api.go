package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

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
	e.Router.GET("/api/subflow/v1/setup/status", a.setupStatus)
	e.Router.POST("/api/subflow/v1/setup/initialize", a.initializeSetup)
	e.Router.POST("/api/subflow/v1/auth/register", a.register)
	e.Router.GET("/api/subflow/v1/groups", a.listGroups).Bind(bind)
	e.Router.GET("/api/subflow/v1/currencies", a.currencies).Bind(bind)
	e.Router.GET("/api/subflow/v1/categories", a.listCategories).Bind(bind)
	e.Router.GET("/api/subflow/v1/system/access", a.systemAccess).Bind(bind)
	e.Router.POST("/api/subflow/v1/categories", a.createCategory).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/categories/{id}", a.updateCategory).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/categories/{id}", a.archiveCategory).Bind(bind)
	e.Router.GET("/api/subflow/v1/exchange-rates/quote", a.exchangeRateQuote).Bind(bind)
	e.Router.GET("/api/subflow/v1/dashboard", a.workspaceDashboard).Bind(bind)
	e.Router.GET("/api/subflow/v1/subscriptions", a.listPersonalSubscriptions).Bind(bind)
	e.Router.POST("/api/subflow/v1/subscriptions", a.createPersonalSubscription).Bind(bind)
	e.Router.GET("/api/subflow/v1/expenses", a.listPersonalExpenses).Bind(bind)
	e.Router.POST("/api/subflow/v1/expenses", a.createPersonalExpense).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups", a.createGroup).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}", a.getGroup).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/groups/{groupId}", a.updateGroup).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/groups/{groupId}", a.deleteGroup).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/currency-change/preview", a.previewCurrencyChange).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/currency-change", a.changeCurrency).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/summary", a.dashboard).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/members", a.listMembers).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/groups/{groupId}/members/{userId}", a.removeMember).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/roles", a.listGroupRoles).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/roles", a.createGroupRole).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/groups/{groupId}/roles/{id}", a.updateGroupRole).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/groups/{groupId}/roles/{id}", a.deleteGroupRole).Bind(bind)
	e.Router.PUT("/api/subflow/v1/groups/{groupId}/members/{userId}/role", a.assignGroupRole).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/audit-logs", a.listGroupAudit).Bind(bind)
	e.Router.GET("/api/subflow/v1/system/roles", a.listSystemRoles).Bind(bind)
	e.Router.POST("/api/subflow/v1/system/roles", a.createSystemRole).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/system/roles/{id}", a.updateSystemRole).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/system/roles/{id}", a.deleteSystemRole).Bind(bind)
	e.Router.PUT("/api/subflow/v1/system/users/{userId}/role", a.assignSystemRole).Bind(bind)
	e.Router.GET("/api/subflow/v1/system/users", a.listSystemUsers).Bind(bind)
	e.Router.GET("/api/subflow/v1/system/audit-logs", a.listSystemAudit).Bind(bind)
	e.Router.GET("/api/subflow/v1/system/settings", a.getSystemSettings).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/system/settings", a.updateSystemSettings).Bind(bind)
	e.Router.POST("/api/subflow/v1/system/recover-admin", a.recoverSystemAdmin).Bind(apis.RequireSuperuserAuth())
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
	case errors.Is(err, domain.ErrRateUnavailable):
		status = http.StatusUnprocessableEntity
		code = "rate_unavailable"
		message = "exchange rate unavailable"
	case errors.Is(err, domain.ErrSetupDisabled):
		status = http.StatusGone
		code = "setup_disabled"
		message = "setup is no longer available"
	case errors.Is(err, domain.ErrSetupToken):
		status = http.StatusForbidden
		code = "setup_token_invalid"
		message = "setup link is invalid or expired"
	}
	return e.JSON(status, errorEnvelope{Error: apiError{Code: code, Message: message}})
}

func (a *API) currencies(e *core.RequestEvent) error {
	return ok(e, http.StatusOK, a.Service.Currencies(), nil)
}
func (a *API) listCategories(e *core.RequestEvent) error {
	q := e.Request.URL.Query()
	values, err := a.Service.ListCategories(e.Request.Context(), authID(e), q.Get("scope"), q.Get("groupId"), q.Get("archived") == "true")
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, values, nil)
}
func (a *API) createCategory(e *core.RequestEvent) error {
	var value domain.Category
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	created, err := a.Service.CreateCategory(e.Request.Context(), authID(e), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) updateCategory(e *core.RequestEvent) error {
	var value domain.Category
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	value.ID = e.Request.PathValue("id")
	updated, err := a.Service.UpdateCategory(e.Request.Context(), authID(e), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}
func (a *API) archiveCategory(e *core.RequestEvent) error {
	updated, err := a.Service.UpdateCategory(e.Request.Context(), authID(e), domain.Category{ID: e.Request.PathValue("id"), Archived: true})
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}
func (a *API) exchangeRateQuote(e *core.RequestEvent) error {
	q := e.Request.URL.Query()
	date, err := time.Parse("2006-01-02", q.Get("date"))
	if err != nil {
		return fail(e, domain.ErrInvalid)
	}
	value, err := a.Service.QuoteRate(e.Request.Context(), domain.Currency(q.Get("from")), domain.Currency(q.Get("to")), date)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, value, nil)
}
func currencyBody(e *core.RequestEvent) (domain.Currency, error) {
	var body struct {
		Currency domain.Currency `json:"currency"`
	}
	if e.BindBody(&body) != nil {
		return "", domain.ErrInvalid
	}
	return body.Currency, nil
}
func (a *API) previewCurrencyChange(e *core.RequestEvent) error {
	currency, err := currencyBody(e)
	if err != nil {
		return fail(e, err)
	}
	value, err := a.Service.PreviewGroupCurrency(e.Request.Context(), authID(e), groupID(e), currency)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, value, nil)
}
func (a *API) changeCurrency(e *core.RequestEvent) error {
	currency, err := currencyBody(e)
	if err != nil {
		return fail(e, err)
	}
	value, err := a.Service.ChangeGroupCurrency(e.Request.Context(), authID(e), groupID(e), currency)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, value, nil)
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

func (a *API) setupStatus(e *core.RequestEvent) error {
	settings, err := a.Service.SetupStatus(e.Request.Context())
	if err != nil {
		return fail(e, err)
	}
	if settings.Initialized {
		return ok(e, http.StatusOK, map[string]any{"initialized": true, "allowRegistration": settings.AllowRegistration}, nil)
	}
	_, valid, err := a.Service.ValidateSetupToken(e.Request.Context(), e.Request.URL.Query().Get("token"))
	if err != nil {
		return fail(e, err)
	}
	if !valid {
		return ok(e, http.StatusOK, map[string]any{"initialized": false, "setupAvailable": false}, nil)
	}
	return ok(e, http.StatusOK, map[string]any{"initialized": false, "setupAvailable": true, "siteName": settings.SiteName, "defaultTimezone": settings.DefaultTimezone, "defaultCurrency": settings.DefaultCurrency, "allowRegistration": settings.AllowRegistration, "currencies": a.Service.Currencies()}, nil)
}
func (a *API) systemAccess(e *core.RequestEvent) error {
	permissions, err := a.Service.SystemPermissions(e.Request.Context(), authID(e))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, map[string]any{"permissions": permissions}, nil)
}
func (a *API) initializeSetup(e *core.RequestEvent) error {
	var value domain.SetupInput
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	created, err := a.Service.InitializeSetup(e.Request.Context(), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, map[string]string{"id": created.ID}, nil)
}
func (a *API) register(e *core.RequestEvent) error {
	var value domain.SetupInput
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	created, err := a.Service.Register(e.Request.Context(), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, map[string]string{"id": created.ID}, nil)
}
func (a *API) listGroupRoles(e *core.RequestEvent) error {
	values, err := a.Service.ListGroupRoles(e.Request.Context(), authID(e), groupID(e))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, values, nil)
}
func (a *API) createGroupRole(e *core.RequestEvent) error {
	var value domain.Role
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	value.GroupID = groupID(e)
	v, err := a.Service.CreateGroupRole(e.Request.Context(), authID(e), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, v, nil)
}
func (a *API) updateGroupRole(e *core.RequestEvent) error {
	var value domain.Role
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	value.ID = e.Request.PathValue("id")
	v, err := a.Service.UpdateGroupRole(e.Request.Context(), authID(e), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v, nil)
}
func (a *API) deleteGroupRole(e *core.RequestEvent) error {
	if err := a.Service.DeleteGroupRole(e.Request.Context(), authID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) assignGroupRole(e *core.RequestEvent) error {
	var body struct {
		RoleID string `json:"roleId"`
	}
	if e.BindBody(&body) != nil || body.RoleID == "" {
		return fail(e, domain.ErrInvalid)
	}
	if err := a.Service.AssignGroupRole(e.Request.Context(), authID(e), groupID(e), e.Request.PathValue("userId"), body.RoleID); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) listGroupAudit(e *core.RequestEvent) error {
	p, err := pageRequest(e, "created")
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListGroupAudit(e.Request.Context(), authID(e), groupID(e), p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, pageMeta(v))
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
func (a *API) listSystemRoles(e *core.RequestEvent) error {
	values, err := a.Service.ListSystemRoles(e.Request.Context(), authID(e))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, values, nil)
}
func (a *API) createSystemRole(e *core.RequestEvent) error {
	var value domain.Role
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	v, err := a.Service.CreateSystemRole(e.Request.Context(), authID(e), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, v, nil)
}
func (a *API) updateSystemRole(e *core.RequestEvent) error {
	var value domain.Role
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	value.ID = e.Request.PathValue("id")
	v, err := a.Service.UpdateSystemRole(e.Request.Context(), authID(e), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v, nil)
}
func (a *API) deleteSystemRole(e *core.RequestEvent) error {
	if err := a.Service.DeleteSystemRole(e.Request.Context(), authID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) assignSystemRole(e *core.RequestEvent) error {
	var body struct {
		RoleID string `json:"roleId"`
	}
	if e.BindBody(&body) != nil || body.RoleID == "" {
		return fail(e, domain.ErrInvalid)
	}
	if err := a.Service.AssignSystemRole(e.Request.Context(), authID(e), e.Request.PathValue("userId"), body.RoleID); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) listSystemUsers(e *core.RequestEvent) error {
	page, err := pageRequest(e, "users")
	if err != nil {
		return fail(e, err)
	}
	values, err := a.Service.ListSystemUsers(e.Request.Context(), authID(e), page, e.Request.URL.Query().Get("q"))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, values.Items, pageMeta(values))
}
func (a *API) listSystemAudit(e *core.RequestEvent) error {
	page, err := pageRequest(e, "audit logs")
	if err != nil {
		return fail(e, err)
	}
	values, err := a.Service.ListSystemAudit(e.Request.Context(), authID(e), page)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, values.Items, pageMeta(values))
}
func (a *API) getSystemSettings(e *core.RequestEvent) error {
	value, err := a.Service.GetSystemSettings(e.Request.Context(), authID(e))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, value, nil)
}
func (a *API) updateSystemSettings(e *core.RequestEvent) error {
	var value domain.SystemSettings
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	result, err := a.Service.UpdateSystemSettings(e.Request.Context(), authID(e), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, result, nil)
}
func (a *API) recoverSystemAdmin(e *core.RequestEvent) error {
	var body struct {
		UserID string `json:"userId"`
	}
	if e.BindBody(&body) != nil || body.UserID == "" {
		return fail(e, domain.ErrInvalid)
	}
	if err := a.Service.RecoverSystemAdmin(e.Request.Context(), body.UserID); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
