package httpapi_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"subflow/internal/domain"
	"subflow/internal/transport/httpapi"
)

func TestAttachmentPeriodHTTPListsPostedExpenseAndEnforcesCount(t *testing.T) {
	for _, typ := range []string{"subscription_period", "expense"} {
		t.Run(typ, func(t *testing.T) {
			f := newAPITestApp(t)
			user := f.createUser(t, "period-attachment@example.com")
			ctx := context.Background()
			at := time.Date(2026, 8, 16, 0, 0, 0, 123000000, time.UTC)
			sub := domain.Subscription{OwnerID: user.Id, PaidBy: user.Id, Name: "Monthly", Currency: domain.CurrencyTWD, AmountMinor: 100, BillingCycle: domain.BillingMonthly, BillingInterval: 1, StartsOn: at, NextBilling: at, Status: domain.SubscriptionActive, SplitMode: domain.SplitAmount}
			if err := f.stores.Subscriptions.Create(ctx, &sub); err != nil {
				t.Fatal(err)
			}
			id := sub.ID
			if typ == "expense" {
				v := domain.Expense{OwnerID: user.Id, PaidBy: user.Id, Title: "Posted expense", SubscriptionID: sub.ID, Currency: domain.CurrencyTWD, AmountMinor: 100, IncurredOn: at}
				if err := f.stores.Expenses.Create(ctx, &v); err != nil {
					t.Fatal(err)
				}
				id = v.ID
			}
			settings, err := f.stores.Settings.Get(ctx)
			if err != nil {
				t.Fatal(err)
			}
			settings.AttachmentDefaultMaxCount = 1
			if err = f.stores.Settings.Save(ctx, settings); err != nil {
				t.Fatal(err)
			}
			router, err := apis.NewRouter(f.app)
			if err != nil {
				t.Fatal(err)
			}
			(&httpapi.API{Service: f.service}).RegisterRoutes(&core.ServeEvent{App: f.app, Router: router})
			mux, err := router.BuildMux()
			if err != nil {
				t.Fatal(err)
			}
			token := f.token(t, user)
			request := func(method, path string, body *bytes.Buffer, contentType string, want int, contains string) *httptest.ResponseRecorder {
				t.Helper()
				if body == nil {
					body = &bytes.Buffer{}
				}
				req := httptest.NewRequest(method, "/api/subflow/v1"+path, body)
				req.Header.Set("Authorization", "Bearer "+token)
				if contentType != "" {
					req.Header.Set("Content-Type", contentType)
				}
				res := httptest.NewRecorder()
				mux.ServeHTTP(res, req)
				if res.Code != want || !strings.Contains(res.Body.String(), contains) {
					t.Fatalf("%s %s: %d %s", method, path, res.Code, res.Body.String())
				}
				return res
			}
			upload := func(rid string, want int) {
				t.Helper()
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				fields := map[string]string{"targetType": typ, "targetId": id, "requestId": rid}
				if typ == "subscription_period" {
					fields["billingAt"] = at.In(time.FixedZone("Taipei", 8*3600)).Format(time.RFC3339Nano)
				}
				for k, v := range fields {
					if err := writer.WriteField(k, v); err != nil {
						t.Fatal(err)
					}
				}
				part, err := writer.CreateFormFile("file", "period.pdf")
				if err != nil {
					t.Fatal(err)
				}
				_, _ = part.Write([]byte("%PDF-1.7 period"))
				if err = writer.Close(); err != nil {
					t.Fatal(err)
				}
				request(http.MethodPost, "/attachments", &body, writer.FormDataContentType(), want, "")
			}
			upload("first", 201)
			query := url.Values{"targetType": {typ}, "targetId": {id}}
			if typ == "subscription_period" {
				query.Set("billingAt", at.Format(time.RFC3339Nano))
			}
			request(http.MethodGet, "/attachments?"+query.Encode(), nil, "", 200, `"fileName":"period.pdf"`)
			request(http.MethodGet, "/attachments?"+query.Encode(), nil, "", 200, `"fileName":"period.pdf"`)
			upload("first", 201)
			upload("second", 409)
			items, err := f.stores.Attachments.List(ctx, "subscription_period", sub.ID, at.In(time.FixedZone("West", -5*3600)))
			if err != nil || len(items) != 1 {
				t.Fatalf("period items: %v %v", items, err)
			}
			request(http.MethodGet, "/attachments/"+items[0].ID+"/content", nil, "", 200, "%PDF-1.7 period")
		})
	}
}

func TestAttachmentPeriodDeleteAndReassignOnlySelectedPeriod(t *testing.T) {
	for _, op := range []string{"delete", "reassign"} {
		t.Run(op, func(t *testing.T) {
			f := newAPITestApp(t)
			u := f.createUser(t, "period-repository@example.com")
			ctx := context.Background()
			at := time.Date(2026, 8, 16, 0, 0, 0, 123000000, time.UTC)
			for i, date := range []time.Time{at, at.AddDate(0, 1, 0), {}} {
				a := domain.Attachment{OwnerID: u.Id, UploadedBy: u.Id, TargetType: "subscription_period", TargetID: "subscription", BillingAt: date, FileName: "receipt.pdf", MediaType: "application/pdf", RequestID: []string{"a", "b", "c"}[i]}
				if err := f.stores.Attachments.Create(ctx, &a, []byte("%PDF-1.7")); err != nil {
					t.Fatal(err)
				}
			}
			offset := at.In(time.FixedZone("Taipei", 8*3600))
			if op == "delete" {
				if err := f.stores.Attachments.DeletePeriod(ctx, "subscription_period", "subscription", offset); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := f.stores.Attachments.ReassignPeriod(ctx, "subscription_period", "subscription", offset, "expense", "posted-expense"); err != nil {
					t.Fatal(err)
				}
				moved, err := f.stores.Attachments.List(ctx, "expense", "posted-expense", time.Time{})
				if err != nil || len(moved) != 1 {
					t.Fatalf("moved items: %v %v", moved, err)
				}
			}
			all, err := f.stores.Attachments.List(ctx, "subscription_period", "subscription", time.Time{})
			if err != nil || len(all) != 2 {
				t.Fatalf("unrelated periods changed or selected period not found: %v %v", all, err)
			}
		})
	}
}
