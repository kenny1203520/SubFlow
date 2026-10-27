package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"ganigeorgiev.com/pocketbase/pocketbase/apis"
	"ganigeorgiev.com/pocketbase/pocketbase/core"

	"subflow/internal/application"
	"subflow/internal/domain"
	"subflow/internal/ports"
	"subflow/internal/upgrade"
)

type shareAttempt struct {
	Count   int
	ResetAt time.Time
}
type API struct {
	Service        *application.Service
	Upgrades       *upgrade.Manager
	shareAttemptMu sync.Mutex
	shareAttempts  map[string]shareAttempt
}

const ClientProtocolVersion = 1

type envelope struct {
	Data any `json:"data"`
	Meta any `json:"meta,omitempty"`
}
type errorEnvelope struct {
	Error apiError `json:"error"`
}
type apiError struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	Details   string            `json:"details,omitempty"`
	RequestID string            `json:"requestId"`
}

type requestIDContextKey struct{}

func (a *API) Register(router *http.ServeMux) {}

func (a *API) RegisterRoutes(e *core.ServeEvent) {
	// Applies to every route registered on e.Router (RouterGroup middlewares
	// are resolved at dispatch time, not baked in per-route at registration),
	// so unauthenticated routes like setup/register are covered too — both
	// call s.audit and need the caller's IP/User-Agent same as anything else.
	e.Router.BindFunc(func(re *core.RequestEvent) error {
		requestID := make([]byte, 12)
		if _, err := rand.Read(requestID); err != nil {
			return err
		}
		id := hex.EncodeToString(requestID)
		re.Response.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(re.Request.Context(), requestIDContextKey{}, id)
		re.Request = re.Request.WithContext(application.WithAuditRequestMeta(ctx, application.AuditRequestMeta{IP: re.RealIP(), UserAgent: re.Request.UserAgent()}))
		if strings.HasPrefix(re.Request.URL.Path, "/api/subflow/v1/") && isBusinessWrite(re.Request.Method) {
			protocol := re.Request.Header.Get("X-SubFlow-Protocol")
			if protocol == "" {
				protocol = "1"
			}
			version, parseErr := strconv.Atoi(protocol)
			if parseErr != nil || version != ClientProtocolVersion {
				re.Response.Header().Set("X-SubFlow-Protocol", strconv.Itoa(ClientProtocolVersion))
				re.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
				re.Response.WriteHeader(http.StatusUpgradeRequired)
				return json.NewEncoder(re.Response).Encode(errorEnvelope{Error: apiError{Code: "client_update_required", Message: "Update SubFlow before writing data", RequestID: id}})
			}
		}
		return re.Next()
	})
	e.Router.GET("/api/subflow/v1/runtime/status", a.runtimeStatus)
	e.Router.GET("/api/ready", a.runtimeReady)
	e.Router.GET("/api/live", a.runtimeLive)
	bind := apis.RequireAuth("users")
	e.Router.GET("/api/subflow/v1/setup/status", a.setupStatus)
	e.Router.GET("/api/subflow/v1/auth/captcha/challenge", a.captchaChallenge)
	e.Router.POST("/api/subflow/v1/setup/initialize", a.initializeSetup)
	e.Router.POST("/api/subflow/v1/auth/register", a.register)
	e.Router.POST("/api/subflow/v1/auth/logout", a.logout)
	e.Router.GET("/api/subflow/v1/shares/{token}", a.publicShare)
	e.Router.POST("/api/subflow/v1/shares/{token}/access", a.sharePasswordAccess)
	e.Router.GET("/api/subflow/v1/shares/{token}/account", a.accountShare).Bind(bind)
	e.Router.GET("/api/subflow/v1/shares/{token}/attachments/{reference}", a.shareAttachmentList)
	e.Router.GET("/api/subflow/v1/shares/{token}/attachment/{reference}", a.shareAttachmentContent)
	e.Router.GET("/api/subflow/v1/shares/{token}/account/attachments/{reference}", a.accountShareAttachmentList).Bind(bind)
	e.Router.GET("/api/subflow/v1/shares/{token}/account/attachment/{reference}", a.accountShareAttachmentContent).Bind(bind)
	e.Router.GET("/api/subflow/v1/auth/external-auths", a.listExternalAuths).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/auth/external-auths/{provider}", a.unlinkExternalAuth).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups", a.listGroups).Bind(bind)
	e.Router.GET("/api/subflow/v1/currencies", a.currencies).Bind(bind)
	e.Router.GET("/api/subflow/v1/categories", a.listCategories).Bind(bind)
	e.Router.GET("/api/subflow/v1/system/access", a.systemAccess).Bind(bind)
	e.Router.POST("/api/subflow/v1/categories", a.createCategory).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/categories/{id}", a.updateCategory).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/categories/{id}", a.archiveCategory).Bind(bind)
	e.Router.GET("/api/subflow/v1/exchange-rates/quote", a.exchangeRateQuote).Bind(bind)
	e.Router.GET("/api/subflow/v1/dashboard", a.workspaceDashboard).Bind(bind)
	e.Router.GET("/api/subflow/v1/subscriptions", a.listPersonalSubscriptions).Bind(bind)
	e.Router.POST("/api/subflow/v1/subscriptions", a.createPersonalSubscription).Bind(bind)
	e.Router.GET("/api/subflow/v1/expenses", a.listPersonalExpenses).Bind(bind)
	e.Router.POST("/api/subflow/v1/expenses", a.createPersonalExpense).Bind(bind)
	e.Router.GET("/api/subflow/v1/incomes", a.listPersonalIncomes).Bind(bind)
	e.Router.POST("/api/subflow/v1/incomes", a.createPersonalIncome).Bind(bind)
	e.Router.GET("/api/subflow/v1/recurring-incomes", a.listRecurringIncomes).Bind(bind)
	e.Router.POST("/api/subflow/v1/recurring-incomes", a.createRecurringIncome).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/recurring-incomes/{id}", a.updateRecurringIncome).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/recurring-incomes/{id}/status", a.changeRecurringIncomeStatus).Bind(bind)
	e.Router.GET("/api/subflow/v1/recurring-incomes/{id}/occurrences", a.listIncomeOccurrences).Bind(bind)
	e.Router.POST("/api/subflow/v1/recurring-incomes/{id}/occurrences/confirm", a.confirmIncomeOccurrence).Bind(bind)
	e.Router.POST("/api/subflow/v1/recurring-incomes/{id}/occurrences/skip", a.skipIncomeOccurrence).Bind(bind)
	e.Router.POST("/api/subflow/v1/recurring-incomes/{id}/occurrences/retry", a.retryIncomeOccurrence).Bind(bind)
	e.Router.GET("/api/subflow/v1/recurring-incomes/{id}/backfill/preview", a.previewRecurringIncomeBackfill).Bind(bind)
	e.Router.POST("/api/subflow/v1/recurring-incomes/{id}/backfill", a.backfillRecurringIncome).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/incomes/{id}", a.updateGroupOrPersonalIncome).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/incomes/{id}", a.deleteGroupOrPersonalIncome).Bind(bind)
	e.Router.GET("/api/subflow/v1/personal/ledger", a.personalLedger).Bind(bind)
	e.Router.GET("/api/subflow/v1/personal/shares", a.listPersonalShares).Bind(bind)
	e.Router.POST("/api/subflow/v1/personal/shares", a.createPersonalShare).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/personal/shares/{id}", a.updateShare).Bind(bind)
	e.Router.POST("/api/subflow/v1/personal/shares/{id}/rotate", a.rotateShare).Bind(bind)
	e.Router.POST("/api/subflow/v1/personal/shares/{id}/remember", a.rememberShare).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/personal/shares/{id}", a.deleteShare).Bind(bind)
	e.Router.GET("/api/subflow/v1/personal/contacts", a.listPersonalContacts).Bind(bind)
	e.Router.POST("/api/subflow/v1/personal/contacts", a.createPersonalContact).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/personal/contacts/{id}", a.updatePersonalContact).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/personal/contacts/{id}", a.deletePersonalContact).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups", a.createGroup).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}", a.getGroup).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/groups/{groupId}", a.updateGroup).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/groups/{groupId}", a.deleteGroup).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/currency-change/preview", a.previewCurrencyChange).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/currency-change", a.changeCurrency).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/summary", a.dashboard).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/ledger", a.groupLedger).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/incomes", a.listGroupIncomes).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/incomes", a.createGroupIncome).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/recurring-incomes", a.listRecurringIncomes).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/recurring-incomes", a.createRecurringIncome).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/groups/{groupId}/recurring-incomes/{id}", a.updateRecurringIncome).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/groups/{groupId}/recurring-incomes/{id}/status", a.changeRecurringIncomeStatus).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/recurring-incomes/{id}/occurrences", a.listIncomeOccurrences).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/recurring-incomes/{id}/occurrences/confirm", a.confirmIncomeOccurrence).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/recurring-incomes/{id}/occurrences/skip", a.skipIncomeOccurrence).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/recurring-incomes/{id}/occurrences/retry", a.retryIncomeOccurrence).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/recurring-incomes/{id}/backfill/preview", a.previewRecurringIncomeBackfill).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/recurring-incomes/{id}/backfill", a.backfillRecurringIncome).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/groups/{groupId}/incomes/{id}", a.updateGroupOrPersonalIncome).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/groups/{groupId}/incomes/{id}", a.deleteGroupOrPersonalIncome).Bind(bind)

	e.Router.GET("/api/subflow/v1/groups/{groupId}/members", a.listMembers).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/access", a.groupAccess).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/shares", a.listGroupShares).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/shares", a.createGroupShare).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/groups/{groupId}/shares/{id}", a.updateShare).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/shares/{id}/rotate", a.rotateShare).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/shares/{id}/remember", a.rememberShare).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/groups/{groupId}/shares/{id}", a.deleteShare).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/groups/{groupId}/members/{userId}", a.removeMember).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/roles", a.listGroupRoles).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/roles", a.createGroupRole).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/groups/{groupId}/roles/{id}", a.updateGroupRole).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/groups/{groupId}/roles/{id}", a.deleteGroupRole).Bind(bind)
	e.Router.PUT("/api/subflow/v1/groups/{groupId}/members/{userId}/role", a.assignGroupRole).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/audit-logs", a.listGroupAudit).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/ownership-transfer", a.getPendingOwnershipTransfer).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/ownership-transfer", a.createOwnershipTransfer).Bind(bind)
	e.Router.POST("/api/subflow/v1/ownership-transfers/{id}/respond", a.respondOwnershipTransfer).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/ownership-transfers/{id}", a.cancelOwnershipTransfer).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/member-transfers", a.listPendingMemberTransfers).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/member-transfers", a.createMemberTransfer).Bind(bind)
	e.Router.POST("/api/subflow/v1/member-transfers/{id}/respond", a.respondMemberTransfer).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/member-transfers/{id}", a.cancelMemberTransfer).Bind(bind)
	e.Router.GET("/api/subflow/v1/system/roles", a.listSystemRoles).Bind(bind)
	e.Router.POST("/api/subflow/v1/system/roles", a.createSystemRole).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/system/roles/{id}", a.updateSystemRole).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/system/roles/{id}", a.deleteSystemRole).Bind(bind)
	e.Router.PUT("/api/subflow/v1/system/users/{userId}/role", a.assignSystemRole).Bind(bind)
	e.Router.GET("/api/subflow/v1/system/users", a.listSystemUsers).Bind(bind)
	e.Router.GET("/api/subflow/v1/system/audit-logs", a.listSystemAudit).Bind(bind)
	e.Router.GET("/api/subflow/v1/system/settings", a.getSystemSettings).Bind(bind)
	e.Router.GET("/api/subflow/v1/system/upgrades", a.listSystemUpgrades).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/system/settings", a.updateSystemSettings).Bind(bind)
	e.Router.POST("/api/subflow/v1/system/recover-admin", a.recoverSystemAdmin).Bind(apis.RequireSuperuserAuth())
	e.Router.GET("/api/subflow/v1/groups/{groupId}/subscriptions", a.listSubscriptions).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/subscriptions", a.createSubscription).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/subscriptions/{id}", a.updateSubscription).Bind(bind)
	e.Router.POST("/api/subflow/v1/subscriptions/{id}/stop", a.stopSubscription).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/subscriptions/{id}/stop", a.resumeSubscription).Bind(bind)
	e.Router.GET("/api/subflow/v1/subscriptions/{id}/billing-dates", a.billingDates).Bind(bind)
	e.Router.GET("/api/subflow/v1/subscriptions/{id}/periods", a.subscriptionPeriods).Bind(bind)
	e.Router.POST("/api/subflow/v1/subscriptions/{id}/backfill", a.backfillSubscriptionPeriods).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/subscriptions/{id}", a.deleteSubscription).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/expenses", a.listExpenses).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/expenses", a.createExpense).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/expenses/{id}", a.updateExpense).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/expenses/{id}", a.deleteExpense).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/settlements", a.listSettlements).Bind(bind)
	e.Router.POST("/api/subflow/v1/groups/{groupId}/settlements", a.createSettlement).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/settlements/{id}", a.updateSettlement).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/settlements/{id}", a.deleteSettlement).Bind(bind)
	e.Router.GET("/api/subflow/v1/attachments", a.listAttachments).Bind(bind)
	e.Router.POST("/api/subflow/v1/attachments", a.uploadAttachment).Bind(bind)
	e.Router.GET("/api/subflow/v1/attachments/{id}/content", a.readAttachment).Bind(bind)
	e.Router.DELETE("/api/subflow/v1/attachments/{id}", a.deleteAttachment).Bind(bind)
	e.Router.PATCH("/api/subflow/v1/attachments/{id}", a.renameAttachment).Bind(bind)
	e.Router.GET("/api/subflow/v1/export/personal", a.exportPersonalLedger).Bind(bind)
	e.Router.GET("/api/subflow/v1/groups/{groupId}/export", a.exportGroupLedger).Bind(bind)
}

