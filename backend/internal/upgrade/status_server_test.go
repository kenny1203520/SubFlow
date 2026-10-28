package upgrade

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
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
	server, err := StartStatusServer("127.0.0.1:0", manager)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	response, err := http.Get("http://" + server.listener.Addr().String() + "/api/subflow/v1/runtime/status")
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
