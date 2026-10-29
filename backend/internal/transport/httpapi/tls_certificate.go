package httpapi

import (
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	"ganigeorgiev.com/pocketbase/pocketbase/core"

	"subflow/internal/security"
)

const maxTLSCertificateFileSize = 256 << 10

var errInvalidTLSCertificateUpload = errors.New("invalid TLS certificate upload")

func (a *API) getTLSCertificate(e *core.RequestEvent) error {
	if a.Certificates == nil {
		return tlsCertificateError(e, http.StatusInternalServerError, "tls_certificate_unavailable", "TLS certificate management is unavailable")
	}
	if err := a.Service.RequireSystemPermission(e.Request.Context(), authID(e), "system.settings.manage"); err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, a.Certificates.Status(), nil)
}

func (a *API) updateTLSCertificate(e *core.RequestEvent) error {
	const action = "system.tls_certificate.updated"
	authorized, err := a.authorizeTLSCertificateMutation(e, action)
	if err != nil || !authorized {
		return err
	}
	if a.Certificates == nil {
		return a.failTLSCertificateMutation(e, action, "certificate_manager_unavailable", http.StatusInternalServerError, "tls_certificate_unavailable", "TLS certificate management is unavailable")
	}
	certificatePEM, keyPEM, err := readTLSCertificateUpload(e)
	if err != nil {
		return a.failTLSCertificateMutation(e, action, "invalid_upload", http.StatusBadRequest, "invalid_tls_certificate", "Upload a valid PEM certificate and matching private key")
	}
	commit := func() error {
		return a.Service.RecordAudit(e.Request.Context(), authID(e), "", action, "system_settings", "tls_certificate", "success", "certificate_changed=true private_key_changed=true")
	}
	err = a.Certificates.Update(certificatePEM, keyPEM, commit)
	if err != nil {
		if errors.Is(err, security.ErrInvalidServerCertificate) {
			return a.failTLSCertificateMutation(e, action, "invalid_certificate", http.StatusBadRequest, "invalid_tls_certificate", "Upload a valid PEM certificate and matching private key")
		}
		if errors.Is(err, security.ErrServerCertificateAudit) {
			return a.failTLSCertificateMutation(e, action, "audit_unavailable", http.StatusInternalServerError, "tls_certificate_audit_failed", "Certificate was not changed because its audit record could not be saved")
		}
		return a.failTLSCertificateMutation(e, action, "certificate_storage_failed", http.StatusInternalServerError, "tls_certificate_update_failed", "The TLS certificate could not be saved")
	}
	return ok(e, http.StatusOK, a.Certificates.Status(), nil)
}

func (a *API) resetTLSCertificate(e *core.RequestEvent) error {
	const action = "system.tls_certificate.reset"
	authorized, err := a.authorizeTLSCertificateMutation(e, action)
	if err != nil || !authorized {
		return err
	}
	if a.Certificates == nil {
		return a.failTLSCertificateMutation(e, action, "certificate_manager_unavailable", http.StatusInternalServerError, "tls_certificate_unavailable", "TLS certificate management is unavailable")
	}
	commit := func() error {
		return a.Service.RecordAudit(e.Request.Context(), authID(e), "", action, "system_settings", "tls_certificate", "success", "source=self_signed")
	}
	if err := a.Certificates.Reset(commit); err != nil {
		if errors.Is(err, security.ErrServerCertificateAudit) {
			return a.failTLSCertificateMutation(e, action, "audit_unavailable", http.StatusInternalServerError, "tls_certificate_audit_failed", "Certificate was not changed because its audit record could not be saved")
		}
		return a.failTLSCertificateMutation(e, action, "certificate_storage_failed", http.StatusInternalServerError, "tls_certificate_update_failed", "The TLS certificate could not be saved")
	}
	return ok(e, http.StatusOK, a.Certificates.Status(), nil)
}

func (a *API) authorizeTLSCertificateMutation(e *core.RequestEvent, action string) (bool, error) {
	if err := a.Service.RequireSystemPermission(e.Request.Context(), authID(e), "system.settings.manage"); err != nil {
		if auditErr := a.recordTLSCertificateAudit(e, action, "failure", "reason=forbidden"); auditErr != nil {
			return false, tlsCertificateError(e, http.StatusInternalServerError, "audit_log_unavailable", "Audit logging is unavailable")
		}
		return false, fail(e, err)
	}
	return true, nil
}

func (a *API) failTLSCertificateMutation(e *core.RequestEvent, action, reason string, status int, code, message string) error {
	if err := a.recordTLSCertificateAudit(e, action, "failure", "reason="+reason); err != nil {
		return tlsCertificateError(e, http.StatusInternalServerError, "audit_log_unavailable", "Audit logging is unavailable")
	}
	return tlsCertificateError(e, status, code, message)
}

func tlsCertificateError(e *core.RequestEvent, status int, code, message string) error {
	requestID, _ := e.Request.Context().Value(requestIDContextKey{}).(string)
	return e.JSON(status, errorEnvelope{Error: apiError{Code: code, Message: message, RequestID: requestID}})
}

func (a *API) recordTLSCertificateAudit(e *core.RequestEvent, action, outcome, summary string) error {
	return a.Service.RecordAudit(e.Request.Context(), authID(e), "", action, "system_settings", "tls_certificate", outcome, summary)
}

func readTLSCertificateUpload(e *core.RequestEvent) ([]byte, []byte, error) {
	mediaType, _, err := mime.ParseMediaType(e.Request.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		return nil, nil, errInvalidTLSCertificateUpload
	}
	e.Request.Body = http.MaxBytesReader(e.Response, e.Request.Body, 2*maxTLSCertificateFileSize+32*1024)
	if err = e.Request.ParseMultipartForm(2 * maxTLSCertificateFileSize); err != nil {
		return nil, nil, errInvalidTLSCertificateUpload
	}
	form := e.Request.MultipartForm
	if form == nil {
		return nil, nil, errInvalidTLSCertificateUpload
	}
	var certificatePEM, keyPEM []byte
	parseErr := func() error {
		certificateFiles := form.File["certificate"]
		keyFiles := form.File["privateKey"]
		if len(certificateFiles) != 1 || len(keyFiles) != 1 {
			return errInvalidTLSCertificateUpload
		}
		var partErr error
		certificatePEM, partErr = readTLSPart(certificateFiles[0])
		if partErr != nil {
			return errInvalidTLSCertificateUpload
		}
		keyPEM, partErr = readTLSPart(keyFiles[0])
		if partErr != nil {
			return errInvalidTLSCertificateUpload
		}
		if !strings.Contains(string(certificatePEM), "-----BEGIN CERTIFICATE-----") {
			return errInvalidTLSCertificateUpload
		}
		return nil
	}()
	cleanupErr := form.RemoveAll()
	if cleanupErr != nil {
		return nil, nil, errors.Join(errInvalidTLSCertificateUpload, cleanupErr)
	}
	if parseErr != nil {
		return nil, nil, parseErr
	}
	return certificatePEM, keyPEM, nil
}

func readTLSPart(header *multipart.FileHeader) ([]byte, error) {
	if header == nil || header.Size < 1 || header.Size > maxTLSCertificateFileSize {
		return nil, errInvalidTLSCertificateUpload
	}
	file, err := header.Open()
	if err != nil {
		return nil, errInvalidTLSCertificateUpload
	}
	defer file.Close()
	value, err := io.ReadAll(io.LimitReader(file, maxTLSCertificateFileSize+1))
	if err != nil || len(value) > maxTLSCertificateFileSize {
		return nil, errInvalidTLSCertificateUpload
	}
	return value, nil
}
