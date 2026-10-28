package application

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"subflow/internal/domain"
)

const attachmentDefaultSize int64 = 20 * 1024 * 1024
const attachmentDefaultCount = 10

type AttachmentTarget struct {
	Type      string    `json:"targetType"`
	ID        string    `json:"targetId"`
	BillingAt time.Time `json:"billingAt"`
	RequestID string    `json:"requestId"`
}
type AttachmentContent struct {
	Data                []byte
	FileName, MediaType string
}

func (s *Service) attachmentTarget(ctx context.Context, userID string, target AttachmentTarget) (domain.Attachment, string, error) {
	a := domain.Attachment{TargetType: target.Type, TargetID: target.ID, UploadedBy: userID, RequestID: target.RequestID, BillingAt: target.BillingAt}
	if target.ID == "" {
		return a, "", domain.ErrInvalid
	}
	permission := ""
	switch target.Type {
	case "expense":
		v, e := s.Stores.Expenses.Get(ctx, target.ID)
		if e != nil {
			return a, "", e
		}
		a.GroupID, a.OwnerID = v.GroupID, v.OwnerID
		permission = "ledger.expenses.read"
		if v.SubscriptionID != "" {
			if _, subErr := s.Stores.Subscriptions.Get(ctx, v.SubscriptionID); subErr == nil {
				// The canonical period target must have the same read boundary
				// in list/upload as in download/rename/delete.
				if a.GroupID != "" {
					if err := s.groupPermission(ctx, userID, a.GroupID, "ledger.expenses.read"); err != nil {
						return a, "", err
					}
					permission = "ledger.subscriptions.read"
				}
				a.TargetType = "subscription_period"
				a.TargetID = v.SubscriptionID
				a.BillingAt = v.IncurredOn
			} else if !errors.Is(subErr, domain.ErrNotFound) {
				return a, "", subErr
			}
		}
	case "income":
		v, e := s.Stores.Incomes.Get(ctx, target.ID)
		if e != nil {
			return a, "", e
		}
		a.GroupID, a.OwnerID = v.GroupID, v.OwnerID
		permission = "ledger.incomes.read"
	case "settlement":
		v, e := s.Stores.Settlements.Get(ctx, target.ID)
		if e != nil {
			return a, "", e
		}
		a.GroupID = v.GroupID
		permission = "ledger.settlements.read"
	case "subscription":
		v, e := s.Stores.Subscriptions.Get(ctx, target.ID)
		if e != nil {
			return a, "", e
		}
		a.GroupID, a.OwnerID = v.GroupID, v.OwnerID
		permission = "ledger.subscriptions.read"
	case "subscription_period":
		if target.BillingAt.IsZero() {
			return a, "", domain.ErrInvalid
		}
		v, e := s.Stores.Subscriptions.Get(ctx, target.ID)
		if e != nil {
			return a, "", e
		}
		a.GroupID, a.OwnerID = v.GroupID, v.OwnerID
		permission = "ledger.subscriptions.read"
		loc := s.accountingLocation(ctx, userID, v.GroupID)
		a.BillingAt = target.BillingAt.In(loc)
		// Keep the timestamp anchored to the subscription's actual schedule.
		// This also allows past scheduled periods after a schedule edit, whose
		// occurrence is no longer part of the current generated list.
		occurrences, occurrenceErr := s.Stores.Subscriptions.ListOccurrences(ctx, v.ID)
		if occurrenceErr != nil {
			return a, "", occurrenceErr
		}
		matched := false
		for _, occurrence := range occurrences {
			if occurrence.BillingAt.In(loc).Equal(a.BillingAt) {
				matched = true
				break
			}
		}
		if !matched && !subscriptionBillingDateAllowed(*v, a.BillingAt, loc, true) {
			return a, "", domain.ErrNotFound
		}
	default:
		return a, "", domain.ErrInvalid
	}
	if a.GroupID == "" {
		if a.OwnerID != userID {
			return a, "", domain.ErrForbidden
		}
	} else if e := s.groupPermission(ctx, userID, a.GroupID, permission); e != nil {
		return a, "", e
	}
	return a, permission, nil
}

