package web

import (
	"net/http"
	"strings"
	"testing"
)

func TestHashedAssetUsesImmutableCache(t *testing.T) {
	e, response := event(http.MethodGet, "/assets/app.123456.js")
	if err := serve(e, "/assets/app.123456.js"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(response.Body.String(), "SubFlow development placeholder") {
		t.Fatal("embedded asset was not served")
	}
	if got := response.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("unexpected cache header %q", got)
	}
}