func isBusinessWrite(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}

func attachmentTargetFromRequest(e *core.RequestEvent) (application.AttachmentTarget, error) {
	t := application.AttachmentTarget{Type: e.Request.URL.Query().Get("targetType"), ID: e.Request.URL.Query().Get("targetId"), RequestID: e.Request.URL.Query().Get("requestId")}
	if raw := e.Request.URL.Query().Get("billingAt"); raw != "" {
		value, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return t, domain.ErrInvalid
		}
		t.BillingAt = value
	}
	return t, nil
}
func (a *API) listAttachments(e *core.RequestEvent) error {
	target, err := attachmentTargetFromRequest(e)
	if err != nil {
		if auditErr := a.Service.RecordAttachmentFailure(e.Request.Context(), authID(e), "attachment.listed", "invalid_billing_time"); auditErr != nil {
			return fail(e, auditErr)
		}
		return fail(e, err)
	}
	result, err := a.Service.ListAttachments(e.Request.Context(), authID(e), target)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, result, nil)
}
func (a *API) uploadAttachment(e *core.RequestEvent) error {
	auditFailure := func(reason string) error {
		if auditErr := a.Service.RecordAttachmentFailure(e.Request.Context(), authID(e), "attachment.uploaded", reason); auditErr != nil {
			return fail(e, auditErr)
		}
		return fail(e, domain.ErrInvalid)
	}
	const maxMultipartBytes = 1024*1024*1024 + 2*1024*1024
	e.Request.Body = http.MaxBytesReader(e.Response, e.Request.Body, maxMultipartBytes)
	if err := e.Request.ParseMultipartForm(8 * 1024 * 1024); err != nil {
		return auditFailure("invalid_multipart")
	}
	defer e.Request.MultipartForm.RemoveAll()
	var target application.AttachmentTarget
	target.Type = e.Request.FormValue("targetType")
	target.ID = e.Request.FormValue("targetId")
	target.RequestID = e.Request.FormValue("requestId")
	if raw := e.Request.FormValue("billingAt"); raw != "" {
		value, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return auditFailure("invalid_billing_time")
		}
		target.BillingAt = value
	}
	file, header, err := e.Request.FormFile("file")
	if err != nil {
		return auditFailure("missing_file")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 1024*1024*1024+1))
	if err != nil || len(data) > 1024*1024*1024 {
		return auditFailure("request_too_large")
	}
	created, err := a.Service.UploadAttachment(e.Request.Context(), authID(e), target, header.Filename, data)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) readAttachment(e *core.RequestEvent) error {
	content, err := a.Service.ReadAttachment(e.Request.Context(), authID(e), e.Request.PathValue("id"))
	if err != nil {
		return fail(e, err)
	}
	e.Response.Header().Set("Content-Type", content.MediaType)
	disposition := "attachment"
	if strings.HasPrefix(content.MediaType, "image/") || content.MediaType == "application/pdf" {
		disposition = "inline"
	}
	e.Response.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": content.FileName}))
	e.Response.Header().Set("Cache-Control", "private, no-store")
	e.Response.Header().Set("X-Content-Type-Options", "nosniff")
	_, err = e.Response.Write(content.Data)
	return err
}
func (a *API) deleteAttachment(e *core.RequestEvent) error {
	if err := a.Service.DeleteAttachment(e.Request.Context(), authID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) renameAttachment(e *core.RequestEvent) error {
	var input struct {
		FileName *string `json:"fileName"`
	}
	e.Request.Body = http.MaxBytesReader(e.Response, e.Request.Body, 16*1024)
	decoder := json.NewDecoder(e.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || input.FileName == nil {
		return fail(e, a.Service.AttachmentRenameRequestFailure(e.Request.Context(), authID(e), e.Request.PathValue("id")))
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fail(e, a.Service.AttachmentRenameRequestFailure(e.Request.Context(), authID(e), e.Request.PathValue("id")))
	}
	attachment, err := a.Service.RenameAttachment(e.Request.Context(), authID(e), e.Request.PathValue("id"), *input.FileName)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, attachment, nil)
}

func authID(e *core.RequestEvent) string {
	if e.Auth == nil {
		return ""
	}
	return e.Auth.Id
}

func (a *API) logout(e *core.RequestEvent) error {
	if err := a.Service.Logout(e.Request.Context(), authID(e)); err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, map[string]bool{"recorded": true}, nil)
}
func groupID(e *core.RequestEvent) string { return e.Request.PathValue("groupId") }
func ok(e *core.RequestEvent, status int, data any, meta any) error {
	return e.JSON(status, envelope{Data: data, Meta: meta})
}
func noContent(e *core.RequestEvent) error {
	return e.JSON(http.StatusOK, envelope{Data: map[string]bool{"deleted": true}})
}
func fail(e *core.RequestEvent, err error) error {
	status := http.StatusInternalServerError
	code := "internal_error"
	message := "服務暫時無法處理請求"
	switch {
	case func() bool { var validation *application.ShareValidationError; return errors.As(err, &validation) }():
		status, code, message = http.StatusBadRequest, "invalid_request", "請檢查分享資料欄位"
	case errors.Is(err, domain.ErrAuditUnavailable):
		status = http.StatusServiceUnavailable
		code = "audit_unavailable"
		message = "稽核紀錄暫時無法寫入，操作未完成"
	case errors.Is(err, domain.ErrAttachmentFormat):
		status, code = http.StatusUnprocessableEntity, "attachment_format"
	case errors.Is(err, domain.ErrAttachmentName):
		status, code, message = http.StatusBadRequest, "attachment_name", "附件名稱無效"
	case errors.Is(err, domain.ErrAttachmentSize):
		status, code = http.StatusRequestEntityTooLarge, "attachment_size"
	case errors.Is(err, domain.ErrAttachmentCount):
		status, code = http.StatusConflict, "attachment_count"
	case errors.Is(err, domain.ErrAttachmentStorage):
		status, code = http.StatusServiceUnavailable, "attachment_storage"
	case errors.Is(err, domain.ErrConfiguration):
		status = http.StatusServiceUnavailable
		code = "configuration_error"
		message = "分享連結保存功能尚未設定"
	case errors.Is(err, domain.ErrInvalid):
		status = http.StatusBadRequest
		code = "invalid_request"
		message = "輸入資料不正確"
	case errors.Is(err, domain.ErrForbidden):
		status = http.StatusForbidden
		code = "forbidden"
		message = "沒有權限執行此操作"
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
		code = "not_found"
		message = "找不到指定資源"
	case errors.Is(err, domain.ErrConflict):
		status = http.StatusConflict
		code = "conflict"
		message = "資料狀態衝突"
	case errors.Is(err, domain.ErrRateUnavailable):
		status = http.StatusUnprocessableEntity
		code = "rate_unavailable"
		message = "目前無法取得匯率"
	case errors.Is(err, domain.ErrSetupDisabled):
		status = http.StatusGone
		code = "setup_disabled"
		message = "setup is no longer available"
	case errors.Is(err, domain.ErrSetupToken):
		status = http.StatusForbidden
		code = "setup_token_invalid"
		message = "setup link is invalid or expired"
	}
	var fields map[string]string
	var validation *application.ShareValidationError
	if errors.As(err, &validation) {
		fields = validation.Fields
	}
	details := ""
	if errors.Is(err, domain.ErrShareKeyMissing) {
		details = "share_key_missing"
	} else if errors.Is(err, domain.ErrShareCiphertext) {
		details = "share_url_unreadable"
	}
	requestID, _ := e.Request.Context().Value(requestIDContextKey{}).(string)
	return e.JSON(status, errorEnvelope{Error: apiError{Code: code, Message: message, Fields: fields, Details: details, RequestID: requestID}})
}

