package httpapi_test

import (
	"context"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"subflow/internal/application"
	"subflow/internal/domain"
	"subflow/internal/transport/httpapi"
)

func TestAttachmentRenameHTTPAndDownload(t *testing.T) {
	f := newAPITestApp(t)
	u := f.createUser(t, "rename-owner@example.com")
	other := f.createUser(t, "rename-other@example.com")
	ctx := context.Background()
	income, err := f.service.CreateIncome(ctx, u.Id, domain.Income{Title: "Income", AmountMinor: 100, Currency: domain.CurrencyTWD, ReceivedOn: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	a, err := f.service.UploadAttachment(ctx, u.Id, application.AttachmentTarget{Type: "income", ID: income.ID, RequestID: "rename"}, "original.pdf", []byte("%PDF-1.7 original"))
	if err != nil {
		t.Fatal(err)
	}
	before, err := f.app.FindRecordById("attachments", a.ID)
	if err != nil {
		t.Fatal(err)
	}
	blobName := before.GetString("file")
	router, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	(&httpapi.API{Service: f.service}).RegisterRoutes(&core.ServeEvent{App: f.app, Router: router})
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path, body, token string, want int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, "/api/subflow/v1"+path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)
		if res.Code != want {
			t.Fatalf("%s %q: %d want %d: %s", method, body, res.Code, want, res.Body.String())
		}
		return res
	}
	token := f.token(t, u)
	path := "/attachments/" + a.ID
	request(http.MethodPatch, path, `{"fileName":"八月發票.pdf"}`, f.token(t, other), 403)
	for _, body := range []string{`{}`, `{"fileName":null}`, `{"fileName":""}`, `{"fileName":".pdf"}`, `{"fileName":"../invoice.pdf"}`, `{"fileName":"bad\\path.pdf"}`, `{"fileName":"bad\nname.pdf"}`, `{"fileName":"invoice.exe"}`, `{"fileName":"invoice.pdf","ownerId":"other"}`, `{"fileName":"a.pdf"} {"fileName":"b.pdf"}`, `{"fileName":"` + strings.Repeat("a", 252) + `.pdf"}`} {
		request(http.MethodPatch, path, body, token, 400)
	}
	response := request(http.MethodPatch, path, `{"fileName":"八月發票.pdf"}`, token, 200)
	if !strings.Contains(response.Body.String(), `"fileName":"八月發票.pdf"`) {
		t.Fatal(response.Body.String())
	}
	request(http.MethodPatch, path, `{"fileName":"八月發票.pdf"}`, token, 200)
	listed := request(http.MethodGet, "/attachments?targetType=income&targetId="+income.ID, "", token, 200)
	if !strings.Contains(listed.Body.String(), `"canRename":true`) || !strings.Contains(listed.Body.String(), `"fileName":"八月發票.pdf"`) {
		t.Fatal(listed.Body.String())
	}
	content := request(http.MethodGet, path+"/content", "", token, 200)
	_, params, err := mime.ParseMediaType(content.Header().Get("Content-Disposition"))
	if err != nil || params["filename"] != "八月發票.pdf" || content.Body.String() != "%PDF-1.7 original" {
		t.Fatalf("download metadata/content mismatch: %v %v %s", params, err, content.Body.String())
	}
	after, err := f.app.FindRecordById("attachments", a.ID)
	if err != nil || after.GetString("file") != blobName {
		t.Fatalf("physical file was changed: %v", err)
	}
}
