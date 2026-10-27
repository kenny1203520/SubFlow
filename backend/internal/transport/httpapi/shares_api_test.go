package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"ganigeorgiev.com/pocketbase/pocketbase/tests"
	"subflow/internal/application"
	"subflow/internal/domain"
	"subflow/internal/ports"
	"subflow/internal/security"
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

func TestPersonalShareFixedRangeReportsMissingEndDateAndAcceptsSameDay(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "fixed-share@example.com")
	token := f.token(t, owner)
	body := `{"name":"Test","accessMode":"link","password":"","viewerEmails":[],"enabled":true,"rangeMode":"fixed","rollingDays":30,"showSummary":true,"showIncomes":true,"showRecurringIncome":true,"showPendingIncome":true,"showExpenses":true,"showSubscriptions":true,"showSettlements":true,"showIdentities":true,"showNotes":true,"startsOn":"2026-08-31T16:00:00.000Z"}`
	scenario := tests.ApiScenario{Name: "fixed share identifies the missing end date", Method: http.MethodPost, URL: "/api/subflow/v1/personal/shares", Body: strings.NewReader(body), Headers: map[string]string{"Authorization": "Bearer " + token}, TestAppFactory: f.factory(), BeforeTestFunc: f.beforeTest(), DisableTestAppCleanup: true, ExpectedStatus: http.StatusBadRequest, ExpectedContent: []string{`"code":"invalid_request"`, `"endsOn":"required"`}}
	scenario.Test(t)
	f = newAPITestApp(t)
	owner = f.createUser(t, "fixed-share-success@example.com")
	token = f.token(t, owner)
	body = strings.TrimSuffix(body, `,"startsOn":"2026-08-31T16:00:00.000Z"}`) + `,"startsOn":"2026-08-31T16:00:00.000Z","endsOn":"2026-08-31T16:00:00.000Z"}`
	body = strings.Replace(body, `"enabled":true`, `"enabled":false`, 1)
	scenario = tests.ApiScenario{Name: "fixed share accepts a one-day range and respects disabled status", Method: http.MethodPost, URL: "/api/subflow/v1/personal/shares", Body: strings.NewReader(body), Headers: map[string]string{"Authorization": "Bearer " + token}, TestAppFactory: f.factory(), BeforeTestFunc: f.beforeTest(), DisableTestAppCleanup: true, ExpectedStatus: http.StatusCreated, ExpectedContent: []string{`"name":"Test"`, `"url":"/share/`, `"enabled":false`}}
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

func TestShareUpdatePreservesEnabledWhenFieldIsOmitted(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "share-enabled-omitted@example.com")
	ctx := context.Background()
	created, err := f.service.CreateShare(ctx, owner.Id, "", application.ShareInput{Share: domain.Share{Name: "Enabled", AccessMode: "link", RangeMode: "all", ShowExpenses: true}})
	if err != nil {
		t.Fatal(err)
	}
	var input application.ShareInput
	if err := json.Unmarshal([]byte(`{"name":"Renamed","accessMode":"link","rangeMode":"all","showExpenses":true}`), &input); err != nil {
		t.Fatal(err)
	}
	updated, err := f.service.UpdateShare(ctx, owner.Id, created.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Enabled {
		t.Fatal("omitting enabled during edit should preserve the existing enabled status")
	}
	input.EnabledInput = new(bool)
	updated, err = f.service.UpdateShare(ctx, owner.Id, created.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Enabled {
		t.Fatal("explicit false should disable the share")
	}
}

func TestShareListDistinguishesMissingKeyFromUnreadableCiphertext(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "share-key-state@example.com")
	if _, err := f.service.CreateShare(context.Background(), owner.Id, "", application.ShareInput{Share: domain.Share{Name: "Encrypted", AccessMode: "link", RangeMode: "all", ShowExpenses: true}}); err != nil {
		t.Fatal(err)
	}
	pageRequest := ports.PageRequest{Page: 1, PerPage: 25}
	f.service.Cipher = security.NewSettingsCipher("")
	if _, err := f.service.ListShares(context.Background(), owner.Id, "", pageRequest); !errors.Is(err, domain.ErrShareKeyMissing) {
		t.Fatalf("missing key error = %v", err)
	}
	f.service.Cipher = security.NewSettingsCipher("different-key")
	if _, err := f.service.ListShares(context.Background(), owner.Id, "", pageRequest); !errors.Is(err, domain.ErrShareCiphertext) {
		t.Fatalf("unreadable ciphertext error = %v", err)
	}
}

func TestShareListReturnsActionableConfigurationDetail(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "share-config-response@example.com")
	if _, err := f.service.CreateShare(context.Background(), owner.Id, "", application.ShareInput{Share: domain.Share{Name: "Encrypted", AccessMode: "link", RangeMode: "all", ShowExpenses: true}}); err != nil {
		t.Fatal(err)
	}
	f.service.Cipher = security.NewSettingsCipher("")
	scenario := tests.ApiScenario{
		Name:                  "share listing identifies a missing encryption key",
		Method:                http.MethodGet,
		URL:                   "/api/subflow/v1/personal/shares",
		Headers:               map[string]string{"Authorization": "Bearer " + f.token(t, owner)},
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusServiceUnavailable,
		ExpectedContent:       []string{`"code":"configuration_error"`, `"details":"share_key_missing"`},
	}
	scenario.Test(t)
}

func TestShareCanExposeRecurringAndPendingIncomeIndependently(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "income-share@example.com")
	ctx := context.Background()
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	f.service.Now = func() time.Time { return now }
	start := now.AddDate(0, 0, -1)
	_, err := f.service.CreateRecurringIncome(ctx, owner.Id, domain.RecurringIncome{
		OwnerID:  owner.Id,
		Template: domain.Income{Title: "Private plan", AmountMinor: 120000, Currency: domain.CurrencyTWD, EarnedBy: owner.Id},
		Timezone: "UTC", Cycle: domain.BillingMonthly, Interval: 1, StartsOn: start, NextOn: start, PostingMode: "manual",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.PostDueRecurringIncomes(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.CreateIncome(ctx, owner.Id, domain.Income{Title: "Hidden actual income", AmountMinor: 5000, Currency: domain.CurrencyTWD, EarnedBy: owner.Id, ReceivedOn: now}); err != nil {
		t.Fatal(err)
	}
	created, err := f.service.CreateShare(ctx, owner.Id, "", application.ShareInput{Share: domain.Share{
		Name: "Income plan", AccessMode: "link", RangeMode: "all", ShowSummary: true,
		ShowRecurringIncome: true, ShowPendingIncome: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	page, err := f.service.SharePageWithOptions(ctx, created.Share, application.SharePageOptions{Page: 1, PerPage: 25})
	if err != nil {
		t.Fatal(err)
	}
	if page.TotalItems != 2 || page.RecordCounts["recurring_income"] != 1 || page.RecordCounts["pending_income"] != 1 {
		t.Fatalf("share should include only the selected plan and pending period: %#v", page.RecordCounts)
	}
	if _, ok := page.Summary["incomeCount"]; ok {
		t.Fatal("summary leaked a disabled actual-income total")
	}
	if page.Summary["recurringIncomeCount"] != 1 || page.Summary["pendingIncomeCount"] != 1 {
		t.Fatalf("share summary should describe selected forecast content: %#v", page.Summary)
	}
	for _, record := range page.Records {
		if record["type"] == "income" || record["title"] == "Hidden actual income" {
			t.Fatalf("share leaked actual income while disabled: %#v", record)
		}
	}
}
