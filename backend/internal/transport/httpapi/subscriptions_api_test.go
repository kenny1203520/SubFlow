package httpapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"
)

func TestListSubscriptionsRequiresAuth(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	group := f.seedGroup(t, owner.Id, "")
	scenario := tests.ApiScenario{
		Name:                  "listing group subscriptions without a token is rejected",
		Method:                http.MethodGet,
		URL:                   "/api/subflow/v1/groups/" + group.ID + "/subscriptions",
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusUnauthorized,
		ExpectedContent:       []string{"authorization"},
	}
	scenario.Test(t)
}

func TestCreateSubscriptionAsGroupMemberSucceeds(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	group := f.seedGroup(t, owner.Id, "")
	body := `{"name":"Netflix","amountMinor":39000,"currency":"TWD","billingCycle":"monthly","status":"active","startsOn":"2026-08-01T00:00:00Z"}`
	scenario := tests.ApiScenario{
		Name:                  "the group owner can create a group subscription",
		Method:                http.MethodPost,
		URL:                   "/api/subflow/v1/groups/" + group.ID + "/subscriptions",
		Body:                  strings.NewReader(body),
		Headers:               map[string]string{"Authorization": f.token(t, owner)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusCreated,
		ExpectedContent:       []string{`"name":"Netflix"`, `"status":"active"`},
	}
	scenario.Test(t)
}

func TestCreateSubscriptionRejectsInvalidBody(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	group := f.seedGroup(t, owner.Id, "")
	body := `{"name":"","amountMinor":39000,"currency":"TWD","billingCycle":"monthly","status":"active","startsOn":"2026-08-01T00:00:00Z"}`
	scenario := tests.ApiScenario{
		Name:                  "creating a subscription with a blank name is rejected",
		Method:                http.MethodPost,
		URL:                   "/api/subflow/v1/groups/" + group.ID + "/subscriptions",
		Body:                  strings.NewReader(body),
		Headers:               map[string]string{"Authorization": f.token(t, owner)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusBadRequest,
		ExpectedContent:       []string{`"code":"invalid_request"`},
	}
	scenario.Test(t)
}

func TestCreateSubscriptionRejectsNonMember(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	outsider := f.createUser(t, "outsider@example.com")
	group := f.seedGroup(t, owner.Id, "")
	body := `{"name":"Netflix","amountMinor":39000,"currency":"TWD","billingCycle":"monthly","status":"active","startsOn":"2026-08-01T00:00:00Z"}`
	scenario := tests.ApiScenario{
		Name:                  "a user outside the group cannot create a subscription in it",
		Method:                http.MethodPost,
		URL:                   "/api/subflow/v1/groups/" + group.ID + "/subscriptions",
		Body:                  strings.NewReader(body),
		Headers:               map[string]string{"Authorization": f.token(t, outsider)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusNotFound,
		ExpectedContent:       []string{`"code":"not_found"`},
	}
	scenario.Test(t)
}

func TestListPersonalSubscriptionsForAuthenticatedUser(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "solo@example.com")
	scenario := tests.ApiScenario{
		Name:                  "an authenticated user can list their own (empty) personal subscriptions",
		Method:                http.MethodGet,
		URL:                   "/api/subflow/v1/subscriptions",
		Headers:               map[string]string{"Authorization": f.token(t, owner)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"data":[]`},
	}
	scenario.Test(t)
}