func (a *API) currencies(e *core.RequestEvent) error {
	return ok(e, http.StatusOK, a.Service.Currencies(), nil)
}

const shareCookieName = "subflow_share_access"

func (a *API) shareAttachmentAccess(e *core.RequestEvent, account bool) (*domain.Share, error) {
	action := "share.attachment.viewed"
	if strings.Contains(e.Request.URL.Path, "/attachments/") {
		action = "share.attachment.listed"
	} else if e.Request.URL.Query().Get("download") == "true" {
		action = "share.attachment.downloaded"
	}
	share, err := a.Service.FindAvailableShare(e.Request.Context(), e.Request.PathValue("token"))
	if err != nil {
		if auditErr := a.Service.AuditMissingShareAccess(e.Request.Context(), authID(e), "attachment_unavailable"); auditErr != nil {
			return nil, auditErr
		}
		return nil, domain.ErrNotFound
	}
	if share.AccessMode == "accounts" {
		if !account || a.Service.AuthorizeShareViewer(e.Request.Context(), share, authID(e)) != nil {
			if auditErr := a.Service.AuditShareAttachmentFailure(e.Request.Context(), share, action, "viewer_not_allowed"); auditErr != nil {
				return nil, auditErr
			}
			return nil, domain.ErrNotFound
		}
	} else if account || (share.AccessMode == "password" && !shareCookieValid(e, share)) {
		if auditErr := a.Service.AuditShareAttachmentFailure(e.Request.Context(), share, action, "share_access_denied"); auditErr != nil {
			return nil, auditErr
		}
		return nil, domain.ErrNotFound
	}
	return share, nil
}

func (a *API) shareAttachmentListFor(e *core.RequestEvent, account bool) error {
	share, err := a.shareAttachmentAccess(e, account)
	if err != nil {
		return fail(e, err)
	}
	items, err := a.Service.ShareAttachmentList(e.Request.Context(), share, e.Request.PathValue("reference"))
	if err != nil {
		return fail(e, err)
	}
	e.Response.Header().Set("Cache-Control", "private, no-store")
	e.Response.Header().Set("X-Content-Type-Options", "nosniff")
	return ok(e, http.StatusOK, items, nil)
}
func (a *API) shareAttachmentList(e *core.RequestEvent) error {
	return a.shareAttachmentListFor(e, false)
}
func (a *API) accountShareAttachmentList(e *core.RequestEvent) error {
	return a.shareAttachmentListFor(e, true)
}