func attachmentLimits(ctx context.Context, s *Service, groupID string) (domain.AttachmentLimits, error) {
	settings, e := s.Stores.Settings.Get(ctx)
	if e != nil {
		return domain.AttachmentLimits{}, e
	}
	size, count := settings.AttachmentDefaultMaxSize, settings.AttachmentDefaultMaxCount
	if size <= 0 {
		size = attachmentDefaultSize
	}
	if count <= 0 {
		count = attachmentDefaultCount
	}
	maxSize, maxCount := settings.AttachmentLimitMaxSize, settings.AttachmentLimitMaxCount
	if maxSize <= 0 {
		maxSize = attachmentDefaultSize
	}
	if maxCount <= 0 {
		maxCount = attachmentDefaultCount
	}
	if groupID != "" {
		if g, err := s.Stores.Groups.Get(ctx, groupID); err == nil {
			if g.AttachmentMaxSize > 0 {
				size = g.AttachmentMaxSize
			}
			if g.AttachmentMaxCount > 0 {
				count = g.AttachmentMaxCount
			}
		} else {
			return domain.AttachmentLimits{}, err
		}
	}
	if size > maxSize {
		size = maxSize
	}
	if count > maxCount {
		count = maxCount
	}
	return domain.AttachmentLimits{MaxSize: size, MaxCount: count}, nil
}

func (s *Service) attachmentFailure(ctx context.Context, userID, groupID, action, id, reason string) error {
	if err := s.audit(ctx, userID, groupID, action, "attachment", id, "failure", encodeAuditSummary(map[string]any{"reason": reason}, nil)); err != nil {
		slog.ErrorContext(ctx, "attachment failure audit unavailable", "action", action, "reason", reason)
		return errors.Join(domain.ErrAuditUnavailable, err)
	}
	return nil
}

func (s *Service) RecordAttachmentFailure(ctx context.Context, userID, action, reason string) error {
	return s.attachmentFailure(ctx, userID, "", action, "", reason)
}

func (s *Service) attachmentSuccess(ctx context.Context, actor, group, action, id, summary string) error {
	if err := s.audit(ctx, actor, group, action, "attachment", id, "success", summary); err != nil {
		slog.ErrorContext(ctx, "attachment success audit unavailable", "action", action)
		return errors.Join(domain.ErrAuditUnavailable, err)
	}
	return nil
}

func (s *Service) ListAttachments(ctx context.Context, userID string, target AttachmentTarget) (domain.AttachmentPage, error) {
	a, _, e := s.attachmentTarget(ctx, userID, target)
	if e != nil {
		if auditErr := s.attachmentFailure(ctx, userID, "", "attachment.listed", target.ID, "target_unavailable"); auditErr != nil {
			return domain.AttachmentPage{}, auditErr
		}
		return domain.AttachmentPage{}, e
	}
	page := domain.AttachmentPage{}
	page.Limits, e = attachmentLimits(ctx, s, a.GroupID)
	if e != nil {
		if err := s.attachmentFailure(ctx, userID, a.GroupID, "attachment.listed", target.ID, "limits_unavailable"); err != nil {
			return page, err
		}
		return page, e
	}
	if a.GroupID == "" {
		page.Access = domain.AttachmentAccess{CanRead: true, CanUpload: true, CanDelete: true}
	} else {
		perms, err := s.GroupPermissions(ctx, userID, a.GroupID)
		if err != nil {
			if auditErr := s.attachmentFailure(ctx, userID, a.GroupID, "attachment.listed", target.ID, "permission_unavailable"); auditErr != nil {
				return page, auditErr
			}
			return page, err
		}
		has := func(want string) bool {
			for _, p := range perms {
				if p == want {
					return true
				}
			}
			return false
		}
		page.Access = domain.AttachmentAccess{CanRead: has("ledger.attachments.read"), CanUpload: has("ledger.attachments.upload"), CanDelete: has("ledger.attachments.delete")}
		if page.Access.CanUpload || page.Access.CanDelete {
			if err := s.attachmentHistoricalWrite(ctx, userID, a); errors.Is(err, domain.ErrForbidden) {
				page.Access.CanUpload = false
				page.Access.CanDelete = false
			} else if err != nil {
				if auditErr := s.attachmentFailure(ctx, userID, a.GroupID, "attachment.listed", target.ID, "permission_unavailable"); auditErr != nil {
					return page, auditErr
				}
				return page, err
			}
		}
	}
	if !page.Access.CanRead {
		if auditErr := s.attachmentFailure(ctx, userID, a.GroupID, "attachment.listed", a.TargetID, "permission_denied"); auditErr != nil {
			return domain.AttachmentPage{}, auditErr
		}
		return page, domain.ErrForbidden
	}
	page.Access.CanRename = page.Access.CanUpload
	page.Items, e = s.Stores.Attachments.List(ctx, a.TargetType, a.TargetID, a.BillingAt)
	if e != nil {
		if err := s.attachmentFailure(ctx, userID, a.GroupID, "attachment.listed", target.ID, "storage_list_failed"); err != nil {
			return page, err
		}
		return page, e
	}
	if e = s.attachmentSuccess(ctx, userID, a.GroupID, "attachment.listed", a.TargetID, encodeAuditSummary(map[string]any{"target_type": a.TargetType, "count": len(page.Items)}, nil)); e != nil {
		return domain.AttachmentPage{}, e
	}
	return page, nil
}

