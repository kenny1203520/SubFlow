package main

import (
	"errors"
	"testing"

	"ganigeorgiev.com/pocketbase/pocketbase/apis"
)

func TestOAuthAuditResultCoversAuthenticationOutcomes(t *testing.T) {
	tests := []struct {
		name    string
		state   *oauthAuditRequestState
		err     error
		action  string
		outcome string
		reason  string
	}{
		{name: "login success", state: &oauthAuditRequestState{provider: "google"}, action: "user.login", outcome: "success", reason: "authenticated"},
		{name: "login failure", state: &oauthAuditRequestState{provider: "github"}, err: errors.New("secret-bearing provider error"), action: "user.login", outcome: "failure", reason: "authentication_failed"},
		{name: "registration", state: &oauthAuditRequestState{provider: "custom-oidc", registration: true}, action: "user.registered", outcome: "success", reason: "authenticated"},
		{name: "provider link", state: &oauthAuditRequestState{provider: "spotify", link: true}, action: "user.provider_linked", outcome: "success", reason: "authenticated"},
		{name: "mfa required", state: &oauthAuditRequestState{provider: "discord"}, err: apis.ErrMFA, action: "user.oauth2.mfa_required", outcome: "success", reason: "mfa_required"},
		{name: "account mismatch", state: &oauthAuditRequestState{provider: "github", link: true, failureReason: "account_mismatch"}, err: errors.New("mismatch"), action: "user.provider_linked", outcome: "failure", reason: "account_mismatch"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			action, outcome, reason := oauthAuditResult(test.state, test.err)
			if action != test.action || outcome != test.outcome || reason != test.reason {
				t.Fatalf("got (%q, %q, %q), expected (%q, %q, %q)", action, outcome, reason, test.action, test.outcome, test.reason)
			}
		})
	}
}

func TestOAuthAuditProviderIDIsRestrictedToProviderIdentifiers(t *testing.T) {
	if got := safeOAuthProviderID("custom-oidc.v2"); got != "custom-oidc.v2" {
		t.Fatalf("expected custom provider ID, got %q", got)
	}
	if got := safeOAuthProviderID("oauth-provider?token=secret"); got != "unknown" {
		t.Fatalf("expected unsafe identifier to be omitted, got %q", got)
	}
}