func (a *API) shareAttachmentContentFor(e *core.RequestEvent, account bool) error {
	share, err := a.shareAttachmentAccess(e, account)
	if err != nil {
		return fail(e, err)
	}
	content, err := a.Service.ReadShareAttachment(e.Request.Context(), share, e.Request.PathValue("reference"), e.Request.URL.Query().Get("download") == "true")
	if err != nil {
		return fail(e, err)
	}
	e.Response.Header().Set("Content-Type", content.MediaType)
	disposition := "attachment"
	if e.Request.URL.Query().Get("download") != "true" && (strings.HasPrefix(content.MediaType, "image/") || content.MediaType == "application/pdf") {
		disposition = "inline"
	}
	e.Response.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": content.FileName}))
	e.Response.Header().Set("Cache-Control", "private, no-store")
	e.Response.Header().Set("X-Content-Type-Options", "nosniff")
	_, err = e.Response.Write(content.Data)
	return err
}
func (a *API) shareAttachmentContent(e *core.RequestEvent) error {
	return a.shareAttachmentContentFor(e, false)
}
func (a *API) accountShareAttachmentContent(e *core.RequestEvent) error {
	return a.shareAttachmentContentFor(e, true)
}

func shareCookieSignature(value string) string {
	key := os.Getenv("SUBFLOW_SHARE_SESSION_HMAC_KEY")
	if key == "" {
		key = "subflow-share-local"
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func shareCookieValid(e *core.RequestEvent, value *domain.Share) bool {
	cookie, err := e.Request.Cookie(shareCookieName)
	if err != nil {
		return false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 4 || parts[0] != value.ID || parts[1] != strconv.Itoa(value.AccessVersion) {
		return false
	}
	expires, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || time.Now().Unix() > expires {
		return false
	}
	payload := strings.Join(parts[:3], ".")
	return hmac.Equal([]byte(parts[3]), []byte(shareCookieSignature(payload)))
}
func setShareCookie(e *core.RequestEvent, value *domain.Share) {
	expires := time.Now().Add(8 * time.Hour).Unix()
	payload := value.ID + "." + strconv.Itoa(value.AccessVersion) + "." + strconv.FormatInt(expires, 10)
	http.SetCookie(e.Response, &http.Cookie{Name: shareCookieName, Value: payload + "." + shareCookieSignature(payload), Path: "/api/subflow/v1/shares/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: e.Request.TLS != nil, MaxAge: 8 * 60 * 60})
}
func (a *API) sharePasswordAllowed(ip, shareID string) bool {
	a.shareAttemptMu.Lock()
	defer a.shareAttemptMu.Unlock()
	if a.shareAttempts == nil {
		a.shareAttempts = map[string]shareAttempt{}
	}
	key := ip + ":" + shareID
	attempt := a.shareAttempts[key]
	if time.Now().After(attempt.ResetAt) {
		return true
	}
	return attempt.Count < 5
}
func (a *API) recordSharePasswordFailure(ip, shareID string) {
	a.shareAttemptMu.Lock()
	defer a.shareAttemptMu.Unlock()
	if a.shareAttempts == nil {
		a.shareAttempts = map[string]shareAttempt{}
	}
	key := ip + ":" + shareID
	attempt := a.shareAttempts[key]
	if time.Now().After(attempt.ResetAt) {
		attempt = shareAttempt{ResetAt: time.Now().Add(15 * time.Minute)}
	}
	attempt.Count++
	a.shareAttempts[key] = attempt
}
func (a *API) clearSharePasswordFailures(ip, shareID string) {
	a.shareAttemptMu.Lock()
	defer a.shareAttemptMu.Unlock()
	delete(a.shareAttempts, ip+":"+shareID)
}
func (a *API) publicShare(e *core.RequestEvent) error {
	share, err := a.Service.FindAvailableShare(e.Request.Context(), e.Request.PathValue("token"))
	if err != nil {
		if auditErr := a.Service.AuditMissingShareAccess(e.Request.Context(), "", "unavailable"); auditErr != nil {
			return fail(e, auditErr)
		}
		return fail(e, domain.ErrNotFound)
	}
	if share.AccessMode == "password" && !shareCookieValid(e, share) {
		return ok(e, http.StatusOK, map[string]bool{"requiresPassword": true}, nil)
	}
	if share.AccessMode == "accounts" {
		return ok(e, http.StatusOK, map[string]bool{"requiresLogin": true}, nil)
	}
	page, err := a.Service.SharePageWithOptions(e.Request.Context(), share, sharePageOptions(e))
	if err != nil {
		if auditErr := a.Service.AuditShareAccess(e.Request.Context(), share, "", "failure", "projection_failed"); auditErr != nil {
			return fail(e, auditErr)
		}
		return fail(e, domain.ErrNotFound)
	}
	if err = a.Service.AuditShareAccess(e.Request.Context(), share, "", "success", "link"); err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, page, nil)
}
func (a *API) sharePasswordAccess(e *core.RequestEvent) error {
	share, err := a.Service.FindAvailableShare(e.Request.Context(), e.Request.PathValue("token"))
	if err != nil {
		if auditErr := a.Service.AuditMissingShareAccess(e.Request.Context(), "", "unavailable"); auditErr != nil {
			return fail(e, auditErr)
		}
		return fail(e, domain.ErrNotFound)
	}
	if !a.sharePasswordAllowed(e.RealIP(), share.ID) {
		if auditErr := a.Service.AuditShareAccess(e.Request.Context(), share, "", "failure", "rate_limited"); auditErr != nil {
			return fail(e, auditErr)
		}
		return fail(e, domain.ErrNotFound)
	}
	var body struct {
		Password string `json:"password"`
	}
	if e.BindBody(&body) != nil || a.Service.VerifySharePassword(e.Request.Context(), share, body.Password) != nil {
		a.recordSharePasswordFailure(e.RealIP(), share.ID)
		if auditErr := a.Service.AuditShareAccess(e.Request.Context(), share, "", "failure", "password_invalid"); auditErr != nil {
			return fail(e, auditErr)
		}
		return fail(e, domain.ErrNotFound)
	}
	page, err := a.Service.SharePageWithOptions(e.Request.Context(), share, sharePageOptions(e))
	if err != nil {
		if auditErr := a.Service.AuditShareAccess(e.Request.Context(), share, "", "failure", "projection_failed"); auditErr != nil {
			return fail(e, auditErr)
		}
		return fail(e, domain.ErrNotFound)
	}
	if err = a.Service.AuditShareAccess(e.Request.Context(), share, "", "success", "password"); err != nil {
		return fail(e, err)
	}
	a.clearSharePasswordFailures(e.RealIP(), share.ID)
	setShareCookie(e, share)
	return ok(e, http.StatusOK, page, nil)
}
func (a *API) accountShare(e *core.RequestEvent) error {
	share, err := a.Service.FindAvailableShare(e.Request.Context(), e.Request.PathValue("token"))
	if err != nil {
		if auditErr := a.Service.AuditMissingShareAccess(e.Request.Context(), authID(e), "unavailable"); auditErr != nil {
			return fail(e, auditErr)
		}
		return fail(e, domain.ErrNotFound)
	}
	if err = a.Service.AuthorizeShareViewer(e.Request.Context(), share, authID(e)); err != nil {
		if auditErr := a.Service.AuditShareAccess(e.Request.Context(), share, authID(e), "failure", "viewer_not_allowed"); auditErr != nil {
			return fail(e, auditErr)
		}
		return fail(e, domain.ErrNotFound)
	}
	page, err := a.Service.SharePageWithOptions(e.Request.Context(), share, sharePageOptions(e))
	if err != nil {
		if auditErr := a.Service.AuditShareAccess(e.Request.Context(), share, authID(e), "failure", "projection_failed"); auditErr != nil {
			return fail(e, auditErr)
		}
		return fail(e, domain.ErrNotFound)
	}
	if err = a.Service.AuditShareAccess(e.Request.Context(), share, authID(e), "success", "account"); err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, page, nil)
}
func sharePageOptions(e *core.RequestEvent) application.SharePageOptions {
	section := e.Request.URL.Query().Get("section")
	if section != "expenses" && section != "incomes" && section != "recurring_income" && section != "pending_income" && section != "subscriptions" && section != "settlements" {
		section = "all"
	}
	sortOrder := e.Request.URL.Query().Get("sort")
	if sortOrder != "oldest" {
		sortOrder = "newest"
	}
	return application.SharePageOptions{
		Page:    sharePageNumber(e),
		PerPage: sharePageSize(e),
		Section: section,
		Query:   e.Request.URL.Query().Get("q"),
		Sort:    sortOrder,
	}
}

func sharePageSize(e *core.RequestEvent) int {
	value, err := strconv.Atoi(e.Request.URL.Query().Get("perPage"))
	if err != nil || (value != 5 && value != 10 && value != 15 && value != 25) {
		return 25
	}
	return value
}
func sharePageNumber(e *core.RequestEvent) int {
	value, err := strconv.Atoi(e.Request.URL.Query().Get("page"))
	if err != nil || value < 1 {
		return 1
	}
	return value
}
func (a *API) listPersonalShares(e *core.RequestEvent) error {
	p, err := pageRequest(e, "shares")
	if err != nil {
		return fail(e, err)
	}
	values, err := a.Service.ListShares(e.Request.Context(), authID(e), "", p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, values.Items, pageMeta(values))
}
func (a *API) listGroupShares(e *core.RequestEvent) error {
	p, err := pageRequest(e, "shares")
	if err != nil {
		return fail(e, err)
	}
	values, err := a.Service.ListShares(e.Request.Context(), authID(e), groupID(e), p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, values.Items, pageMeta(values))
}
func (a *API) createPersonalShare(e *core.RequestEvent) error {
	var input application.ShareInput
	if e.BindBody(&input) != nil {
		return fail(e, domain.ErrInvalid)
	}
	value, err := a.Service.CreateShare(e.Request.Context(), authID(e), "", input)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, value, nil)
}
func (a *API) createGroupShare(e *core.RequestEvent) error {
	var input application.ShareInput
	if e.BindBody(&input) != nil {
		return fail(e, domain.ErrInvalid)
	}
	value, err := a.Service.CreateShare(e.Request.Context(), authID(e), groupID(e), input)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, value, nil)
}
func (a *API) updateShare(e *core.RequestEvent) error {
	var input application.ShareInput
	if e.BindBody(&input) != nil {
		return fail(e, domain.ErrInvalid)
	}
	value, err := a.Service.UpdateShare(e.Request.Context(), authID(e), e.Request.PathValue("id"), input)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, value, nil)
}
func (a *API) rotateShare(e *core.RequestEvent) error {
	value, err := a.Service.RotateShare(e.Request.Context(), authID(e), e.Request.PathValue("id"))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, value, nil)
}
func (a *API) rememberShare(e *core.RequestEvent) error {
	var body struct {
		Token string `json:"token"`
	}
	if e.BindBody(&body) != nil || strings.TrimSpace(body.Token) == "" {
		return fail(e, domain.ErrInvalid)
	}
	value, err := a.Service.RememberShareToken(e.Request.Context(), authID(e), e.Request.PathValue("id"), body.Token)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, value, nil)
}
func (a *API) deleteShare(e *core.RequestEvent) error {
	if err := a.Service.DeleteShare(e.Request.Context(), authID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) listPersonalContacts(e *core.RequestEvent) error {
	p, err := pageRequest(e, "contacts")
	if err != nil {
		return fail(e, err)
	}
	values, err := a.Service.ListContacts(e.Request.Context(), authID(e), p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, values.Items, pageMeta(values))
}
func (a *API) createPersonalContact(e *core.RequestEvent) error {
	var input application.ContactInput
	if e.BindBody(&input) != nil {
		return fail(e, domain.ErrInvalid)
	}
	value, err := a.Service.CreateContact(e.Request.Context(), authID(e), input)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, value, nil)
}
func (a *API) updatePersonalContact(e *core.RequestEvent) error {
	var input application.ContactInput
	if e.BindBody(&input) != nil {
		return fail(e, domain.ErrInvalid)
	}
	value, err := a.Service.UpdateContact(e.Request.Context(), authID(e), e.Request.PathValue("id"), input)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, value, nil)
}
func (a *API) deletePersonalContact(e *core.RequestEvent) error {
	if err := a.Service.DeleteContact(e.Request.Context(), authID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) listCategories(e *core.RequestEvent) error {
	q := e.Request.URL.Query()
	values, err := a.Service.ListCategories(e.Request.Context(), authID(e), q.Get("scope"), q.Get("groupId"), q.Get("archived") == "true", q.Get("kind"))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, values, nil)
}
func (a *API) createCategory(e *core.RequestEvent) error {
	var value domain.Category
	if e.BindBody(&value) != nil {
		return fail(e, a.Service.CategoryRequestFailure(e.Request.Context(), authID(e), "category.created", "", domain.ErrInvalid))
	}
	created, err := a.Service.CreateCategory(e.Request.Context(), authID(e), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) updateCategory(e *core.RequestEvent) error {
	var value domain.Category
	if e.BindBody(&value) != nil {
		return fail(e, a.Service.CategoryRequestFailure(e.Request.Context(), authID(e), "category.updated", e.Request.PathValue("id"), domain.ErrInvalid))
	}
	value.ID = e.Request.PathValue("id")
	updated, err := a.Service.UpdateCategory(e.Request.Context(), authID(e), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}
func (a *API) archiveCategory(e *core.RequestEvent) error {
	updated, err := a.Service.UpdateCategory(e.Request.Context(), authID(e), domain.Category{ID: e.Request.PathValue("id"), Archived: true})
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}
func (a *API) exchangeRateQuote(e *core.RequestEvent) error {
	q := e.Request.URL.Query()
	date, err := time.Parse("2006-01-02", q.Get("date"))
	if err != nil {
		return fail(e, domain.ErrInvalid)
	}
	value, err := a.Service.QuoteRate(e.Request.Context(), domain.Currency(q.Get("from")), domain.Currency(q.Get("to")), date)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, value, nil)
}
func currencyBody(e *core.RequestEvent) (domain.Currency, error) {
	var body struct {
		Currency domain.Currency `json:"currency"`
	}
	if e.BindBody(&body) != nil {
		return "", domain.ErrInvalid
	}
	return body.Currency, nil
}
func (a *API) previewCurrencyChange(e *core.RequestEvent) error {
	currency, err := currencyBody(e)
	if err != nil {
		return fail(e, a.Service.CurrencyRequestFailure(e.Request.Context(), authID(e), groupID(e), "group.currency_previewed", err))
	}
	value, err := a.Service.PreviewGroupCurrency(e.Request.Context(), authID(e), groupID(e), currency)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, value, nil)
}
func (a *API) changeCurrency(e *core.RequestEvent) error {
	currency, err := currencyBody(e)
	if err != nil {
		return fail(e, a.Service.CurrencyRequestFailure(e.Request.Context(), authID(e), groupID(e), "group.currency_changed", err))
	}
	value, err := a.Service.ChangeGroupCurrency(e.Request.Context(), authID(e), groupID(e), currency)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, value, nil)
}