func attachmentMedia(name string, data []byte) (string, bool) {
	ext := strings.ToLower(filepath.Ext(name))
	if len(data) == 0 {
		return "", false
	}
	switch ext {
	case ".jpg", ".jpeg":
		if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
			return "image/jpeg", true
		}
	case ".png":
		if bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
			return "image/png", true
		}
	case ".webp":
		if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
			return "image/webp", true
		}
	case ".heic", ".heif":
		if len(data) >= 32 && string(data[4:8]) == "ftyp" && (bytes.Contains(data[:32], []byte("heic")) || bytes.Contains(data[:32], []byte("heix")) || bytes.Contains(data[:32], []byte("mif1"))) {
			return "image/heic", true
		}
	case ".pdf":
		if bytes.HasPrefix(data, []byte("%PDF-")) {
			return "application/pdf", true
		}
	case ".docx":
		if officeZipIsSafe(data, ext) {
			return "application/vnd.openxmlformats-officedocument.wordprocessingml.document", true
		}
	case ".xlsx":
		if officeZipIsSafe(data, ext) {
			return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", true
		}
	case ".pptx":
		if officeZipIsSafe(data, ext) {
			return "application/vnd.openxmlformats-officedocument.presentationml.presentation", true
		}
	case ".doc", ".xls", ".ppt":
		if bytes.HasPrefix(data, []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}) && !containsMacroStorage(data) {
			return http.DetectContentType(data), true
		}
	}
	return "", false
}

func containsMacroStorage(data []byte) bool {
	lower := bytes.ToLower(data)
	return bytes.Contains(lower, []byte("vba")) || bytes.Contains(lower, []byte{'v', 0, 'b', 0, 'a', 0})
}

func officeZipIsSafe(data []byte, ext string) bool {
	if !bytes.HasPrefix(data, []byte("PK\x03\x04")) {
		return false
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return false
	}
	contentTypes, mainDocument := false, false
	wantMain := map[string]string{".docx": "word/document.xml", ".xlsx": "xl/workbook.xml", ".pptx": "ppt/presentation.xml"}[ext]
	for _, file := range archive.File {
		name := strings.ToLower(file.Name)
		if name == wantMain {
			mainDocument = true
		}
		if strings.Contains(name, "vbaproject.bin") || strings.HasSuffix(name, ".xlsm") || strings.HasSuffix(name, ".docm") || strings.HasSuffix(name, ".pptm") {
			return false
		}
		if name == "[content_types].xml" {
			contentTypes = true
			reader, openErr := file.Open()
			if openErr != nil {
				return false
			}
			content, readErr := io.ReadAll(io.LimitReader(reader, 2*1024*1024))
			_ = reader.Close()
			if readErr != nil || bytes.Contains(bytes.ToLower(content), []byte("macroenabled")) {
				return false
			}
		}
	}
	return contentTypes && mainDocument
}

