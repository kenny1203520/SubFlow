package httpapi_test

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"ganigeorgiev.com/pocketbase/pocketbase/tests"

	"subflow/internal/domain"
	"subflow/internal/ports"
	"subflow/internal/security"
)

type tlsCertificateAuditUnavailable struct{ ports.AuditRepository }

func (r tlsCertificateAuditUnavailable) Create(context.Context, *domain.AuditLog) error {
	return errors.New("audit unavailable")
}

func TestTLSCertificateUploadAndResetAreAudited(t *testing.T) {
	f := newAPITestApp(t)
	f.enableCertificateManagement(t)
	admin := f.createUser(t, "tls-admin@example.test")
	f.assignSystemAdmin(t, admin)
	certificatePEM, keyPEM, err := security.GenerateLocalServerCertificate()
	if err != nil {
		t.Fatal(err)
	}
	body, contentType := tlsCertificateUploadBody(t, certificatePEM, keyPEM)
	scenario := tests.ApiScenario{
		Name:   "administrator can upload a TLS certificate and private key",
		Method: http.MethodPost, URL: "/api/subflow/v1/system/tls-certificate", Body: body,
		Headers:        map[string]string{"Authorization": f.token(t, admin), "Content-Type": contentType},
		TestAppFactory: f.factory(), BeforeTestFunc: f.beforeTest(), DisableTestAppCleanup: true,
		ExpectedStatus: http.StatusOK, ExpectedContent: []string{`"source":"uploaded"`},
	}
	scenario.Test(t)
	if f.certificates.Status().Source != "uploaded" {
		t.Fatalf("active certificate source = %q, want uploaded", f.certificates.Status().Source)
	}
	assertTLSAudit(t, f, "system.tls_certificate.updated", "success", "certificate_changed=true")

	reset := tests.ApiScenario{
		Name:   "administrator can restore the generated local certificate",
		Method: http.MethodDelete, URL: "/api/subflow/v1/system/tls-certificate",
		Headers:        map[string]string{"Authorization": f.token(t, admin)},
		TestAppFactory: f.factory(), BeforeTestFunc: f.beforeTest(), DisableTestAppCleanup: true,
		ExpectedStatus: http.StatusOK, ExpectedContent: []string{`"source":"self_signed"`},
	}
	reset.Test(t)
	assertTLSAudit(t, f, "system.tls_certificate.reset", "success", "source=self_signed")
}

func TestTLSCertificateDenialAndInvalidUploadAreAuditedWithoutSecrets(t *testing.T) {
	f := newAPITestApp(t)
	f.enableCertificateManagement(t)
	user := f.createUser(t, "tls-user@example.test")
	certificatePEM, keyPEM, err := security.GenerateLocalServerCertificate()
	if err != nil {
		t.Fatal(err)
	}
	body, contentType := tlsCertificateUploadBody(t, certificatePEM, keyPEM)
	denied := tests.ApiScenario{
		Name:   "regular user cannot change the server certificate",
		Method: http.MethodPost, URL: "/api/subflow/v1/system/tls-certificate", Body: body,
		Headers:        map[string]string{"Authorization": f.token(t, user), "Content-Type": contentType},
		TestAppFactory: f.factory(), BeforeTestFunc: f.beforeTest(), DisableTestAppCleanup: true,
		ExpectedStatus: http.StatusForbidden, ExpectedContent: []string{`"code":"forbidden"`},
	}
	denied.Test(t)
	assertTLSAudit(t, f, "system.tls_certificate.updated", "failure", "reason=forbidden")
	if f.certificates.Status().Source != "self_signed" {
		t.Fatal("unauthorized request changed the active certificate")
	}

	admin := f.createUser(t, "tls-admin-invalid@example.test")
	f.assignSystemAdmin(t, admin)
	badBody, badType := tlsCertificateUploadBody(t, certificatePEM, []byte("not a private key"))
	invalid := tests.ApiScenario{
		Name:   "administrator cannot upload a certificate with an invalid key",
		Method: http.MethodPost, URL: "/api/subflow/v1/system/tls-certificate", Body: badBody,
		Headers:        map[string]string{"Authorization": f.token(t, admin), "Content-Type": badType},
		TestAppFactory: f.factory(), BeforeTestFunc: f.beforeTest(), DisableTestAppCleanup: true,
		ExpectedStatus: http.StatusBadRequest, ExpectedContent: []string{`"code":"invalid_tls_certificate"`},
	}
	invalid.Test(t)
	assertTLSAudit(t, f, "system.tls_certificate.updated", "failure", "reason=invalid_certificate")
	if f.certificates.Status().Source != "self_signed" {
		t.Fatal("invalid upload changed the active certificate")
	}

	logs, err := f.stores.Audits.List(context.Background(), "", ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 25}, Action: "system.tls_certificate.updated"})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range logs.Items {
		if strings.Contains(entry.Summary, string(keyPEM)) || strings.Contains(entry.Summary, string(certificatePEM)) {
			t.Fatal("TLS audit summary included certificate or private-key contents")
		}
	}
}

