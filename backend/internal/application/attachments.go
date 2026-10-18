package application

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

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
	return s.audit(ctx, userID, groupID, action, "attachment", id, "failure", encodeAuditSummary(map[string]any{"reason": reason}, nil))
}

func (s *Service) RecordAttachmentFailure(ctx context.Context, userID, action, reason string) error {
	return s.attachmentFailure(ctx, userID, "", action, "", reason)
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
		return page, e
	}
	if a.GroupID == "" {
		page.Access = domain.AttachmentAccess{CanRead: true, CanUpload: true, CanDelete: true}
	} else {
		perms, err := s.GroupPermissions(ctx, userID, a.GroupID)
		if err != nil {
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
	}
	if !page.Access.CanRead {
		if auditErr := s.attachmentFailure(ctx, userID, a.GroupID, "attachment.listed", a.TargetID, "permission_denied"); auditErr != nil {
			return domain.AttachmentPage{}, auditErr
		}
		return page, domain.ErrForbidden
	}
	page.Items, e = s.Stores.Attachments.List(ctx, a.TargetType, a.TargetID, a.BillingAt)
	if e != nil {
		return page, e
	}
	if e = s.audit(ctx, userID, a.GroupID, "attachment.listed", "attachment", a.TargetID, "success", encodeAuditSummary(map[string]any{"target_type": a.TargetType, "count": len(page.Items)}, nil)); e != nil {
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
		if len(data) >= 12 && string(data[4:8]) == "ftyp" && (bytes.Contains(data[:32], []byte("heic")) || bytes.Contains(data[:32], []byte("heix")) || bytes.Contains(data[:32], []byte("mif1"))) {
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
	if target.RequestID == "" || len(target.RequestID) > 64 {
		return nil, domain.ErrInvalid
	}
	fail := func(err error) (*domain.Attachment, error) {
		if auditErr := s.audit(ctx, userID, a.GroupID, "attachment.uploaded", "attachment", a.TargetID, "failure", encodeAuditSummary(map[string]any{"reason": "upload_rejected", "target_type": a.TargetType}, nil)); auditErr != nil {
			return nil, auditErr
		}
		return nil, err
	}
	if a.GroupID != "" {
		if e = s.groupPermission(ctx, userID, a.GroupID, "ledger.attachments.read"); e != nil {
			return fail(e)
		}
		if e = s.groupPermission(ctx, userID, a.GroupID, "ledger.attachments.upload"); e != nil {
			return fail(e)
		}
	}
	if old, err := s.Stores.Attachments.FindByRequest(ctx, userID, target.RequestID); err == nil {
		if old.TargetType != a.TargetType || old.TargetID != a.TargetID || !old.BillingAt.Equal(a.BillingAt) {
			return fail(domain.ErrConflict)
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
	if !ok || int64(len(data)) > limits.MaxSize {
		return fail(domain.ErrInvalid)
	}
	name = strings.ReplaceAll(name, "\\", "/")
	a.FileName = filepath.Base(name)
	a.MediaType = media
	e = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		items, listErr := s.Stores.Attachments.List(tx, a.TargetType, a.TargetID, a.BillingAt)
		if listErr != nil {
			return listErr
		}
		if len(items) >= limits.MaxCount {
			return domain.ErrConflict
		}
		if err := s.Stores.Attachments.Create(tx, &a, data); err != nil {
			return err
		}
		return s.audit(tx, userID, a.GroupID, "attachment.uploaded", "attachment", a.TargetID, "success", encodeAuditSummary(map[string]any{"target_type": a.TargetType, "size": len(data)}, nil))
	})
	if e != nil {
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
	if e = s.audit(ctx, userID, a.GroupID, "attachment.downloaded", "attachment", id, "success", encodeAuditSummary(map[string]any{"target_type": a.TargetType, "size": a.Size}, nil)); e != nil {
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
		return s.audit(tx, userID, a.GroupID, "attachment.deleted", "attachment", id, "success", encodeAuditSummary(map[string]any{"target_type": a.TargetType, "size": a.Size}, nil))
	})
	if e != nil {
		if auditErr := s.attachmentFailure(ctx, userID, a.GroupID, "attachment.deleted", id, "storage_delete_failed"); auditErr != nil {
			return auditErr
		}
	}
	return e
}
