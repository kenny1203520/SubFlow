package httpapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"
)

func TestCreateGroupSucceeds(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	body := `{"name":"Roommates","currency":"TWD"}`
	scenario := tests.ApiScenario{
		Name:                  "an authenticated user can create a group and becomes its owner",
		Method:                http.MethodPost,
		URL:                   "/api/subflow/v1/groups",
		Body:                  strings.NewReader(body),
		Headers:               map[string]string{"Authorization": f.token(t, owner)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusCreated,
		ExpectedContent:       []string{`"name":"Roommates"`},
	}
	scenario.Test(t)
}

func TestGetGroupRejectsNonMember(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	outsider := f.createUser(t, "outsider@example.com")
	group := f.seedGroup(t, owner.Id, "")
	scenario := tests.ApiScenario{
		Name:                  "a user outside the group cannot fetch its details",
		Method:                http.MethodGet,
		URL:                   "/api/subflow/v1/groups/" + group.ID,
		Headers:               map[string]string{"Authorization": f.token(t, outsider)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusNotFound,
		ExpectedContent:       []string{`"code":"not_found"`},
	}
	scenario.Test(t)
}

func TestDeleteGroupRequiresOwner(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	member := f.createUser(t, "member@example.com")
	group := f.seedGroup(t, owner.Id, member.Id)
	scenario := tests.ApiScenario{
		Name:                  "a regular member cannot delete the group, only its owner can",
		Method:                http.MethodDelete,
		URL:                   "/api/subflow/v1/groups/" + group.ID,
		Headers:               map[string]string{"Authorization": f.token(t, member)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusForbidden,
		ExpectedContent:       []string{`"code":"forbidden"`},
	}
	scenario.Test(t)
}

func TestListMembersReturnsOwnerAndMember(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	member := f.createUser(t, "member@example.com")
	group := f.seedGroup(t, owner.Id, member.Id)
	scenario := tests.ApiScenario{
		Name:                  "listing group members returns both the owner and the member",
		Method:                http.MethodGet,
		URL:                   "/api/subflow/v1/groups/" + group.ID + "/members",
		Headers:               map[string]string{"Authorization": f.token(t, owner)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"totalItems":2`},
	}
	scenario.Test(t)
}
