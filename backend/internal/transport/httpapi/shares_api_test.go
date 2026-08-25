package httpapi_test

import (
  "context"
  "net/http"
  "strings"
  "testing"

  "github.com/pocketbase/pocketbase/tests"
  "subflow/internal/application"
  "subflow/internal/domain"
)

func TestPublicShareWithAllHistoryReturnsPage(t *testing.T) {
  f := newAPITestApp(t)
  owner := f.createUser(t, "share-owner@example.com")
  created, err := f.service.CreateShare(context.Background(), owner.Id, "", application.ShareInput{Share: domain.Share{Name: "Public", AccessMode: "link", RangeMode: "all", ShowSummary: true, ShowExpenses: true, ShowSubscriptions: true}})
  if err != nil { t.Fatal(err) }
  token := strings.TrimPrefix(created.URL, "/share/")
  scenario := tests.ApiScenario{
    Name: "public share returns its read-only projection",
    Method: http.MethodGet,
    URL: "/api/subflow/v1/shares/" + token,
    TestAppFactory: f.factory(),
    BeforeTestFunc: f.beforeTest(),
    DisableTestAppCleanup: true,
    ExpectedStatus: http.StatusOK,
    ExpectedContent: []string{`"name":"Public"`},
  }
  scenario.Test(t)
}
func TestPublicGroupShareWithAllHistoryReturnsPage(t *testing.T) {
  f := newAPITestApp(t)
  owner := f.createUser(t, "group-share-owner@example.com")
  group := f.seedGroup(t, owner.Id, "")
  created, err := f.service.CreateShare(context.Background(), owner.Id, group.ID, application.ShareInput{Share: domain.Share{Name: "Group Public", AccessMode: "link", RangeMode: "all", ShowSummary: true, ShowExpenses: true, ShowSubscriptions: true, ShowSettlements: true}})
  if err != nil { t.Fatal(err) }
  token := strings.TrimPrefix(created.URL, "/share/")
  scenario := tests.ApiScenario{Name: "public group share returns its read-only projection", Method: http.MethodGet, URL: "/api/subflow/v1/shares/" + token, TestAppFactory: f.factory(), BeforeTestFunc: f.beforeTest(), DisableTestAppCleanup: true, ExpectedStatus: http.StatusOK, ExpectedContent: []string{`"name":"Group Public"`}}
  scenario.Test(t)
}