var sorts = map[string]map[string]bool{"groups": {"name": true, "-name": true, "created": true, "-created": true}, "members": {"created": true, "-created": true}, "subscriptions": {"name": true, "-name": true, "next_billing": true, "-next_billing": true, "created": true, "-created": true}, "expenses": {"incurred_on": true, "-incurred_on": true, "created": true, "-created": true}, "incomes": {"received_on": true, "-received_on": true, "created": true, "-created": true}, "settlements": {"settled_on": true, "-settled_on": true, "created": true, "-created": true}, "audits": {"created": true, "-created": true}, "notification_emails": {"created": true, "-created": true}}

func pageRequest(e *core.RequestEvent, resource string) (ports.PageRequest, error) {
	q := e.Request.URL.Query()
	p, _ := strconv.Atoi(q.Get("page"))
	pp, _ := strconv.Atoi(q.Get("perPage"))
	sort := q.Get("sort")
	if sort != "" && !sorts[resource][sort] {
		return ports.PageRequest{}, domain.ErrInvalid
	}
	return ports.PageRequest{Page: p, PerPage: pp, Sort: sort}, nil
}
func personalAllScope(e *core.RequestEvent) (bool, error) {
	switch e.Request.URL.Query().Get("scope") {
	case "", "personal":
		return false, nil
	case "all":
		return true, nil
	default:
		return false, domain.ErrInvalid
	}
}

func pageMeta[T any](p ports.Page[T]) map[string]int {
	return map[string]int{"page": p.Page, "perPage": p.PerPage, "totalItems": p.TotalItems, "totalPages": p.TotalPages}
}

