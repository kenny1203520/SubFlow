package httpapi

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"ganigeorgiev.com/pocketbase/pocketbase/core"

	"subflow/internal/adapters"
	"subflow/internal/application"
	"subflow/internal/domain"
)

func newShareCookieEvent(t *testing.T, directTLS bool, applicationURL, forwardedProto string) (*API, *core.RequestEvent, *httptest.ResponseRecorder, *domain.Share) {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/subflow/v1/shares/share-id/password", nil)
	if directTLS {
		request.TLS = &tls.ConnectionState{}
	}
	if forwardedProto != "" {
		request.Header.Set("X-Forwarded-Proto", forwardedProto)
	}
	service := application.New(adapters.Stores{}, "test-cookie-root-key")
	api := &API{Service: service, ApplicationURL: func() string { return applicationURL }}
	event := &core.RequestEvent{}
	event.Request = request
	event.Response = recorder
	share := &domain.Share{ID: "share-id", AccessVersion: 3}
	api.setShareCookie(event, share)
	return api, event, recorder, share
}

func TestShareCookieSecureRequiresDirectOrConfiguredHTTPS(t *testing.T) {
	tests := []struct {
		name           string
		directTLS      bool
		applicationURL string
		forwardedProto string
		wantSecure     bool
	}{
		{name: "direct TLS", directTLS: true, applicationURL: "http://subflow.example.test", wantSecure: true},
		{name: "PocketBase public HTTPS URL", applicationURL: "https://subflow.example.test", wantSecure: true},
		{name: "explicit HTTP stays compatible", applicationURL: "http://subflow.example.test", wantSecure: false},
		{name: "forwarded header is not trusted", applicationURL: "http://subflow.example.test", forwardedProto: "https", wantSecure: false},
		{name: "missing URL remains HTTP", wantSecure: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, recorder, _ := newShareCookieEvent(t, test.directTLS, test.applicationURL, test.forwardedProto)
			cookies := recorder.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("issued %d cookies, want one", len(cookies))
			}
			if cookies[0].Secure != test.wantSecure {
				t.Fatalf("Secure = %v, want %v", cookies[0].Secure, test.wantSecure)
			}
		})
	}
}

func TestNewShareCookieValidAndLegacySignatureRejected(t *testing.T) {
	api, event, recorder, share := newShareCookieEvent(t, false, "https://subflow.example.test", "")
	issued := recorder.Result().Cookies()
	if len(issued) != 1 {
		t.Fatalf("issued %d cookies, want one", len(issued))
	}
	parts := strings.Split(issued[0].Value, ".")
	if len(parts) != 5 || parts[0] != "v1" {
		t.Fatalf("share cookie = %q, want v1 plus four signed fields", issued[0].Value)
	}
	event.Request.AddCookie(issued[0])
	if !shareCookieValid(event, share, api.Service.ShareKey) {
		t.Fatal("cookie signed with the current purpose key should validate")
	}

	legacyExpiry := time.Now().Add(time.Hour).Unix()
	legacyPayload := share.ID + "." + strconv.Itoa(share.AccessVersion) + "." + strconv.FormatInt(legacyExpiry, 10)
	legacyValue := legacyPayload + "." + shareCookieSignature(legacyPayload, []byte("legacy-share-cookie-key"))
	for _, test := range []struct {
		name  string
		value string
	}{
		{name: "legacy four-part cookie", value: legacyValue},
		{name: "unknown version", value: signedShareCookie("v2", share.ID, share.AccessVersion, legacyExpiry, api.Service.ShareKey)},
		{name: "tampered signature", value: strings.Join(append(append([]string(nil), parts[:4]...), "tampered"), ".")},
		{name: "expired cookie", value: signedShareCookie("v1", share.ID, share.AccessVersion, time.Now().Add(-time.Hour).Unix(), api.Service.ShareKey)},
		{name: "different access version", value: signedShareCookie("v1", share.ID, share.AccessVersion+1, legacyExpiry, api.Service.ShareKey)},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/subflow/v1/shares/share-id/password", nil)
			request.AddCookie(&http.Cookie{Name: shareCookieName, Value: test.value})
			invalidEvent := &core.RequestEvent{}
			invalidEvent.Request = request
			if shareCookieValid(invalidEvent, share, api.Service.ShareKey) {
				t.Fatalf("share cookie %q unexpectedly validated", test.value)
			}
		})
	}
}

func signedShareCookie(version, id string, accessVersion int, expiry int64, key []byte) string {
	payload := strings.Join([]string{version, id, strconv.Itoa(accessVersion), strconv.FormatInt(expiry, 10)}, ".")
	return payload + "." + shareCookieSignature(payload, key)
}
