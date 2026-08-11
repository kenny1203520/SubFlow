package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func event(method, target string) (*core.RequestEvent, *httptest.ResponseRecorder) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, nil)
	return &core.RequestEvent{Event: router.Event{Response: response, Request: request}}, response
}
func TestEmbeddedIndexAndSPAFallback(t *testing.T) {
	for _, target := range []string{"/", "/invite/test-token"} {
		e, response := event(http.MethodGet, target)
		if err := serve(e, target); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "SubFlow backend is running") {
			t.Fatalf("unexpected response for %s: %d %s", target, response.Code, response.Body.String())
		}
		if response.Header().Get("Cache-Control") != "no-cache" {
			t.Fatal("index must not be cached")
		}
		if response.Header().Get("Content-Security-Policy") == "" || response.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatal("security headers missing")
		}
	}
}
func TestReservedAndMissingAssetReturnNotFound(t *testing.T) {
	for _, target := range []string{"/api/unknown", "/_/unknown", "/assets/missing.js"} {
		e, _ := event(http.MethodGet, target)
		err := serve(e, target)
		apiErr, ok := err.(*router.ApiError)
		if !ok || apiErr.Status != http.StatusNotFound {
			t.Fatalf("%s: %#v", target, err)
		}
	}
}
func TestHeadReturnsNoBody(t *testing.T) {
	e, response := event(http.MethodHead, "/")
	if err := serve(e, "/"); err != nil {
		t.Fatal(err)
	}
	if response.Body.Len() != 0 {
		t.Fatal("HEAD response must not include a body")
	}
}
