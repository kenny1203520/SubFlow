package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"subflow/internal/application"
	"subflow/internal/domain"
)

type CollaborationAPI struct {
	Service *application.CollaborationService
}

func (a *CollaborationAPI) RegisterRoutes(e *core.ServeEvent) {
	bind := apis.RequireAuth("users")
	e.Router.GET("/api/subflow/v1/groups/{groupId}/invitations", a.list).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/invitations", a.create).Bind(bind)
	e.Router.POST("/api/subflow/v1/invitations/{id}/resend", a.resend).Bind(bind)
	e.Router.POST("/api/subflow/v1/invitations/{id}/revoke", a.revoke).Bind(bind)
	e.Router.POST("/api/subflow/v1/invitations/accept", a.accept).Bind(bind)
	e.Router.GET("/api/subflow/v1/events", a.events).Bind(bind)
}
func (a *CollaborationAPI) list(e *core.RequestEvent) error {
	p, err := pageRequest(e, "members")
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListInvitations(e.Request.Context(), authID(e), groupID(e), p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, pageMeta(v))
}
func (a *CollaborationAPI) create(e *core.RequestEvent) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := e.BindBody(&body); err != nil {
		return fail(e, domain.ErrInvalid)
	}
	v, err := a.Service.CreateInvitation(e.Request.Context(), authID(e), groupID(e), body.Email)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, v, nil)
}
func (a *CollaborationAPI) resend(e *core.RequestEvent) error {
	v, err := a.Service.ResendInvitation(e.Request.Context(), authID(e), e.Request.PathValue("id"))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v, nil)
}
func (a *CollaborationAPI) revoke(e *core.RequestEvent) error {
	if err := a.Service.RevokeInvitation(e.Request.Context(), authID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, map[string]bool{"revoked": true}, nil)
}
func (a *CollaborationAPI) accept(e *core.RequestEvent) error {
	var body struct {
		Token string `json:"token"`
	}
	if err := e.BindBody(&body); err != nil || body.Token == "" {
		return fail(e, domain.ErrInvalid)
	}
	v, err := a.Service.AcceptInvitation(e.Request.Context(), authID(e), body.Token)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v, nil)
}
func (a *CollaborationAPI) events(e *core.RequestEvent) error {
	group := e.Request.URL.Query().Get("groupId")
	if group == "" {
		return fail(e, domain.ErrInvalid)
	}
	if _, err := a.Service.Base.GetGroup(e.Request.Context(), authID(e), group); err != nil {
		return fail(e, err)
	}
	ch, cancel := a.Service.Events.Subscribe(e.Request.Context(), group)
	defer cancel()
	e.Response.Header().Set("Content-Type", "text/event-stream")
	e.Response.Header().Set("Cache-Control", "no-cache")
	e.Response.Header().Set("Connection", "keep-alive")
	e.Response.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(e.Response, "retry: 2000\n\n")
	_ = e.Flush()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case event, open := <-ch:
			if !open {
				return nil
			}
			payload, _ := json.Marshal(event)
			if _, err := fmt.Fprintf(e.Response, "event: subflow\ndata: %s\n\n", payload); err != nil {
				return nil
			}
			_ = e.Flush()
		case <-ticker.C:
			if _, err := fmt.Fprint(e.Response, ": keepalive\n\n"); err != nil {
				return nil
			}
			_ = e.Flush()
		case <-e.Request.Context().Done():
			return nil
		}
	}
}
