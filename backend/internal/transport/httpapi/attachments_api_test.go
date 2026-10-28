package httpapi_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"ganigeorgiev.com/pocketbase/pocketbase/apis"
	"ganigeorgiev.com/pocketbase/pocketbase/core"
	"strings"
	"subflow/internal/domain"
	"subflow/internal/ports"
	"subflow/internal/transport/httpapi"
)

func TestAttachmentHTTPPersonalExpenseAndIncomeLifecycle(t *testing.T) {
	for _, typ := range []string{"expense", "income"} {
		t.Run(typ, func(t *testing.T) {
			f := newAPITestApp(t)
			owner := f.createUser(t, "attachment-owner@example.com")
			outsider := f.createUser(t, "attachment-outsider@example.com")
			ctx := context.Background()
			id := ""
			if typ == "expense" {
				v := domain.Expense{OwnerID: owner.Id, Title: "Receipt", AmountMinor: 100, Currency: domain.CurrencyTWD, PaidBy: owner.Id, IncurredOn: time.Now().UTC()}
				if err := f.stores.Expenses.Create(ctx, &v); err != nil {
					t.Fatal(err)
				}
				id = v.ID
			} else {
				v, err := f.service.CreateIncome(ctx, owner.Id, domain.Income{Title: "Receipt", AmountMinor: 100, Currency: domain.CurrencyTWD, ReceivedOn: time.Now().UTC()})
				if err != nil {
					t.Fatal(err)
				}
				id = v.ID
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
			request := func(name, method, path, token string, body *bytes.Buffer, contentType string, status int, content ...string) {
				t.Helper()
				headers := map[string]string{"Authorization": "Bearer " + token}
				if contentType != "" {
					headers["Content-Type"] = contentType
				}
				if body == nil {
					body = &bytes.Buffer{}
				}
				req := httptest.NewRequest(method, "/api/subflow/v1"+path, body)
				for k, v := range headers {
					req.Header.Set(k, v)
				}
				res := httptest.NewRecorder()
				mux.ServeHTTP(res, req)
				if res.Code != status {
					t.Fatalf("%s: status %d want %d: %s", name, res.Code, status, res.Body.String())
				}
				for _, want := range content {
					if !strings.Contains(res.Body.String(), want) {
						t.Fatalf("%s: missing %q: %s", name, want, res.Body.String())
					}
				}
			}
			payload := []byte("%PDF-1.7\nreceipt content\n%%EOF")
			upload := func(token, requestID string, status int) {
				t.Helper()
				var body bytes.Buffer
				w := multipart.NewWriter(&body)
				for k, v := range map[string]string{"targetType": typ, "targetId": id, "requestId": requestID} {
					if err := w.WriteField(k, v); err != nil {
						t.Fatal(err)
					}
				}
				p, err := w.CreateFormFile("file", "receipt.pdf")
				if err != nil {
					t.Fatal(err)
				}
				if _, err = p.Write(payload); err != nil {
					t.Fatal(err)
				}
				if err = w.Close(); err != nil {
					t.Fatal(err)
				}
				request("upload", http.MethodPost, "/attachments", token, &body, w.FormDataContentType(), status)
			}
			token := f.token(t, owner)
			upload(token, "upload-1", 201)
			upload(token, "upload-1", 201)
			upload(token, "", 400)
			items, err := f.stores.Attachments.List(ctx, typ, id, time.Time{})
			if err != nil || len(items) != 1 {
				t.Fatalf("stored attachments: %v %v", items, err)
			}
			aid := items[0].ID
			query := "/attachments?" + url.Values{"targetType": {typ}, "targetId": {id}}.Encode()
			request("list after upload", http.MethodGet, query, token, nil, "", 200, `"fileName":"receipt.pdf"`, `"canRead":true`)
			request("download", http.MethodGet, "/attachments/"+aid+"/content", token, nil, "", 200, string(payload))
			strangerToken := f.token(t, outsider)
			request("outsider list", http.MethodGet, query, strangerToken, nil, "", 403)
			upload(strangerToken, "denied", 403)
			request("outsider download", http.MethodGet, "/attachments/"+aid+"/content", strangerToken, nil, "", 403)
			request("outsider delete", http.MethodDelete, "/attachments/"+aid, strangerToken, nil, "", 403)
			request("reopen", http.MethodGet, query, token, nil, "", 200, `"fileName":"receipt.pdf"`)
			request("delete", http.MethodDelete, "/attachments/"+aid, token, nil, "", 200)
			request("list after delete", http.MethodGet, query, token, nil, "", 200, `"items":[]`)
			logs, err := f.stores.Audits.List(ctx, "", ports.AuditQuery{Action: "attachment.uploaded", Outcome: "failure", PageRequest: ports.PageRequest{Page: 1, PerPage: 100}})
			if err != nil || logs.TotalItems < 2 {
				t.Fatalf("missing invalid request and permission audits: %v %v", logs, err)
			}
		})
	}
}
