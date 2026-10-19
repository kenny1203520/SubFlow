package httpapi_test

import (
	"context"
	"github.com/pocketbase/pocketbase/tests"
	"net/http"
	"strings"
	"subflow/internal/domain"
	"testing"
	"time"
)

func TestIncomeHTTPPatchAndGroupBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, body      string
		group, mismatch bool
		status          int
		content         []string
	}{
		{"personal clear", `{"amountMinor":0,"notes":"","categoryId":""}`, false, false, 200, []string{`"amountMinor":0`, `"notes":""`, `"title":"Salary"`}},
		{"group clear", `{"notes":""}`, true, false, 200, []string{`"notes":""`, `"amountMinor":10000`}},
		{"wrong group", `{"notes":"changed"}`, true, true, 403, []string{`"code":"forbidden"`}},
		{"invalid body", `{"amountMinor":"wrong"}`, false, false, 400, []string{`"code":"invalid_request"`}},
		{"blank title", `{"title":""}`, false, false, 400, []string{`"code":"invalid_request"`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newAPITestApp(t)
			owner := f.createUser(t, "income-owner@example.com")
			input := domain.Income{Title: "Salary", AmountMinor: 10000, Currency: domain.CurrencyTWD, ReceivedOn: time.Now().UTC(), Notes: "old"}
			var value *domain.Income
			var err error
			prefix := "/api/subflow/v1"
			if tc.group {
				g := f.seedGroup(t, owner.Id, "")
				input.GroupID = g.ID
				value, err = f.service.CreateGroupIncome(context.Background(), owner.Id, input)
				prefix += "/groups/" + g.ID
			} else {
				value, err = f.service.CreateIncome(context.Background(), owner.Id, input)
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.mismatch {
				other := f.seedGroup(t, owner.Id, "")
				prefix = "/api/subflow/v1/groups/" + other.ID
			}
			scenario := tests.ApiScenario{Name: tc.name, Method: http.MethodPatch, URL: prefix + "/incomes/" + value.ID, Body: strings.NewReader(tc.body), Headers: map[string]string{"Authorization": f.token(t, owner)}, TestAppFactory: f.factory(), BeforeTestFunc: f.beforeTest(), DisableTestAppCleanup: true, ExpectedStatus: tc.status, ExpectedContent: tc.content}
			scenario.Test(t)
		})
	}
}

func TestGroupIncomeAcceptsDocumentedSort(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "income-sort@example.com")
	g := f.seedGroup(t, owner.Id, "")
	scenario := tests.ApiScenario{Name: "income sort", Method: http.MethodGet, URL: "/api/subflow/v1/groups/" + g.ID + "/incomes?sort=-received_on", Headers: map[string]string{"Authorization": f.token(t, owner)}, TestAppFactory: f.factory(), BeforeTestFunc: f.beforeTest(), DisableTestAppCleanup: true, ExpectedStatus: 200, ExpectedContent: []string{`"data":[]`}}
	scenario.Test(t)
}
