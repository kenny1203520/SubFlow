package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadEnvFileLoadsUnsetVariablesAndPreservesProcessValues(t *testing.T) {
	t.Setenv("SUBFLOW_ENV_TEST_OVERRIDE", "process-value")
	t.Setenv("SUBFLOW_ENV_TEST_FILE", "temporary")
	if err := os.Unsetenv("SUBFLOW_ENV_TEST_FILE"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("# comment\nSUBFLOW_ENV_TEST_OVERRIDE=file-value\nexport SUBFLOW_ENV_TEST_FILE=\"from-file\" # note\n"), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"subflow", "serve", "--env-file", path, "--http", "127.0.0.1:8080"}
	clean, err := LoadEnvFile(args)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(clean, []string{"subflow", "serve", "--http", "127.0.0.1:8080"}) {
		t.Fatalf("remaining args = %#v", clean)
	}
	if got := os.Getenv("SUBFLOW_ENV_TEST_OVERRIDE"); got != "process-value" {
		t.Fatalf("process value overwritten: %q", got)
	}
	if got := os.Getenv("SUBFLOW_ENV_TEST_FILE"); got != "from-file" {
		t.Fatalf("file value not loaded: %q", got)
	}
}

func TestLoadEnvFileSupportsEqualsFormAndRejectsBadEntry(t *testing.T) {
	t.Setenv("SUBFLOW_ENV_TEST_EQUALS", "temporary")
	if err := os.Unsetenv("SUBFLOW_ENV_TEST_EQUALS"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("SUBFLOW_ENV_TEST_EQUALS=value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadEnvFile([]string{"subflow", "serve", "--env-file=" + path}); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("SUBFLOW_ENV_TEST_EQUALS"); got != "value" {
		t.Fatalf("got %q", got)
	}
	if _, err := LoadEnvFile([]string{"subflow", "serve", "--env-file"}); err == nil {
		t.Fatal("expected missing path error")
	}
}
