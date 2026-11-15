package upgrade

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"subflow/internal/security"
)

func TestPublicStatusExposesOnlySafeUpgradeFields(t *testing.T) {
	manager, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Save(State{
		State: StatusMigrating, Phase: StatusMigrating,
		AppVersion: "0.2.0-alpha.6", PocketBaseVersion: "0.41.0",
		ProblemCode: "backup_failed", Ready: false,
	}); err != nil {
		t.Fatal(err)
	}
	certificates, err := security.OpenServerCertificateManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server, err := StartStatusServer("127.0.0.1:0", manager, certificates)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	request, err := http.NewRequest(http.MethodGet, "https://"+server.listener.Addr().String()+"/api/subflow/v1/runtime/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := certificates.HealthClient(time.Second).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Data PublicStatus `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.State != StatusMigrating || envelope.Data.Ready || envelope.Data.ProblemCode != "" {
		t.Fatalf("unexpected public status: %+v", envelope.Data)
	}
	for _, privateValue := range []string{"0.2.0-alpha.6", "0.41.0", "backup_failed"} {
		if strings.Contains(string(body), privateValue) {
			t.Fatalf("public status exposed private upgrade metadata %q: %s", privateValue, body)
		}
	}
}

// The maintenance listener still owns the port for a moment after the state
// is saved as ready. A probe answered there must not report readiness, or a
// caller starts using an application that is not serving yet.
func TestMaintenanceListenerNeverReportsReadiness(t *testing.T) {
	manager, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Save(State{State: StatusReady, Phase: StatusReady, Ready: true}); err != nil {
		t.Fatal(err)
	}
	certificates, err := security.OpenServerCertificateManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server, err := StartStatusServer("127.0.0.1:0", manager, certificates)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	client := certificates.HealthClient(time.Second)
	base := "https://" + server.listener.Addr().String()

	for _, path := range []string{"/api/ready", "/api/health"} {
		response, err := client.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("%s on the maintenance listener returned %d, want 503", path, response.StatusCode)
		}
	}

	response, err := client.Get(base + "/api/subflow/v1/runtime/status")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var envelope struct {
		Data PublicStatus `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.Ready || envelope.Data.State != StatusVerifying || envelope.Data.Phase != StatusVerifying {
		t.Fatalf("maintenance listener reported %+v, want verifying and not ready", envelope.Data)
	}
}
