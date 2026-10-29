package security

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrInvalidServerCertificate = errors.New("invalid TLS certificate or private key")
	ErrServerCertificateAbsent  = errors.New("TLS certificate is unavailable")
	ErrServerCertificateAudit   = errors.New("TLS certificate update could not be committed")
)

const localCertificateName = "self-signed"

type ServerCertificateStatus struct {
	Source    string    `json:"source"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type loadedServerCertificate struct {
	pair       tls.Certificate
	leaf       *x509.Certificate
	serverName string
	source     string
	version    string
	certPath   string
}

// ServerCertificateManager manages the HTTPS listener's certificate. It
// keeps private keys under pb_data with owner-only permissions and swaps the
// active certificate atomically for new TLS handshakes.
type ServerCertificateManager struct {
	dir      string
	versions string
	mu       sync.Mutex
	active   atomic.Pointer[loadedServerCertificate]
	fallback *loadedServerCertificate
}

// OpenServerCertificateManager loads the active certificate and creates a
// local self-signed certificate when the installation has none.
func OpenServerCertificateManager(dataDir string) (*ServerCertificateManager, error) {
	return openServerCertificateManager(dataDir, true)
}

// LoadServerCertificateManager reads certificate state without creating any
// files. It is intended for health checks running beside the server process.
func LoadServerCertificateManager(dataDir string) (*ServerCertificateManager, error) {
	return openServerCertificateManager(dataDir, false)
}

func openServerCertificateManager(dataDir string, create bool) (*ServerCertificateManager, error) {
	if strings.TrimSpace(dataDir) == "" {
		return nil, errors.New("SubFlow data directory is required for HTTPS")
	}
	dir := filepath.Join(dataDir, ".subflow", "tls")
	versions := filepath.Join(dir, "versions")
	if create {
		if err := os.MkdirAll(versions, 0o700); err != nil {
			return nil, fmt.Errorf("create TLS data directory: %w", err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return nil, fmt.Errorf("secure TLS data directory: %w", err)
		}
		if err := os.Chmod(versions, 0o700); err != nil {
			return nil, fmt.Errorf("secure TLS version directory: %w", err)
		}
	} else if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrServerCertificateAbsent, err)
	}

	manager := &ServerCertificateManager{dir: dir, versions: versions}
	if create {
		if err := manager.ensureLocalCertificate(); err != nil {
			return nil, err
		}
	}
	fallback, err := manager.readVersion(localCertificateName, "self_signed")
	if err != nil {
		return nil, fmt.Errorf("load generated HTTPS certificate: %w", err)
	}
	manager.fallback = fallback
	activeName, err := os.ReadFile(filepath.Join(dir, "active"))
	if err != nil {
		return nil, fmt.Errorf("%w: active certificate selection is missing", ErrServerCertificateAbsent)
	}
	version := strings.TrimSpace(string(activeName))
	source := "uploaded"
	if version == localCertificateName {
		source = "self_signed"
	}
	active, err := manager.readVersion(version, source)
	if err != nil {
		return nil, fmt.Errorf("load active HTTPS certificate: %w", err)
	}
	manager.active.Store(active)
	return manager, nil
}

func (m *ServerCertificateManager) ensureLocalCertificate() error {
	localDir := filepath.Join(m.dir, localCertificateName)
	if _, err := os.Stat(localDir); err == nil {
		if _, loadErr := m.readVersion(localCertificateName, "self_signed"); loadErr != nil {
			return fmt.Errorf("load generated HTTPS certificate: %w", loadErr)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect generated HTTPS certificate: %w", err)
	} else {
		certPEM, keyPEM, generateErr := GenerateLocalServerCertificate()
		if generateErr != nil {
			return fmt.Errorf("generate local HTTPS certificate: %w", generateErr)
		}
		stage, stageErr := os.MkdirTemp(m.dir, ".self-signed-")
		if stageErr != nil {
			return fmt.Errorf("stage local HTTPS certificate: %w", stageErr)
		}
		defer os.RemoveAll(stage)
		if stageErr = writeServerPair(stage, certPEM, keyPEM); stageErr != nil {
			return stageErr
		}
		if stageErr = os.Rename(stage, localDir); stageErr != nil {
			return fmt.Errorf("install local HTTPS certificate: %w", stageErr)
		}
	}

	activePath := filepath.Join(m.dir, "active")
	if _, err := os.Stat(activePath); errors.Is(err, os.ErrNotExist) {
		if err = writeActiveServerCertificate(m.dir, localCertificateName); err != nil {
			return fmt.Errorf("select local HTTPS certificate: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("inspect active HTTPS certificate: %w", err)
	}
	return nil
}

// GenerateLocalServerCertificate returns a long-lived self-signed pair for
// localhost, loopback IPs and the default Docker Compose service name.
func GenerateLocalServerCertificate() ([]byte, []byte, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "SubFlow local HTTPS certificate"},
		NotBefore:    now.Add(-5 * time.Minute),
		NotAfter:     now.AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost", "subflow"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), nil
}

func writeServerPair(dir string, certPEM, keyPEM []byte) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create certificate version: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cert.pem"), certPEM, 0o644); err != nil {
		return fmt.Errorf("write public certificate: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "key.pem"), keyPEM, 0o600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}
	return nil
}

func validCertificateVersion(value string) bool {
	if !strings.HasPrefix(value, "cert-") || len(value) != len("cert-")+32 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "cert-"))
	return err == nil
}

func (m *ServerCertificateManager) readVersion(version, source string) (*loadedServerCertificate, error) {
	if version != localCertificateName && !validCertificateVersion(version) {
		return nil, ErrServerCertificateAbsent
	}
	dir := filepath.Join(m.dir, version)
	if version != localCertificateName {
		dir = filepath.Join(m.versions, version)
	}
	certPEM, err := os.ReadFile(filepath.Join(dir, "cert.pem"))
	if err != nil {
		return nil, err
	}
	keyPEM, err := os.ReadFile(filepath.Join(dir, "key.pem"))
	if err != nil {
		return nil, err
	}
	active, err := parseServerPair(certPEM, keyPEM, source)
	if err != nil {
		return nil, err
	}
	active.version = version
	active.certPath = filepath.Join(dir, "cert.pem")
	return active, nil
}

func parseServerPair(certPEM, keyPEM []byte, source string) (*loadedServerCertificate, error) {
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil || len(pair.Certificate) == 0 {
		return nil, ErrInvalidServerCertificate
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return nil, ErrInvalidServerCertificate
	}
	now := time.Now()
	if leaf.NotBefore.After(now.Add(5*time.Minute)) || !leaf.NotAfter.After(now) || (len(leaf.DNSNames) == 0 && len(leaf.IPAddresses) == 0) {
		return nil, ErrInvalidServerCertificate
	}
	if len(leaf.ExtKeyUsage) > 0 {
		validForServer := false
		for _, usage := range leaf.ExtKeyUsage {
			if usage == x509.ExtKeyUsageServerAuth || usage == x509.ExtKeyUsageAny {
				validForServer = true
				break
			}
		}
		if !validForServer {
			return nil, ErrInvalidServerCertificate
		}
	}
	pair.Leaf = leaf
	serverName := ""
	if len(leaf.DNSNames) > 0 {
		serverName = leaf.DNSNames[0]
	} else if len(leaf.IPAddresses) > 0 {
		serverName = leaf.IPAddresses[0].String()
	}
	return &loadedServerCertificate{pair: pair, leaf: leaf, serverName: serverName, source: source}, nil
}

func writeActiveServerCertificate(dir, version string) error {
	file, err := os.CreateTemp(dir, ".active-")
	if err != nil {
		return err
	}
	tmpPath := file.Name()
	defer os.Remove(tmpPath)
	if err = file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err = file.WriteString(version + "\n"); err != nil {
		_ = file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, filepath.Join(dir, "active"))
}

// Update stages an uploaded certificate pair, selects it for new handshakes,
// and commits the corresponding audit event. A failed audit restores the
// prior certificate and removes the staged private key.
func (m *ServerCertificateManager) Update(certPEM, keyPEM []byte, commit func() error) error {
	candidate, err := parseServerPair(certPEM, keyPEM, "uploaded")
	if err != nil {
		return err
	}
	if commit == nil {
		return errors.New("TLS certificate update requires an audit commit")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	versionBytes := make([]byte, 16)
	if _, err = rand.Read(versionBytes); err != nil {
		return fmt.Errorf("create TLS certificate version: %w", err)
	}
	version := "cert-" + hex.EncodeToString(versionBytes)
	stage := filepath.Join(m.versions, version)
	if err = os.Mkdir(stage, 0o700); err != nil {
		return fmt.Errorf("stage uploaded TLS certificate: %w", err)
	}
	keepStage := false
	defer func() {
		if !keepStage {
			_ = os.RemoveAll(stage)
		}
	}()
	if err = writeServerPair(stage, certPEM, keyPEM); err != nil {
		return err
	}
	candidate.version = version
	candidate.certPath = filepath.Join(stage, "cert.pem")
	previous := m.active.Load()
	previousVersion := previous.version
	if err = writeActiveServerCertificate(m.dir, version); err != nil {
		return fmt.Errorf("activate uploaded TLS certificate: %w", err)
	}
	m.active.Store(candidate)
	if err = commit(); err != nil {
		if rollbackErr := m.restore(previous); rollbackErr != nil {
			return errors.Join(ErrServerCertificateAudit, err, fmt.Errorf("TLS certificate rollback failed: %w", rollbackErr))
		}
		return errors.Join(ErrServerCertificateAudit, err)
	}
	keepStage = true
	m.removeOldVersion(previousVersion, version)
	return nil
}

// Reset switches back to the generated self-signed certificate and removes
// the uploaded private key after the audit event is committed.
func (m *ServerCertificateManager) Reset(commit func() error) error {
	if commit == nil {
		return errors.New("TLS certificate reset requires an audit commit")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	previous := m.active.Load()
	if previous.version == localCertificateName {
		if err := commit(); err != nil {
			return errors.Join(ErrServerCertificateAudit, err)
		}
		return nil
	}
	if err := writeActiveServerCertificate(m.dir, localCertificateName); err != nil {
		return fmt.Errorf("activate generated TLS certificate: %w", err)
	}
	m.active.Store(m.fallback)
	if err := commit(); err != nil {
		if rollbackErr := m.restore(previous); rollbackErr != nil {
			return errors.Join(ErrServerCertificateAudit, err, fmt.Errorf("TLS certificate rollback failed: %w", rollbackErr))
		}
		return errors.Join(ErrServerCertificateAudit, err)
	}
	m.removeOldVersion(previous.version, localCertificateName)
	return nil
}

func (m *ServerCertificateManager) restore(certificate *loadedServerCertificate) error {
	if certificate == nil {
		return ErrServerCertificateAbsent
	}
	if err := writeActiveServerCertificate(m.dir, certificate.version); err != nil {
		return err
	}
	m.active.Store(certificate)
	return nil
}

func (m *ServerCertificateManager) removeOldVersion(oldVersion, newVersion string) {
	if oldVersion != localCertificateName && oldVersion != newVersion && validCertificateVersion(oldVersion) {
		_ = os.RemoveAll(filepath.Join(m.versions, oldVersion))
	}
}

func (m *ServerCertificateManager) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	active := m.active.Load()
	if active == nil {
		return nil, ErrServerCertificateAbsent
	}
	return &active.pair, nil
}

func (m *ServerCertificateManager) TLSConfig() *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12, GetCertificate: m.GetCertificate}
}

func (m *ServerCertificateManager) Status() ServerCertificateStatus {
	active := m.active.Load()
	if active == nil {
		return ServerCertificateStatus{}
	}
	return ServerCertificateStatus{Source: active.source, ExpiresAt: active.leaf.NotAfter.UTC()}
}

// CertificatePath exposes only the active public certificate for proxy
// trust configuration. It never returns a private-key path.
func (m *ServerCertificateManager) CertificatePath() string {
	active := m.active.Load()
	if active == nil {
		return ""
	}
	return active.certPath
}

// HealthClient verifies the TLS peer against the active certificate and
// checks its SAN while connecting to the local HTTPS listener.
func (m *ServerCertificateManager) HealthClient(timeout time.Duration) *http.Client {
	active := m.active.Load()
	roots := x509.NewCertPool()
	serverName := ""
	if active != nil {
		for _, der := range active.pair.Certificate {
			if certificate, err := x509.ParseCertificate(der); err == nil {
				roots.AddCert(certificate)
			}
		}
		serverName = active.serverName
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    roots,
			ServerName: serverName,
		}},
	}
}
