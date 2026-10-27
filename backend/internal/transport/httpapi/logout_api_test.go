package httpapi_test

import (
	"net/http"
	"strings"
	"testing"

	"ganigeorgiev.com/pocketbase/pocketbase/tests"

	"subflow/internal/ports"
)

func TestLogoutAuditsSuccessAndRejection(t *testing.T) {
	for _, authenticated := range []bool{true, false} {
		t.Run(map[bool]string{true: "authenticated", false: "anonymous"}[authenticated], func(t *testing.T) {
			f := newAPITestApp(t)
			user := f.createUser(t, "logout@example.test")
			token := f.token(t, user)
			headers := map[string]string{}
			status, outcome := http.StatusForbidden, "failure"
			content := []string{`"code":"forbidden"`}
			if authenticated {
				headers["Authorization"] = token
				status, outcome = http.StatusOK, "success"
				content = []string{`"recorded":true`}
			}
			scenario := tests.ApiScenario{Method: http.MethodPost, URL: "/api/subflow/v1/auth/logout", Headers: headers, TestAppFactory: f.factory(), BeforeTestFunc: f.beforeTest(), DisableTestAppCleanup: true, ExpectedStatus: status, ExpectedContent: content}
			scenario.Test(t)
			logs, err := f.stores.Audits.List(t.Context(), "", ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 10}, Action: "auth.logout"})
			if err != nil || len(logs.Items) != 1 || logs.Items[0].Outcome != outcome {
				t.Fatalf("logout audit missing: %#v, %v", logs.Items, err)
			}
			if strings.Contains(logs.Items[0].Summary, token) {
				t.Fatal("logout audit leaked auth token")
			}
		})
	}
}

func TestLogoutAuditFailureRejectsSuccessAndRecordsFailure(t *testing.T) {
	f := newAPITestApp(t)
	user := f.createUser(t, "logout-audit-failure@example.test")
	f.service.Stores.Audits = attachmentAuditFailure{f.stores.Audits}
	scenario := tests.ApiScenario{Method: http.MethodPost, URL: "/api/subflow/v1/auth/logout", Headers: map[string]string{"Authorization": f.token(t, user)}, TestAppFactory: f.factory(), BeforeTestFunc: f.beforeTest(), DisableTestAppCleanup: true, ExpectedStatus: http.StatusServiceUnavailable, ExpectedContent: []string{`"code":"audit_unavailable"`}}
	scenario.Test(t)
	logs, err := f.stores.Audits.List(t.Context(), "", ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 10}, Action: "auth.logout"})
	if err != nil || len(logs.Items) != 1 || logs.Items[0].Outcome != "failure" {
		t.Fatalf("expected an explicit failure audit: %#v, %v", logs.Items, err)
	}
}