func (s *Service) UploadAttachment(ctx context.Context, userID string, target AttachmentTarget, name string, data []byte) (*domain.Attachment, error) {
	a, _, e := s.attachmentTarget(ctx, userID, target)
	if e != nil {
		if auditErr := s.RecordAttachmentFailure(ctx, userID, "attachment.uploaded", "target_unavailable"); auditErr != nil {
			return nil, auditErr
		}
		return nil, e
	}
	fail := func(err error) (*domain.Attachment, error) {
		reason := "storage_write_failed"
		switch {
		case errors.Is(err, domain.ErrInvalid):
			reason = "invalid_request"
		case errors.Is(err, domain.ErrForbidden):
			reason = "permission_denied"
		case errors.Is(err, domain.ErrAttachmentFormat):
			reason = "unsupported_format"
		case errors.Is(err, domain.ErrAttachmentSize):
			reason = "file_too_large"
		case errors.Is(err, domain.ErrAttachmentCount):
			reason = "count_limit"
		case errors.Is(err, domain.ErrConflict):
			reason = "request_conflict"
		case errors.Is(err, domain.ErrAuditUnavailable):
			reason = "audit_unavailable"
		}
		if auditErr := s.audit(ctx, userID, a.GroupID, "attachment.uploaded", "attachment", a.TargetID, "failure", encodeAuditSummary(map[string]any{"reason": reason, "target_type": a.TargetType}, nil)); auditErr != nil {
			slog.ErrorContext(ctx, "attachment failure audit unavailable", "action", "attachment.uploaded", "reason", reason)
			return nil, errors.Join(domain.ErrAuditUnavailable, err, auditErr)
		}
		return nil, err
	}
	if target.RequestID == "" || len(target.RequestID) > 64 {
		return fail(domain.ErrInvalid)
	}
	if a.GroupID != "" {
		if e = s.groupPermission(ctx, userID, a.GroupID, "ledger.attachments.read"); e != nil {
			return fail(e)
		}
		if e = s.groupPermission(ctx, userID, a.GroupID, "ledger.attachments.upload"); e != nil {
			return fail(e)
		}
		if e = s.attachmentHistoricalWrite(ctx, userID, a); e != nil {
			return fail(e)
		}
	}
	if old, err := s.Stores.Attachments.FindByRequest(ctx, userID, target.RequestID); err == nil {
		if old.TargetType != a.TargetType || old.TargetID != a.TargetID || !old.BillingAt.Equal(a.BillingAt) {
			return fail(domain.ErrConflict)
		}
		if err := s.attachmentSuccess(ctx, userID, a.GroupID, "attachment.uploaded", a.TargetID, encodeAuditSummary(map[string]any{"target_type": a.TargetType, "replayed": true}, nil)); err != nil {
			return nil, err
		}
		return old, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return fail(err)
	}
	limits, e := attachmentLimits(ctx, s, a.GroupID)
	if e != nil {
		return fail(e)
	}
	media, ok := attachmentMedia(name, data)
	if int64(len(data)) > limits.MaxSize {
		return fail(domain.ErrAttachmentSize)
	}
	if !ok {
		return fail(domain.ErrAttachmentFormat)
	}
	name = strings.ReplaceAll(name, "\\", "/")
	a.FileName = filepath.Base(name)
	a.MediaType = media
	e = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		items, listErr := s.Stores.Attachments.List(tx, a.TargetType, a.TargetID, a.BillingAt)
		if listErr != nil {
			return fmt.Errorf("%w: %w", domain.ErrAttachmentStorage, listErr)
		}
		if len(items) >= limits.MaxCount {
			return domain.ErrAttachmentCount
		}
		if err := s.Stores.Attachments.Create(tx, &a, data); err != nil {
			return fmt.Errorf("%w: %w", domain.ErrAttachmentStorage, err)
		}
		return s.attachmentSuccess(tx, userID, a.GroupID, "attachment.uploaded", a.TargetID, encodeAuditSummary(map[string]any{"target_type": a.TargetType, "size": len(data)}, nil))
	})
	if e != nil {
		if errors.Is(e, domain.ErrAttachmentStorage) {
			slog.ErrorContext(ctx, "attachment storage operation failed", "error", e)
		}
		return fail(e)
	}
	return &a, nil
}