func TestTLSCertificateAuditFailureRollsBackAndReturnsExplicitError(t *testing.T) {
	f := newAPITestApp(t)
	f.enableCertificateManagement(t)
	admin := f.createUser(t, "tls-admin-audit-failure@example.test")
	f.assignSystemAdmin(t, admin)
	certificatePEM, keyPEM, err := security.GenerateLocalServerCertificate()
	if err != nil {
		t.Fatal(err)
	}
	body, contentType := tlsCertificateUploadBody(t, certificatePEM, keyPEM)
	f.service.Stores.Audits = attachmentAuditFailure{f.stores.Audits}
	scenario := tests.ApiScenario{
		Name:   "certificate upload is rolled back when its audit cannot be saved",
		Method: http.MethodPost, URL: "/api/subflow/v1/system/tls-certificate", Body: body,
		Headers:        map[string]string{"Authorization": f.token(t, admin), "Content-Type": contentType},
		TestAppFactory: f.factory(), BeforeTestFunc: f.beforeTest(), DisableTestAppCleanup: true,
		ExpectedStatus: http.StatusInternalServerError, ExpectedContent: []string{`"code":"tls_certificate_audit_failed"`},
	}
	scenario.Test(t)
	if f.certificates.Status().Source != "self_signed" {
		t.Fatal("audit failure did not restore the in-memory certificate")
	}
	assertTLSAudit(t, f, "system.tls_certificate.updated", "failure", "reason=audit_unavailable")
	loaded, err := security.LoadServerCertificateManager(f.certificateDataDir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status().Source != "self_signed" {
		t.Fatal("audit failure did not restore the persisted certificate")
	}
}

func TestTLSCertificateReturnsAuditUnavailableWhenFailureEventCannotBeRecorded(t *testing.T) {
	f := newAPITestApp(t)
	f.enableCertificateManagement(t)
	admin := f.createUser(t, "tls-admin-audit-offline@example.test")
	f.assignSystemAdmin(t, admin)
	certificatePEM, keyPEM, err := security.GenerateLocalServerCertificate()
	if err != nil {
		t.Fatal(err)
	}
	body, contentType := tlsCertificateUploadBody(t, certificatePEM, keyPEM)
	f.service.Stores.Audits = tlsCertificateAuditUnavailable{f.stores.Audits}
	scenario := tests.ApiScenario{
		Name:   "certificate upload reports when both success and failure audits are unavailable",
		Method: http.MethodPost, URL: "/api/subflow/v1/system/tls-certificate", Body: body,
		Headers:        map[string]string{"Authorization": f.token(t, admin), "Content-Type": contentType},
		TestAppFactory: f.factory(), BeforeTestFunc: f.beforeTest(), DisableTestAppCleanup: true,
		ExpectedStatus: http.StatusInternalServerError, ExpectedContent: []string{`"code":"audit_log_unavailable"`},
	}
	scenario.Test(t)
	if f.certificates.Status().Source != "self_signed" {
		t.Fatal("audit outage changed the active certificate")
	}
	loaded, err := security.LoadServerCertificateManager(f.certificateDataDir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status().Source != "self_signed" {
		t.Fatal("audit outage changed the persisted certificate")
	}
}

func tlsCertificateUploadBody(t *testing.T, certificatePEM, keyPEM []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	certificatePart, err := writer.CreateFormFile("certificate", "certificate.pem")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = certificatePart.Write(certificatePEM); err != nil {
		t.Fatal(err)
	}
	keyPart, err := writer.CreateFormFile("privateKey", "private-key.pem")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = keyPart.Write(keyPEM); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body, writer.FormDataContentType()
}

func assertTLSAudit(t *testing.T, f *apiFixture, action, outcome, summary string) {
	t.Helper()
	logs, err := f.stores.Audits.List(context.Background(), "", ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 25}, Action: action})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range logs.Items {
		if entry.Outcome == outcome && strings.Contains(entry.Summary, summary) {
			return
		}
	}
	if len(logs.Items) == 0 {
		t.Fatalf("missing TLS certificate audit outcome=%q summary=%q: %#v", outcome, summary, logs.Items)
	}
	t.Fatalf("missing TLS certificate audit outcome=%q summary=%q: %#v", outcome, summary, logs.Items)
}