func (a *API) auditQuery(e *core.RequestEvent) (ports.AuditQuery, error) {
	page, err := pageRequest(e, "audits")
	if err != nil {
		return ports.AuditQuery{}, err
	}
	query := e.Request.URL.Query()
	result := ports.AuditQuery{PageRequest: page, Query: query.Get("q"), Action: query.Get("action"), Resource: query.Get("resource"), Outcome: query.Get("outcome")}
	if result.Outcome != "" && result.Outcome != "success" && result.Outcome != "failure" {
		return ports.AuditQuery{}, domain.ErrInvalid
	}
	location := time.UTC
	if user, findErr := a.Service.Stores.Users.Get(e.Request.Context(), authID(e)); findErr == nil && user.Timezone != "" {
		if value, loadErr := time.LoadLocation(user.Timezone); loadErr == nil {
			location = value
		}
	}
	parseDate := func(value string, end bool) (time.Time, error) {
		if value == "" {
			return time.Time{}, nil
		}
		date, parseErr := time.ParseInLocation("2006-01-02", value, location)
		if parseErr != nil {
			return time.Time{}, domain.ErrInvalid
		}
		if end {
			date = date.AddDate(0, 0, 1).Add(-time.Nanosecond)
		}
		return date.UTC(), nil
	}
	if result.From, err = parseDate(query.Get("from"), false); err != nil {
		return ports.AuditQuery{}, err
	}
	if result.To, err = parseDate(query.Get("to"), true); err != nil {
		return ports.AuditQuery{}, err
	}
	if !result.From.IsZero() && !result.To.IsZero() && result.From.After(result.To) {
		return ports.AuditQuery{}, domain.ErrInvalid
	}
	return result, nil
}

