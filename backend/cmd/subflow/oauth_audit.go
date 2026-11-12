package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"

	"ganigeorgiev.com/pocketbase/pocketbase/apis"
	"ganigeorgiev.com/pocketbase/pocketbase/core"

	"subflow/internal/application"
)

type oauthAuditRequestStateKey struct{}

type oauthAuditRequestState struct {
	provider      string
	registration  bool
	link          bool
	failureReason string
	recorded      bool
}

type oauthAuditSummary struct {
	Provider string `json:"provider"`
	Reason   string `json:"reason"`
}

func registerOAuthAudit(serve *core.ServeEvent, service *application.Service) {
	serve.App.OnRecordAuthWithOAuth2Request().BindFunc(func(event *core.RecordAuthWithOAuth2RequestEvent) error {
		state := oauthAuditState(event.Request.Context())
		if state != nil {
			state.provider = safeOAuthProviderID(event.ProviderName)
			state.registration = event.IsNewRecord
			state.link = event.Auth != nil
			state.failureReason = ""
		}

		if event.Auth != nil && event.Record != nil && event.Auth.Id != event.Record.Id {
			if state != nil {
				state.failureReason = "account_mismatch"
			}
			return event.ForbiddenError("oauth_account_mismatch", nil)
		}

		return event.Next()
	})

	serve.App.OnRecordAuthRequest().BindFunc(func(event *core.RecordAuthRequestEvent) error {
		err := event.Next()
		if event.AuthMethod != core.MFAMethodOAuth2 {
			return err
		}

		state := oauthAuditState(event.Request.Context())
		action, outcome, reason := oauthAuditResult(state, err)
		actorID := ""
		if event.Record != nil {
			actorID = event.Record.Id
		}
		recordOAuthAudit(event.Request.Context(), service, event.App.Logger(), state, actorID, action, outcome, reason)
		return err
	})

	serve.Router.BindFunc(func(event *core.RequestEvent) error {
		if event.Request.Method != http.MethodPost || !strings.HasSuffix(event.Request.URL.Path, "/auth-with-oauth2") {
			return event.Next()
		}

		state := &oauthAuditRequestState{}
		event.Request = event.Request.WithContext(context.WithValue(event.Request.Context(), oauthAuditRequestStateKey{}, state))
		originalResponse := event.Response
		bufferedResponse := httptest.NewRecorder()
		event.Response = bufferedResponse
		requestErr := event.Next()
		event.Response = originalResponse

		if !state.recorded && (requestErr != nil || bufferedResponse.Code >= http.StatusBadRequest) {
			action := oauthAuditAction(state)
			reason := state.failureReason
			if reason == "" {
				reason = "authentication_failed"
			}
			actorID := ""
			if event.Auth != nil {
				actorID = event.Auth.Id
			}
			recordOAuthAudit(event.Request.Context(), service, event.App.Logger(), state, actorID, action, "failure", reason)
		}

		if requestErr != nil {
			return requestErr
		}
		replayResponse(originalResponse, bufferedResponse)
		return nil
	})
}

func oauthAuditState(ctx context.Context) *oauthAuditRequestState {
	state, _ := ctx.Value(oauthAuditRequestStateKey{}).(*oauthAuditRequestState)
	return state
}

func safeOAuthProviderID(value string) string {
	if value == "" || len(value) > 100 {
		return "unknown"
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' {
			continue
		}
		return "unknown"
	}
	return value
}

func oauthAuditAction(state *oauthAuditRequestState) string {
	if state != nil && state.registration {
		return "user.registered"
	}
	if state != nil && state.link {
		return "user.provider_linked"
	}
	return "user.login"
}

func oauthAuditResult(state *oauthAuditRequestState, authErr error) (action, outcome, reason string) {
	if errors.Is(authErr, apis.ErrMFA) {
		return "user.oauth2.mfa_required", "success", "mfa_required"
	}
	action = oauthAuditAction(state)
	if authErr != nil {
		reason = "authentication_failed"
		if state != nil && state.failureReason != "" {
			reason = state.failureReason
		}
		return action, "failure", reason
	}
	return action, "success", "authenticated"
}

func recordOAuthAudit(ctx context.Context, service *application.Service, logger interface{ Error(string, ...any) }, state *oauthAuditRequestState, actorID, action, outcome, reason string) {
	if state == nil {
		state = &oauthAuditRequestState{provider: "unknown"}
	}
	state.recorded = true
	summary, _ := json.Marshal(oauthAuditSummary{Provider: safeOAuthProviderID(state.provider), Reason: reason})
	if err := service.RecordAudit(ctx, actorID, "", action, "user", actorID, outcome, string(summary)); err != nil {
		logger.Error("OAuth audit write failed", "action", action, "provider", safeOAuthProviderID(state.provider), "reason", "audit_write_failed")
	}
}
