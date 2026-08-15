package httpapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"
)

func TestCreateExpenseAsGroupMemberSucceeds(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	member := f.createUser(t, "member@example.com")
	group := f.seedGroup(t, owner.Id, member.Id)
	body := `{"title":"Groceries","amountMinor":52000,"currency":"TWD","incurredOn":"2026-08-01T00:00:00Z"}`
	scenario := tests.ApiScenario{
		Name:                  "a regular group member can create a group expense",
		Method:                http.MethodPost,
		URL:                   "/api/subflow/v1/groups/" + group.ID + "/expenses",
		Body:                  strings.NewReader(body),
		Headers:               map[string]string{"Authorization": f.token(t, member)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusCreated,
		ExpectedContent:       []string{`"title":"Groceries"`},
	}
	scenario.Test(t)
}

func TestCreateExpenseRejectsInvalidBody(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	group := f.seedGroup(t, owner.Id, "")
	body := `{"title":"Groceries","amountMinor":-100,"currency":"TWD","incurredOn":"2026-08-01T00:00:00Z"}`
	scenario := tests.ApiScenario{
		Name:                  "a negative amount is rejected",
		Method:                http.MethodPost,
		URL:                   "/api/subflow/v1/groups/" + group.ID + "/expenses",
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

func TestListExpensesRequiresAuth(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	group := f.seedGroup(t, owner.Id, "")
	scenario := tests.ApiScenario{
		Name:                  "listing group expenses without a token is rejected",
		Method:                http.MethodGet,
		URL:                   "/api/subflow/v1/groups/" + group.ID + "/expenses",
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusUnauthorized,
		ExpectedContent:       []string{"authorization"},
	}
	scenario.Test(t)
}

func TestDeleteExpenseRejectsUnknownID(t *testing.T) {
	f := newAPITestApp(t)
	outsider := f.createUser(t, "outsider@example.com")
	scenario := tests.ApiScenario{
		Name:                  "deleting a nonexistent expense is rejected, not silently accepted",
		Method:                http.MethodDelete,
		URL:                   "/api/subflow/v1/expenses/nonexistent-id",
		Headers:               map[string]string{"Authorization": f.token(t, outsider)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusNotFound,
		ExpectedContent:       []string{`"code":"not_found"`},
	}
	scenario.Test(t)
}
