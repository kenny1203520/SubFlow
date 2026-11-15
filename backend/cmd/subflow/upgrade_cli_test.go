package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"subflow/internal/upgrade"
)

func TestUpgradeCommandProcess(t *testing.T) {
	if len(os.Args) > 2 && os.Args[2] == "--" {
		os.Args = append([]string{os.Args[0]}, os.Args[3:]...)
		main()
		os.Exit(0)
	}
	dir := t.TempDir()
	run := func(args ...string) ([]byte, error) {
		args = append([]string{"-test.run=^TestUpgradeCommandProcess$", "--", "upgrade"}, args...)
		cmd := exec.Command(os.Args[0], append(args, "--dir="+dir)...)
		cmd.Env = append(os.Environ(), "SUBFLOW_SETTINGS_ENCRYPTION_KEY=diagnostics-test-key-1234567890")
		return cmd.CombinedOutput()
	}
	if out, err := run("retry"); err != nil || !strings.Contains(string(out), "upgrade retry completed") {
		t.Fatalf("retry should report success: %s (%v)", out, err)
	}
	if out, err := run("status", "--state-only"); err != nil || !strings.Contains(string(out), `"ready": true`) || strings.Contains(string(out), `"backups"`) {
		t.Fatalf("state-only should return the saved state: %s (%v)", out, err)
	}
	manager, err := upgrade.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Save(upgrade.State{AppVersion: "99.0.0", State: upgrade.StatusReady}); err != nil {
		t.Fatal(err)
	}
	if out, err := run("retry"); err == nil || !strings.Contains(string(out), "downgrades are not supported") || strings.Contains(string(out), "retry completed") {
		t.Fatalf("retry should exit with a downgrade error: %s (%v)", out, err)
	}
}
