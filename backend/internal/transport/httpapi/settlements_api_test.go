package httpapi_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tests"

	"subflow/internal/domain"
)

func TestCreateSettlementAsGroupMemberSucceeds(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	member := f.createUser(t, "member@example.com")
	group := f.seedGroup(t, owner.Id, member.Id)
	body := `{"fromUserId":"` + member.Id + `","toUserId":"` + owner.Id + `","amountMinor":500,"settledOn":"2026-08-01T00:00:00Z"}`
	scenario := tests.ApiScenario{
		Name:                  "a member can record their own settlement",
		Method:                http.MethodPost,
		URL:                   "/api/subflow/v1/groups/" + group.ID + "/settlements",
		Body:                  strings.NewReader(body),
		Headers:               map[string]string{"Authorization": f.token(t, member)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusCreated,
		ExpectedContent:       []string{`"amountMinor":500`},
	}
	scenario.Test(t)
}

func TestUpdateSettlementAsCreatorSucceeds(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	member := f.createUser(t, "member@example.com")
	group := f.seedGroup(t, owner.Id, member.Id)
	created := f.createSettlement(t, group.ID, member.Id, member.Id, owner.Id, 500)
	body := `{"fromUserId":"` + member.Id + `","toUserId":"` + owner.Id + `","amountMinor":900,"settledOn":"2026-08-01T00:00:00Z"}`
	scenario := tests.ApiScenario{
		Name:                  "the creator can edit their own settlement",
		Method:                http.MethodPatch,
		URL:                   "/api/subflow/v1/settlements/" + created.ID,
		Body:                  strings.NewReader(body),
		Headers:               map[string]string{"Authorization": f.token(t, member)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"amountMinor":900`},
	}
	scenario.Test(t)
}

func TestUpdateSettlementRejectsInvalidBody(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	member := f.createUser(t, "member@example.com")
	group := f.seedGroup(t, owner.Id, member.Id)
	created := f.createSettlement(t, group.ID, member.Id, member.Id, owner.Id, 500)
	body := `{"fromUserId":"` + member.Id + `","toUserId":"` + owner.Id + `","amountMinor":-100,"settledOn":"2026-08-01T00:00:00Z"}`
	scenario := tests.ApiScenario{
		Name:                  "a negative amount is rejected",
		Method:                http.MethodPatch,
		URL:                   "/api/subflow/v1/settlements/" + created.ID,
		Body:                  strings.NewReader(body),
		Headers:               map[string]string{"Authorization": f.token(t, member)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusBadRequest,
		ExpectedContent:       []string{`"code":"invalid_request"`},
	}
	scenario.Test(t)
}

func TestUpdateSettlementRejectsUnknownID(t *testing.T) {
	f := newAPITestApp(t)
	outsider := f.createUser(t, "outsider@example.com")
	body := `{"fromUserId":"a","toUserId":"b","amountMinor":500,"settledOn":"2026-08-01T00:00:00Z"}`
	scenario := tests.ApiScenario{
		Name:                  "updating a nonexistent settlement is rejected, not silently accepted",
		Method:                http.MethodPatch,
		URL:                   "/api/subflow/v1/settlements/nonexistent-id",
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

func TestListSettlementsRequiresAuth(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	group := f.seedGroup(t, owner.Id, "")
	scenario := tests.ApiScenario{
		Name:                  "listing group settlements without a token is rejected",
		Method:                http.MethodGet,
		URL:                   "/api/subflow/v1/groups/" + group.ID + "/settlements",
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusUnauthorized,
		ExpectedContent:       []string{"authorization"},
	}
	scenario.Test(t)
}

func TestListSettlementsHonorsPageAndPerPage(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	member := f.createUser(t, "member@example.com")
	group := f.seedGroup(t, owner.Id, member.Id)
	for i := 0; i < 3; i++ {
		f.createSettlement(t, group.ID, member.Id, member.Id, owner.Id, 100)
	}
	scenario := tests.ApiScenario{
		Name:                  "listing settlements with perPage=2 reports 2 total pages for 3 records",
		Method:                http.MethodGet,
		URL:                   "/api/subflow/v1/groups/" + group.ID + "/settlements?page=1&perPage=2",
		Headers:               map[string]string{"Authorization": f.token(t, owner)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"totalItems":3`, `"totalPages":2`},
	}
	scenario.Test(t)
}

func TestListSettlementsFilterByMemberId(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	member := f.createUser(t, "member@example.com")
	other := f.createUser(t, "other@example.com")
	group := f.seedGroup(t, owner.Id, member.Id)
	if err := f.stores.Memberships.Create(context.Background(), &domain.Membership{GroupID: group.ID, UserID: other.Id, Role: domain.RoleMember}); err != nil {
		t.Fatal(err)
	}
	f.createSettlement(t, group.ID, member.Id, member.Id, owner.Id, 500)
	f.createSettlement(t, group.ID, owner.Id, owner.Id, other.Id, 700)
	scenario := tests.ApiScenario{
		Name:                  "filtering settlements by memberId returns only that member's settlements",
		Method:                http.MethodGet,
		URL:                   "/api/subflow/v1/groups/" + group.ID + "/settlements?memberId=" + member.Id,
		Headers:               map[string]string{"Authorization": f.token(t, owner)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"totalItems":1`, `"amountMinor":500`},
		NotExpectedContent:    []string{`"amountMinor":700`},
	}
	scenario.Test(t)
}

func TestListSettlementsFiltersByDateRange(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	member := f.createUser(t, "member@example.com")
	group := f.seedGroup(t, owner.Id, member.Id)
	f.createSettlement(t, group.ID, member.Id, member.Id, owner.Id, 500)
	scenario := tests.ApiScenario{
		Name:                  "a date range excludes settlements outside the requested local date",
		Method:                http.MethodGet,
		URL:                   "/api/subflow/v1/groups/" + group.ID + "/settlements?from=2026-08-02&to=2026-08-02",
		Headers:               map[string]string{"Authorization": f.token(t, owner)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"totalItems":0`},
	}
	scenario.Test(t)
}

func TestListSettlementsRejectsInvalidFilters(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "owner@example.com")
	group := f.seedGroup(t, owner.Id, "")
	scenario := tests.ApiScenario{
		Name:                  "an unsupported settlement sort is rejected",
		Method:                http.MethodGet,
		URL:                   "/api/subflow/v1/groups/" + group.ID + "/settlements?sort=amount_minor",
		Headers:               map[string]string{"Authorization": f.token(t, owner)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusBadRequest,
		ExpectedContent:       []string{`"code":"invalid_request"`},
	}
	scenario.Test(t)
}

// createSettlement seeds a settlement directly through the service (bypassing
// HTTP) so tests that PATCH/DELETE/list by ID have a known fixture, mirroring
// the *_api_test.go convention of seeding via stores/service directly.
func (f *apiFixture) createSettlement(t *testing.T, groupID, actorID, fromUserID, toUserID string, amountMinor int64) *domain.Settlement {
	t.Helper()
	settledOn, err := time.Parse(time.RFC3339, "2026-08-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	created, err := f.service.CreateSettlement(context.Background(), actorID, domain.Settlement{
		GroupID: groupID, FromUserID: fromUserID, ToUserID: toUserID, AmountMinor: amountMinor, SettledOn: settledOn,
	})
	if err != nil {
		t.Fatal(err)
	}
	return created
}
