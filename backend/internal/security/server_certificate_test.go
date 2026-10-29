package security

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestOpenServerCertificateManagerCreatesPrivateSelfSignedCertificate(t *testing.T) {
	dataDir := t.TempDir()
	manager, err := OpenServerCertificateManager(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if status := manager.Status(); status.Source != "self_signed" || !status.ExpiresAt.After(time.Now()) {
		t.Fatalf("unexpected default certificate status: %+v", status)
	}
	if _, err := LoadServerCertificateManager(dataDir); err != nil {
		t.Fatalf("existing certificate state could not be loaded: %v", err)
	}
	if runtime.GOOS != "windows" {
		keyPath := filepath.Join(dataDir, ".subflow", "tls", "self-signed", "key.pem")
		info, err := os.Stat(keyPath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("private key permissions = %04o, want 0600", info.Mode().Perm())
		}
	}
}

func TestServerCertificateHealthClientVerifiesLocalTLS(t *testing.T) {
	manager, err := OpenServerCertificateManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	server.TLS = manager.TLSConfig()
	server.StartTLS()
	defer server.Close()
	response, err := manager.HealthClient(time.Second).Get(server.URL)
	if err != nil {
		t.Fatalf("certificate-trusting HTTPS request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("health response status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
}

func TestServerCertificateUpdatePersistsAndResetRestoresDefault(t *testing.T) {
	dataDir := t.TempDir()
	manager, err := OpenServerCertificateManager(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	certPEM, keyPEM, err := GenerateLocalServerCertificate()
	if err != nil {
		t.Fatal(err)
	}
	committed := false
	if err := manager.Update(certPEM, keyPEM, func() error { committed = true; return nil }); err != nil {
		t.Fatal(err)
	}
	if !committed || manager.Status().Source != "uploaded" {
		t.Fatalf("certificate update was not activated and committed: committed=%v status=%+v", committed, manager.Status())
	}
	loaded, err := LoadServerCertificateManager(dataDir)
	if err != nil || loaded.Status().Source != "uploaded" {
		t.Fatalf("uploaded certificate did not survive restart: status=%+v error=%v", loaded.Status(), err)
	}
	if err := loaded.Reset(func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if loaded.Status().Source != "self_signed" {
		t.Fatalf("reset source = %q, want self_signed", loaded.Status().Source)
	}
	reopened, err := LoadServerCertificateManager(dataDir)
	if err != nil || reopened.Status().Source != "self_signed" {
		t.Fatalf("self-signed certificate did not survive reset: status=%+v error=%v", reopened.Status(), err)
	}
}

func TestServerCertificateUpdateRollsBackWhenAuditCommitFails(t *testing.T) {
	dataDir := t.TempDir()
	manager, err := OpenServerCertificateManager(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	certPEM, keyPEM, err := GenerateLocalServerCertificate()
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Update(certPEM, keyPEM, func() error { return errors.New("audit unavailable") }); !errors.Is(err, ErrServerCertificateAudit) {
		t.Fatalf("Update() error = %v, want audit failure", err)
	}
	if status := manager.Status(); status.Source != "self_signed" {
		t.Fatalf("failed audit left active certificate source %q", status.Source)
	}
	reopened, err := LoadServerCertificateManager(dataDir)
	if err != nil || reopened.Status().Source != "self_signed" {
		t.Fatalf("failed audit left persisted certificate active: status=%+v error=%v", reopened.Status(), err)
	}
	entries, err := os.ReadDir(filepath.Join(dataDir, ".subflow", "tls", "versions"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed audit retained uploaded private key material: entries=%d error=%v", len(entries), err)
	}
}

func TestServerCertificateRejectsMismatchedKey(t *testing.T) {
	manager, err := OpenServerCertificateManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	certPEM, _, err := GenerateLocalServerCertificate()
	if err != nil {
		t.Fatal(err)
	}
	_, wrongKey, err := GenerateLocalServerCertificate()
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Update(certPEM, wrongKey, func() error { return nil }); !errors.Is(err, ErrInvalidServerCertificate) {
		t.Fatalf("Update() error = %v, want invalid certificate", err)
	}
	if manager.Status().Source != "self_signed" {
		t.Fatal("invalid certificate changed the active source")
	}
}