func (s *Service) attachmentFor(ctx context.Context, userID, id, action string) (*domain.Attachment, error) {
	a, e := s.Stores.Attachments.Get(ctx, id)
	if e != nil {
		if auditErr := s.attachmentFailure(ctx, userID, "", action, id, "not_found"); auditErr != nil {
			return nil, auditErr
		}
		return nil, e
	}
	target := AttachmentTarget{Type: a.TargetType, ID: a.TargetID, BillingAt: a.BillingAt}
	resolved, base, e := s.attachmentTarget(ctx, userID, target)
	if e != nil {
		if auditErr := s.attachmentFailure(ctx, userID, a.GroupID, action, id, "permission_denied"); auditErr != nil {
			return nil, auditErr
		}
		return nil, e
	}
	if resolved.GroupID != a.GroupID || resolved.OwnerID != a.OwnerID {
		if auditErr := s.attachmentFailure(ctx, userID, a.GroupID, action, id, "scope_mismatch"); auditErr != nil {
			return nil, auditErr
		}
		return nil, domain.ErrForbidden
	}
	if a.GroupID != "" {
		if e = s.groupPermission(ctx, userID, a.GroupID, base); e != nil {
			if auditErr := s.attachmentFailure(ctx, userID, a.GroupID, action, id, "ledger_read_denied"); auditErr != nil {
				return nil, auditErr
			}
			return nil, e
		}
		if e = s.groupPermission(ctx, userID, a.GroupID, "ledger.attachments.read"); e != nil {
			if auditErr := s.attachmentFailure(ctx, userID, a.GroupID, action, id, "permission_denied"); auditErr != nil {
				return nil, auditErr
			}
			return nil, e
		}
	}
	return a, nil
}
func (s *Service) ReadAttachment(ctx context.Context, userID, id string) (AttachmentContent, error) {
	a, e := s.attachmentFor(ctx, userID, id, "attachment.downloaded")
	if e != nil {
		return AttachmentContent{}, e
	}
	data, e := s.Stores.Attachments.Read(ctx, id)
	if e != nil {
		if auditErr := s.attachmentFailure(ctx, userID, a.GroupID, "attachment.downloaded", id, "storage_read_failed"); auditErr != nil {
			return AttachmentContent{}, auditErr
		}
		return AttachmentContent{}, e
	}
	if e = s.attachmentSuccess(ctx, userID, a.GroupID, "attachment.downloaded", id, encodeAuditSummary(map[string]any{"target_type": a.TargetType, "size": a.Size}, nil)); e != nil {
		return AttachmentContent{}, e
	}
	return AttachmentContent{Data: data, FileName: a.FileName, MediaType: a.MediaType}, nil
}
func (s *Service) DeleteAttachment(ctx context.Context, userID, id string) error {
	a, e := s.attachmentFor(ctx, userID, id, "attachment.deleted")
	if e != nil {
		return e
	}
	if a.GroupID != "" {
		if e = s.attachmentHistoricalWrite(ctx, userID, *a); e != nil {
			if auditErr := s.attachmentFailure(ctx, userID, a.GroupID, "attachment.deleted", id, "permission_denied"); auditErr != nil {
				return auditErr
			}
			return e
		}
		if e = s.groupPermission(ctx, userID, a.GroupID, "ledger.attachments.delete"); e != nil {
			if auditErr := s.attachmentFailure(ctx, userID, a.GroupID, "attachment.deleted", id, "permission_denied"); auditErr != nil {
				return auditErr
			}
			return e
		}
	}
	e = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if err := s.Stores.Attachments.Delete(tx, id); err != nil {
			return err
		}
		return s.attachmentSuccess(tx, userID, a.GroupID, "attachment.deleted", id, encodeAuditSummary(map[string]any{"target_type": a.TargetType, "size": a.Size}, nil))
	})
	if e != nil {
		if auditErr := s.attachmentFailure(ctx, userID, a.GroupID, "attachment.deleted", id, "storage_delete_failed"); auditErr != nil {
			return auditErr
		}
	}
	return e
}

