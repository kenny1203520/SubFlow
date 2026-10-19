package httpapi

import (
	"github.com/pocketbase/pocketbase/core"
	"net/http"
	"subflow/internal/application"
	"subflow/internal/domain"
)

func (a *API) updateGroupOrPersonalIncome(e *core.RequestEvent) error {
	var patch application.IncomePatch
	if err := e.BindBody(&patch); err != nil {
		return fail(e, a.Service.IncomeRequestFailure(e.Request.Context(), authID(e), groupID(e), "income.updated", e.Request.PathValue("id"), domain.ErrInvalid))
	}
	updated, err := a.Service.PatchIncome(e.Request.Context(), authID(e), groupID(e), e.Request.PathValue("id"), patch)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}

func (a *API) deleteGroupOrPersonalIncome(e *core.RequestEvent) error {
	if err := a.Service.DeleteIncomeInScope(e.Request.Context(), authID(e), groupID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