func (a *API) listGroups(e *core.RequestEvent) error {
	p, err := pageRequest(e, "groups")
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListGroups(e.Request.Context(), authID(e), p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, pageMeta(v))
}
func (a *API) createGroup(e *core.RequestEvent) error {
	var v domain.Group
	if err := e.BindBody(&v); err != nil {
		return fail(e, domain.ErrInvalid)
	}
	created, err := a.Service.CreateGroup(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) getGroup(e *core.RequestEvent) error {
	v, err := a.Service.GetGroup(e.Request.Context(), authID(e), groupID(e))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v, nil)
}
func (a *API) updateGroup(e *core.RequestEvent) error {
	var v domain.Group
	if err := e.BindBody(&v); err != nil {
		return fail(e, domain.ErrInvalid)
	}
	v.ID = groupID(e)
	updated, err := a.Service.UpdateGroup(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}
func (a *API) deleteGroup(e *core.RequestEvent) error {
	if err := a.Service.DeleteGroup(e.Request.Context(), authID(e), groupID(e)); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) dashboard(e *core.RequestEvent) error {
	v, err := a.Service.WorkspaceDashboard(e.Request.Context(), authID(e), application.DashboardQuery{Scope: "group", GroupID: groupID(e), Month: e.Request.URL.Query().Get("month")})
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v, nil)
}
func (a *API) workspaceDashboard(e *core.RequestEvent) error {
	scope := e.Request.URL.Query().Get("scope")
	v, err := a.Service.WorkspaceDashboard(e.Request.Context(), authID(e), application.DashboardQuery{Scope: scope, GroupID: e.Request.URL.Query().Get("groupId"), Month: e.Request.URL.Query().Get("month")})
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v, nil)
}
func (a *API) billingDates(e *core.RequestEvent) error {
	limit, _ := strconv.Atoi(e.Request.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 12
	}
	includePast := e.Request.URL.Query().Get("includePast") == "true"
	value, err := a.Service.BillingDates(e.Request.Context(), authID(e), e.Request.PathValue("id"), e.Request.URL.Query().Get("cursor"), limit, includePast)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, value, nil)
}
func (a *API) subscriptionPeriods(e *core.RequestEvent) error {
	limit, _ := strconv.Atoi(e.Request.URL.Query().Get("limit"))
	value, err := a.Service.SubscriptionPeriods(e.Request.Context(), authID(e), e.Request.PathValue("id"), e.Request.URL.Query().Get("cursor"), limit)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, value, nil)
}
func (a *API) backfillSubscriptionPeriods(e *core.RequestEvent) error {
	created, err := a.Service.BackfillSubscriptionPeriods(e.Request.Context(), authID(e), e.Request.PathValue("id"))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, map[string]int{"created": created}, nil)
}
func (a *API) listMembers(e *core.RequestEvent) error {
	p, err := pageRequest(e, "members")
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListMembers(e.Request.Context(), authID(e), groupID(e), p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, pageMeta(v))
}
func (a *API) removeMember(e *core.RequestEvent) error {
	if err := a.Service.RemoveMember(e.Request.Context(), authID(e), groupID(e), e.Request.PathValue("userId")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}

func (a *API) setupStatus(e *core.RequestEvent) error {
	settings, err := a.Service.SetupStatus(e.Request.Context())
	if err != nil {
		return fail(e, err)
	}
	if settings.Initialized {
		return ok(e, http.StatusOK, map[string]any{"initialized": true, "allowRegistration": settings.AllowPasswordRegistration, "allowPasswordRegistration": settings.AllowPasswordRegistration, "allowOidcRegistration": settings.AllowOIDCRegistration, "captchaProvider": settings.CaptchaProvider, "captchaSiteKey": settings.CaptchaSiteKey, "captchaChallengeUrl": settings.CaptchaChallengeURL, "captchaFlows": settings.CaptchaFlows}, nil)
	}
	_, valid, err := a.Service.ValidateSetupToken(e.Request.Context(), e.Request.URL.Query().Get("token"))
	if err != nil {
		return fail(e, err)
	}
	if !valid {
		return ok(e, http.StatusOK, map[string]any{"initialized": false, "setupAvailable": false}, nil)
	}
	return ok(e, http.StatusOK, map[string]any{"initialized": false, "setupAvailable": true, "siteName": settings.SiteName, "defaultTimezone": settings.DefaultTimezone, "defaultCurrency": settings.DefaultCurrency, "allowRegistration": settings.AllowPasswordRegistration, "allowPasswordRegistration": settings.AllowPasswordRegistration, "allowOidcRegistration": settings.AllowOIDCRegistration, "currencies": a.Service.Currencies()}, nil)
}

func (a *API) captchaChallenge(e *core.RequestEvent) error {
	challenge, err := a.Service.CommunityCaptchaChallenge(e.Request.Context(), e.Request.URL.Query().Get("flow"))
	if err != nil {
		return fail(e, err)
	}
	return e.JSON(http.StatusOK, challenge)
}
func (a *API) listExternalAuths(e *core.RequestEvent) error {
	providers, err := a.Service.ListLinkedProviders(e.Request.Context(), authID(e))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, providers, nil)
}
func (a *API) unlinkExternalAuth(e *core.RequestEvent) error {
	if err := a.Service.UnlinkProvider(e.Request.Context(), authID(e), e.Request.PathValue("provider")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) systemAccess(e *core.RequestEvent) error {
	permissions, err := a.Service.SystemPermissions(e.Request.Context(), authID(e))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, map[string]any{"permissions": permissions}, nil)
}
func (a *API) groupAccess(e *core.RequestEvent) error {
	permissions, err := a.Service.GroupPermissions(e.Request.Context(), authID(e), groupID(e))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, map[string]any{"permissions": permissions}, nil)
}
func (a *API) initializeSetup(e *core.RequestEvent) error {
	var value domain.SetupInput
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	created, err := a.Service.InitializeSetup(e.Request.Context(), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, map[string]string{"id": created.ID}, nil)
}
func (a *API) register(e *core.RequestEvent) error {
	var value domain.SetupInput
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	if err := a.Service.VerifyCaptcha(e.Request.Context(), domain.CaptchaFlowRegister, value.CaptchaToken, e.RealIP()); err != nil {
		return fail(e, domain.ErrForbidden)
	}
	created, err := a.Service.Register(e.Request.Context(), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, map[string]string{"id": created.ID}, nil)
}
func (a *API) listGroupRoles(e *core.RequestEvent) error {
	values, err := a.Service.ListGroupRoles(e.Request.Context(), authID(e), groupID(e))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, values, nil)
}
func (a *API) createGroupRole(e *core.RequestEvent) error {
	var value domain.Role
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	value.GroupID = groupID(e)
	v, err := a.Service.CreateGroupRole(e.Request.Context(), authID(e), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, v, nil)
}
func (a *API) updateGroupRole(e *core.RequestEvent) error {
	var value domain.Role
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	value.ID = e.Request.PathValue("id")
	v, err := a.Service.UpdateGroupRole(e.Request.Context(), authID(e), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v, nil)
}
func (a *API) deleteGroupRole(e *core.RequestEvent) error {
	if err := a.Service.DeleteGroupRole(e.Request.Context(), authID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) assignGroupRole(e *core.RequestEvent) error {
	var body struct {
		RoleID string `json:"roleId"`
	}
	if e.BindBody(&body) != nil || body.RoleID == "" {
		return fail(e, domain.ErrInvalid)
	}
	if err := a.Service.AssignGroupRole(e.Request.Context(), authID(e), groupID(e), e.Request.PathValue("userId"), body.RoleID); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) listGroupAudit(e *core.RequestEvent) error {
	p, err := a.auditQuery(e)
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListGroupAudit(e.Request.Context(), authID(e), groupID(e), p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, pageMeta(v))
}
func (a *API) getPendingOwnershipTransfer(e *core.RequestEvent) error {
	transfer, err := a.Service.PendingOwnershipTransfer(e.Request.Context(), authID(e), groupID(e))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, transfer, nil)
}
func (a *API) createOwnershipTransfer(e *core.RequestEvent) error {
	var body struct {
		ToUserID string `json:"toUserId"`
	}
	if e.BindBody(&body) != nil || body.ToUserID == "" {
		return fail(e, domain.ErrInvalid)
	}
	transfer, err := a.Service.CreateOwnershipTransfer(e.Request.Context(), authID(e), groupID(e), body.ToUserID)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, transfer, nil)
}
func (a *API) respondOwnershipTransfer(e *core.RequestEvent) error {
	var body struct {
		Accept bool `json:"accept"`
	}
	if e.BindBody(&body) != nil {
		return fail(e, domain.ErrInvalid)
	}
	transfer, err := a.Service.RespondOwnershipTransfer(e.Request.Context(), authID(e), e.Request.PathValue("id"), body.Accept)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, transfer, nil)
}
func (a *API) cancelOwnershipTransfer(e *core.RequestEvent) error {
	if err := a.Service.CancelOwnershipTransfer(e.Request.Context(), authID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) listPendingMemberTransfers(e *core.RequestEvent) error {
	transfers, err := a.Service.PendingMemberTransfers(e.Request.Context(), authID(e), groupID(e))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, transfers, nil)
}
func (a *API) createMemberTransfer(e *core.RequestEvent) error {
	var body struct {
		FromUserID string `json:"fromUserId"`
		ToUserID   string `json:"toUserId"`
	}
	if e.BindBody(&body) != nil || body.FromUserID == "" || body.ToUserID == "" {
		return fail(e, domain.ErrInvalid)
	}
	transfer, err := a.Service.CreateMemberTransfer(e.Request.Context(), authID(e), groupID(e), body.FromUserID, body.ToUserID)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, transfer, nil)
}
func (a *API) respondMemberTransfer(e *core.RequestEvent) error {
	var body struct {
		Accept bool `json:"accept"`
	}
	if e.BindBody(&body) != nil {
		return fail(e, domain.ErrInvalid)
	}
	transfer, err := a.Service.RespondMemberTransfer(e.Request.Context(), authID(e), e.Request.PathValue("id"), body.Accept)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, transfer, nil)
}
func (a *API) cancelMemberTransfer(e *core.RequestEvent) error {
	if err := a.Service.CancelMemberTransfer(e.Request.Context(), authID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) listSubscriptions(e *core.RequestEvent) error {
	p, err := pageRequest(e, "subscriptions")
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListSubscriptions(e.Request.Context(), authID(e), groupID(e), p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, pageMeta(v))
}
func (a *API) listPersonalSubscriptions(e *core.RequestEvent) error {
	p, err := pageRequest(e, "subscriptions")
	if err != nil {
		return fail(e, err)
	}
	all, err := personalAllScope(e)
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListPersonalSubscriptions(e.Request.Context(), authID(e), p)
	if all {
		v, err = a.Service.ListAllScopeSubscriptions(e.Request.Context(), authID(e), p)
	}
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, pageMeta(v))
}
func (a *API) createPersonalSubscription(e *core.RequestEvent) error {
	var v domain.Subscription
	if e.BindBody(&v) != nil {
		return fail(e, domain.ErrInvalid)
	}
	created, err := a.Service.CreateSubscription(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) createSubscription(e *core.RequestEvent) error {
	var v domain.Subscription
	if err := e.BindBody(&v); err != nil {
		return fail(e, domain.ErrInvalid)
	}
	v.GroupID = groupID(e)
	created, err := a.Service.CreateSubscription(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) updateSubscription(e *core.RequestEvent) error {
	var v domain.Subscription
	if err := e.BindBody(&v); err != nil {
		return fail(e, domain.ErrInvalid)
	}
	v.ID = e.Request.PathValue("id")
	updated, err := a.Service.UpdateSubscription(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}
func (a *API) stopSubscription(e *core.RequestEvent) error {
	var body struct {
		EndsOn string `json:"endsOn"`
	}
	if e.BindBody(&body) != nil {
		return fail(e, domain.ErrInvalid)
	}
	updated, err := a.Service.StopSubscription(e.Request.Context(), authID(e), e.Request.PathValue("id"), body.EndsOn)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}
func (a *API) resumeSubscription(e *core.RequestEvent) error {
	updated, err := a.Service.ResumeSubscription(e.Request.Context(), authID(e), e.Request.PathValue("id"))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}
func (a *API) deleteSubscription(e *core.RequestEvent) error {
	if err := a.Service.DeleteSubscription(e.Request.Context(), authID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) listExpenses(e *core.RequestEvent) error {
	p, err := pageRequest(e, "expenses")
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListExpenses(e.Request.Context(), authID(e), groupID(e), p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, pageMeta(v))
}
func (a *API) listPersonalExpenses(e *core.RequestEvent) error {
	p, err := pageRequest(e, "expenses")
	if err != nil {
		return fail(e, err)
	}
	all, err := personalAllScope(e)
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListPersonalExpenses(e.Request.Context(), authID(e), p)
	if all {
		v, err = a.Service.ListAllScopeExpenses(e.Request.Context(), authID(e), p)
	}
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, pageMeta(v))
}
func (a *API) createPersonalExpense(e *core.RequestEvent) error {
	var v domain.Expense
	if e.BindBody(&v) != nil {
		return fail(e, domain.ErrInvalid)
	}
	if v.PaidBy == "" {
		v.PaidBy = authID(e)
	}
	created, err := a.Service.CreateExpense(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) listPersonalIncomes(e *core.RequestEvent) error {
	p, err := pageRequest(e, "incomes")
	if err != nil {
		return fail(e, err)
	}
	all, err := personalAllScope(e)
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListPersonalIncomes(e.Request.Context(), authID(e), p)
	if all {
		v, err = a.Service.ListAllScopeIncomes(e.Request.Context(), authID(e), p)
	}
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, pageMeta(v))
}
func (a *API) createPersonalIncome(e *core.RequestEvent) error {
	var v domain.Income
	if err := e.BindBody(&v); err != nil {
		return fail(e, a.Service.IncomeRequestFailure(e.Request.Context(), authID(e), "", "income.created", "", domain.ErrInvalid))
	}
	created, err := a.Service.CreateIncome(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) updatePersonalIncome(e *core.RequestEvent) error {
	var v domain.Income
	if err := e.BindBody(&v); err != nil {
		return fail(e, domain.ErrInvalid)
	}
	v.ID = e.Request.PathValue("id")
	updated, err := a.Service.UpdateIncome(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}
func (a *API) deletePersonalIncome(e *core.RequestEvent) error {
	if err := a.Service.DeleteIncome(e.Request.Context(), authID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) personalLedger(e *core.RequestEvent) error {
	all, err := personalAllScope(e)
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.PersonalLedger(e.Request.Context(), authID(e), e.Request.URL.Query().Get("date"), all)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v, nil)
}

func (a *API) listGroupIncomes(e *core.RequestEvent) error {
	p, err := pageRequest(e, "incomes")
	if err != nil {
		return fail(e, err)
	}
	v, err := a.Service.ListGroupIncomes(e.Request.Context(), authID(e), groupID(e), p)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v.Items, pageMeta(v))
}
func (a *API) createGroupIncome(e *core.RequestEvent) error {
	var v domain.Income
	if err := e.BindBody(&v); err != nil {
		return fail(e, a.Service.IncomeRequestFailure(e.Request.Context(), authID(e), groupID(e), "income.created", "", domain.ErrInvalid))
	}
	v.GroupID = groupID(e)
	if v.EarnedBy == "" {
		v.EarnedBy = authID(e)
	}
	created, err := a.Service.CreateGroupIncome(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) groupLedger(e *core.RequestEvent) error {
	v, err := a.Service.GroupLedger(e.Request.Context(), authID(e), groupID(e), e.Request.URL.Query().Get("date"))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v, nil)
}

func (a *API) createExpense(e *core.RequestEvent) error {
	var v domain.Expense
	if err := e.BindBody(&v); err != nil {
		return fail(e, domain.ErrInvalid)
	}
	v.GroupID = groupID(e)
	if v.PaidBy == "" {
		v.PaidBy = authID(e)
	}
	created, err := a.Service.CreateExpense(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) updateExpense(e *core.RequestEvent) error {
	var v domain.Expense
	if err := e.BindBody(&v); err != nil {
		return fail(e, domain.ErrInvalid)
	}
	v.ID = e.Request.PathValue("id")
	updated, err := a.Service.UpdateExpense(e.Request.Context(), authID(e), v)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}
func (a *API) deleteExpense(e *core.RequestEvent) error {
	if err := a.Service.DeleteExpense(e.Request.Context(), authID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) settlementQuery(e *core.RequestEvent) (ports.SettlementQuery, error) {
	page, err := pageRequest(e, "settlements")
	if err != nil {
		return ports.SettlementQuery{}, err
	}
	query := e.Request.URL.Query()
	result := ports.SettlementQuery{PageRequest: page, MemberID: query.Get("memberId")}
	location := time.UTC
	if user, findErr := a.Service.Stores.Users.Get(e.Request.Context(), authID(e)); findErr == nil && user.Timezone != "" {
		if value, loadErr := time.LoadLocation(user.Timezone); loadErr == nil {
			location = value
		}
	}
	parseDate := func(value string, end bool) (time.Time, error) {
		if value == "" {
			return time.Time{}, nil
		}
		date, parseErr := time.ParseInLocation("2006-01-02", value, location)
		if parseErr != nil {
			return time.Time{}, domain.ErrInvalid
		}
		if end {
			date = date.AddDate(0, 0, 1).Add(-time.Nanosecond)
		}
		return date.UTC(), nil
	}
	if result.From, err = parseDate(query.Get("from"), false); err != nil {
		return ports.SettlementQuery{}, err
	}
	if result.To, err = parseDate(query.Get("to"), true); err != nil {
		return ports.SettlementQuery{}, err
	}
	if !result.From.IsZero() && !result.To.IsZero() && result.From.After(result.To) {
		return ports.SettlementQuery{}, domain.ErrInvalid
	}
	return result, nil
}
func (a *API) listSettlements(e *core.RequestEvent) error {
	query, err := a.settlementQuery(e)
	if err != nil {
		return fail(e, err)
	}
	values, err := a.Service.ListSettlements(e.Request.Context(), authID(e), groupID(e), query)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, values.Items, pageMeta(values))
}
func (a *API) createSettlement(e *core.RequestEvent) error {
	var value domain.Settlement
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	value.GroupID = groupID(e)
	created, err := a.Service.CreateSettlement(e.Request.Context(), authID(e), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, created, nil)
}
func (a *API) updateSettlement(e *core.RequestEvent) error {
	var value domain.Settlement
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	updated, err := a.Service.UpdateSettlement(e.Request.Context(), authID(e), e.Request.PathValue("id"), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, updated, nil)
}
func (a *API) deleteSettlement(e *core.RequestEvent) error {
	if err := a.Service.DeleteSettlement(e.Request.Context(), authID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) exportPersonalLedger(e *core.RequestEvent) error {
	query := e.Request.URL.Query()
	includePersonal := query.Get("includePersonal") != "false"
	groupIDs := make([]string, 0)
	for _, value := range query["groupIds"] {
		for _, id := range strings.Split(value, ",") {
			if id = strings.TrimSpace(id); id != "" {
				groupIDs = append(groupIDs, id)
			}
		}
	}
	data, filename, err := a.Service.ExportLedgerSources(e.Request.Context(), authID(e), includePersonal, groupIDs, query.Get("includeForecast") == "true", query.Get("locale"))
	if err != nil {
		return fail(e, err)
	}
	e.Response.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	return e.Blob(http.StatusOK, "text/csv; charset=utf-8", data)
}
func (a *API) exportGroupLedger(e *core.RequestEvent) error {
	query := e.Request.URL.Query()
	data, filename, err := a.Service.ExportLedgerSources(e.Request.Context(), authID(e), false, []string{groupID(e)}, query.Get("includeForecast") == "true", query.Get("locale"))
	if err != nil {
		return fail(e, err)
	}
	e.Response.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	return e.Blob(http.StatusOK, "text/csv; charset=utf-8", data)
}
func (a *API) listSystemRoles(e *core.RequestEvent) error {
	values, err := a.Service.ListSystemRoles(e.Request.Context(), authID(e))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, values, nil)
}
func (a *API) createSystemRole(e *core.RequestEvent) error {
	var value domain.Role
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	v, err := a.Service.CreateSystemRole(e.Request.Context(), authID(e), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusCreated, v, nil)
}
func (a *API) updateSystemRole(e *core.RequestEvent) error {
	var value domain.Role
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	value.ID = e.Request.PathValue("id")
	v, err := a.Service.UpdateSystemRole(e.Request.Context(), authID(e), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, v, nil)
}
func (a *API) deleteSystemRole(e *core.RequestEvent) error {
	if err := a.Service.DeleteSystemRole(e.Request.Context(), authID(e), e.Request.PathValue("id")); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) assignSystemRole(e *core.RequestEvent) error {
	var body struct {
		RoleID string `json:"roleId"`
	}
	if e.BindBody(&body) != nil || body.RoleID == "" {
		return fail(e, domain.ErrInvalid)
	}
	if err := a.Service.AssignSystemRole(e.Request.Context(), authID(e), e.Request.PathValue("userId"), body.RoleID); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
func (a *API) listSystemUsers(e *core.RequestEvent) error {
	page, err := pageRequest(e, "users")
	if err != nil {
		return fail(e, err)
	}
	values, err := a.Service.ListSystemUsers(e.Request.Context(), authID(e), page, e.Request.URL.Query().Get("q"))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, values.Items, pageMeta(values))
}
func (a *API) listSystemAudit(e *core.RequestEvent) error {
	page, err := a.auditQuery(e)
	if err != nil {
		return fail(e, err)
	}
	values, err := a.Service.ListSystemAudit(e.Request.Context(), authID(e), page)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, values.Items, pageMeta(values))
}
func (a *API) getSystemSettings(e *core.RequestEvent) error {
	value, err := a.Service.GetSystemSettings(e.Request.Context(), authID(e))
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, value, nil)
}
func (a *API) updateSystemSettings(e *core.RequestEvent) error {
	var value domain.SystemSettings
	if e.BindBody(&value) != nil {
		return fail(e, domain.ErrInvalid)
	}
	result, err := a.Service.UpdateSystemSettings(e.Request.Context(), authID(e), value)
	if err != nil {
		return fail(e, err)
	}
	return ok(e, http.StatusOK, result, nil)
}
func (a *API) recoverSystemAdmin(e *core.RequestEvent) error {
	var body struct {
		UserID string `json:"userId"`
	}
	if e.BindBody(&body) != nil || body.UserID == "" {
		return fail(e, domain.ErrInvalid)
	}
	if err := a.Service.RecoverSystemAdmin(e.Request.Context(), body.UserID); err != nil {
		return fail(e, err)
	}
	return noContent(e)
}