func (s *Service) AttachmentRenameRequestFailure(ctx context.Context, actor, id string) error {
	a, err := s.attachmentFor(ctx, actor, id, "attachment.renamed")
	if err != nil {
		return err
	}
	if err = s.attachmentFailure(ctx, actor, a.GroupID, "attachment.renamed", id, "invalid_request"); err != nil {
		return err
	}
	return domain.ErrInvalid
}

func validAttachmentName(name, original string) bool {
	ext := filepath.Ext(original)
	if name == "" || utf8.RuneCountInString(name) > 255 || strings.ContainsAny(name, "/\\") || strings.IndexFunc(name, unicode.IsControl) >= 0 || filepath.Ext(name) != ext {
		return false
	}
	return strings.TrimSpace(strings.TrimSuffix(name, ext)) != ""
}

func (s *Service) RenameAttachment(ctx context.Context, actor, id, name string) (*domain.Attachment, error) {
	a, err := s.attachmentFor(ctx, actor, id, "attachment.renamed")
	if err != nil {
		return nil, err
	}
	fail := func(cause error, reason string) (*domain.Attachment, error) {
		if auditErr := s.attachmentFailure(ctx, actor, a.GroupID, "attachment.renamed", id, reason); auditErr != nil {
			return nil, errors.Join(cause, auditErr)
		}
		return nil, cause
	}
	if a.GroupID != "" {
		if err = s.groupPermission(ctx, actor, a.GroupID, "ledger.attachments.upload"); err != nil {
			return fail(err, "permission_denied")
		}
		if err = s.attachmentHistoricalWrite(ctx, actor, *a); err != nil {
			return fail(err, "permission_denied")
		}
	}
	// Reject controls before trimming so a newline cannot be hidden by normalization.
	if strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return fail(domain.ErrAttachmentName, "invalid_name")
	}
	name = strings.TrimSpace(name)
	if !validAttachmentName(name, a.FileName) {
		return fail(domain.ErrAttachmentName, "invalid_name")
	}
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		current, readErr := s.Stores.Attachments.Get(tx, id)
		if readErr != nil {
			return readErr
		}
		if !validAttachmentName(name, current.FileName) {
			return domain.ErrAttachmentName
		}
		if writeErr := s.Stores.Attachments.Rename(tx, id, name); writeErr != nil {
			return fmt.Errorf("%w: %w", domain.ErrAttachmentStorage, writeErr)
		}
		var changes changeSet
		changes.addString("file_name", current.FileName, name)
		return s.attachmentSuccess(tx, actor, a.GroupID, "attachment.renamed", id, encodeAuditSummary(map[string]any{"target_type": a.TargetType}, changes))
	})
	if err != nil {
		return fail(err, "rename_write_failed")
	}
	a.FileName = name
	return a, nil
}

// Changing evidence attached to a historical record uses the same group cutoff.
func (s *Service) attachmentHistoricalWrite(ctx context.Context, actor string, a domain.Attachment) error {
	if a.GroupID == "" {
		return nil
	}
	var at time.Time
	switch a.TargetType {
	case "expense":
		v, err := s.Stores.Expenses.Get(ctx, a.TargetID)
		if err != nil {
			return err
		}
		at = v.IncurredOn
	case "income":
		v, err := s.Stores.Incomes.Get(ctx, a.TargetID)
		if err != nil {
			return err
		}
		at = v.ReceivedOn
	case "settlement":
		v, err := s.Stores.Settlements.Get(ctx, a.TargetID)
		if err != nil {
			return err
		}
		at = v.SettledOn
	case "subscription_period":
		at = a.BillingAt
	default:
		return nil
	}
	start, _, err := s.monthRange(ctx, actor, a.GroupID, "")
	if err != nil {
		return err
	}
	if at.Before(start) {
		return s.groupPermission(ctx, actor, a.GroupID, "ledger.records.historical_write")
	}
	return nil
}
