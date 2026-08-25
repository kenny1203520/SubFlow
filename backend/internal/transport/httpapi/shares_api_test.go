package httpapi_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"
	"subflow/internal/application"
	"subflow/internal/domain"
	"subflow/internal/ports"
)

func TestPublicShareWithAllHistoryReturnsPage(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "share-owner@example.com")
	created, err := f.service.CreateShare(context.Background(), owner.Id, "", application.ShareInput{Share: domain.Share{Name: "Public", AccessMode: "link", RangeMode: "all", ShowSummary: true, ShowExpenses: true, ShowSubscriptions: true}})
	if err != nil {
		t.Fatal(err)
	}
	token := strings.TrimPrefix(created.URL, "/share/")
	scenario := tests.ApiScenario{
		Name:                  "public share returns its read-only projection",
		Method:                http.MethodGet,
		URL:                   "/api/subflow/v1/shares/" + token,
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"name":"Public"`},
	}
	scenario.Test(t)
}
func TestPublicGroupShareWithAllHistoryReturnsPage(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "group-share-owner@example.com")
	group := f.seedGroup(t, owner.Id, "")
	created, err := f.service.CreateShare(context.Background(), owner.Id, group.ID, application.ShareInput{Share: domain.Share{Name: "Group Public", AccessMode: "link", RangeMode: "all", ShowSummary: true, ShowExpenses: true, ShowSubscriptions: true, ShowSettlements: true}})
	if err != nil {
		t.Fatal(err)
	}
	token := strings.TrimPrefix(created.URL, "/share/")
	scenario := tests.ApiScenario{Name: "public group share returns its read-only projection", Method: http.MethodGet, URL: "/api/subflow/v1/shares/" + token, TestAppFactory: f.factory(), BeforeTestFunc: f.beforeTest(), DisableTestAppCleanup: true, ExpectedStatus: http.StatusOK, ExpectedContent: []string{`"name":"Group Public"`}}
	scenario.Test(t)
}
func TestShareURLPersistsAcrossManagementOperations(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "persist@example.com")
	ctx := context.Background()
	created, err := f.service.CreateShare(ctx, owner.Id, "", application.ShareInput{Share: domain.Share{Name: "Persistent", AccessMode: "link", RangeMode: "all", ShowSummary: true, ShowExpenses: true, ShowSubscriptions: true}})
	if err != nil {
		t.Fatal(err)
	}
	original := created.URL
	if original == "" || created.TokenCiphertext == "" {
		t.Fatal("created URL was not encrypted")
	}
	page, err := f.service.ListShares(ctx, owner.Id, "", ports.PageRequest{Page: 1, PerPage: 25})
	if err != nil {
		t.Fatal(err)
	}
	if page.Items[0].URL != original {
		t.Fatalf("listed URL = %q, want %q", page.Items[0].URL, original)
	}
	edited := *created.Share
	edited.Name = "Edited"
	updated, err := f.service.UpdateShare(ctx, owner.Id, created.ID, application.ShareInput{Share: edited})
	if err != nil {
		t.Fatal(err)
	}
	if updated.URL != original {
		t.Fatalf("updated URL = %q, want %q", updated.URL, original)
	}
	legacy := *updated
	legacy.TokenCiphertext = ""
	if err = f.stores.Shares.Update(ctx, &legacy); err != nil {
		t.Fatal(err)
	}
	token := strings.TrimPrefix(original, "/share/")
	remembered, err := f.service.RememberShareToken(ctx, owner.Id, created.ID, token)
	if err != nil {
		t.Fatal(err)
	}
	if remembered.URL != original {
		t.Fatalf("remembered URL = %q, want %q", remembered.URL, original)
	}
	rotated, err := f.service.RotateShare(ctx, owner.Id, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.URL == original {
		t.Fatal("rotation kept the old URL")
	}
	if _, err = f.service.FindAvailableShare(ctx, token); err != domain.ErrNotFound {
		t.Fatalf("old URL lookup = %v, want ErrNotFound", err)
	}
	if _, err = f.service.FindAvailableShare(ctx, strings.TrimPrefix(rotated.URL, "/share/")); err != nil {
		t.Fatal(err)
	}
}
