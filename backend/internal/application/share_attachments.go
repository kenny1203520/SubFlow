package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"subflow/internal/domain"
)

type shareAttachmentLocator struct {
	ShareID      string    `json:"s"`
	Type         string    `json:"t"`
	TargetID     string    `json:"i"`
	BillingAt    time.Time `json:"b,omitempty"`
	AttachmentID string    `json:"a,omitempty"`
}

func (s *Service) makeShareAttachmentReference(share *domain.Share, locator shareAttachmentLocator) (string, error) {
	locator.ShareID = share.ID
	data, err := json.Marshal(locator)
	if err != nil {
		return "", err
	}
	sealed, err := s.Cipher.Encrypt(string(data))
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString([]byte(sealed)), nil
}

func (s *Service) parseShareAttachmentReference(ref string) (shareAttachmentLocator, error) {
	sealed, err := base64.RawURLEncoding.DecodeString(ref)
	if err != nil {
		return shareAttachmentLocator{}, domain.ErrNotFound
	}
	plain, err := s.Cipher.Decrypt(string(sealed))
	if err != nil {
		return shareAttachmentLocator{}, domain.ErrNotFound
	}
	var locator shareAttachmentLocator
	if json.Unmarshal([]byte(plain), &locator) != nil || locator.ShareID == "" {
		return shareAttachmentLocator{}, domain.ErrNotFound
	}
	return locator, nil
}

func (s *Service) requireShareAttachmentAccess(ctx context.Context, share *domain.Share) error {
	if !share.ShowAttachments || share.AttachmentOwnerID == "" {
		return domain.ErrForbidden
	}
	if share.GroupID == "" {
		if share.OwnerID != share.AttachmentOwnerID {
			return domain.ErrForbidden
		}
		return nil
	}
	if err := s.groupPermission(ctx, share.AttachmentOwnerID, share.GroupID, "ledger.attachments.read"); err != nil {
		share.ShowAttachments, share.AttachmentOwnerID = false, ""
		if auditErr := s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
			if updateErr := s.Stores.Shares.Update(tx, share); updateErr != nil {
				return updateErr
			}
			return s.audit(tx, "", share.GroupID, "share.attachments_revoked", "share", share.ID, "success", encodeAuditSummary(map[string]any{"reason": "authorization_revoked"}, nil))
		}); auditErr != nil {
			return errors.Join(domain.ErrAuditUnavailable, auditErr)
		}
		return domain.ErrForbidden
	}
	return nil
}

func (s *Service) ShareAttachmentList(ctx context.Context, share *domain.Share, ref string) (result []domain.ShareAttachment, retErr error) {
	defer func() {
		if retErr != nil {
			if auditErr := s.audit(ctx, "", share.GroupID, "share.attachment.listed", "share", share.ID, "failure", encodeAuditSummary(map[string]any{"reason": "unavailable"}, nil)); auditErr != nil {
				retErr = errors.Join(domain.ErrAuditUnavailable, retErr, auditErr)
			}
		}
	}()
	locator, err := s.parseShareAttachmentReference(ref)
	if err != nil || locator.ShareID != share.ID || locator.AttachmentID != "" {
		return nil, domain.ErrNotFound
	}
	if err = s.requireShareAttachmentAccess(ctx, share); err != nil {
		return nil, err
	}
	if !shareAttachmentTypeVisible(share, locator.Type) {
		return nil, domain.ErrNotFound
	}
	if err = s.requireShareAttachmentScope(ctx, share, locator.Type); err != nil {
		return nil, err
	}
	start, end, _, err := s.shareRange(ctx, share)
	if err != nil {
		return nil, err
	}
	if locator.Type == "subscription_period" && !inRange(locator.BillingAt, start, end) {
		return nil, domain.ErrNotFound
	}
	items, err := s.Stores.Attachments.List(ctx, locator.Type, locator.TargetID, locator.BillingAt)
	if err != nil {
		return nil, err
	}
	if locator.Type == "subscription" {
		periods, periodErr := s.Stores.Attachments.List(ctx, "subscription_period", locator.TargetID, time.Time{})
		if periodErr != nil {
			return nil, periodErr
		}
		for _, item := range periods {
			if inRange(item.BillingAt, start, end) {
				items = append(items, item)
			}
		}
	}
	result = make([]domain.ShareAttachment, 0, len(items))
	for _, item := range items {
		if item.GroupID != share.GroupID || (share.GroupID == "" && item.OwnerID != share.OwnerID) {
			continue
		}
		itemLocator := locator
		if locator.Type == "subscription" && item.TargetType == "subscription_period" {
			itemLocator.Type, itemLocator.BillingAt = "subscription_period", item.BillingAt
		}
		itemLocator.AttachmentID = item.ID
		sealed, sealErr := s.makeShareAttachmentReference(share, itemLocator)
		if sealErr != nil {
			return nil, sealErr
		}
		result = append(result, domain.ShareAttachment{Reference: sealed, FileName: item.FileName, MediaType: item.MediaType, Size: item.Size, BillingAt: item.BillingAt})
	}
	if err = s.audit(ctx, "", share.GroupID, "share.attachment.listed", "share", share.ID, "success", encodeAuditSummary(map[string]any{"count": len(result)}, nil)); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) ReadShareAttachment(ctx context.Context, share *domain.Share, ref string, download bool) (result AttachmentContent, retErr error) {
	action := "share.attachment.viewed"
	if download {
		action = "share.attachment.downloaded"
	}
	defer func() {
		if retErr != nil {
			if auditErr := s.audit(ctx, "", share.GroupID, action, "share", share.ID, "failure", encodeAuditSummary(map[string]any{"reason": "unavailable"}, nil)); auditErr != nil {
				retErr = errors.Join(domain.ErrAuditUnavailable, retErr, auditErr)
			}
		}
	}()
	locator, err := s.parseShareAttachmentReference(ref)
	if err != nil || locator.ShareID != share.ID || locator.AttachmentID == "" {
		return AttachmentContent{}, domain.ErrNotFound
	}
	if err = s.requireShareAttachmentAccess(ctx, share); err != nil {
		return AttachmentContent{}, err
	}
	if !shareAttachmentTypeVisible(share, locator.Type) {
		return AttachmentContent{}, domain.ErrNotFound
	}
	if err = s.requireShareAttachmentScope(ctx, share, locator.Type); err != nil {
		return AttachmentContent{}, err
	}
	start, end, _, err := s.shareRange(ctx, share)
	if err != nil {
		return AttachmentContent{}, err
	}
	if (locator.Type == "subscription_period" || locator.Type == "subscription") && !locator.BillingAt.IsZero() && !inRange(locator.BillingAt, start, end) {
		return AttachmentContent{}, domain.ErrNotFound
	}
	item, err := s.Stores.Attachments.Get(ctx, locator.AttachmentID)
	expectedType := locator.Type
	if locator.Type == "subscription" && !locator.BillingAt.IsZero() {
		expectedType = "subscription_period"
	}
	if err != nil || item.TargetType != expectedType || item.TargetID != locator.TargetID || !item.BillingAt.Equal(locator.BillingAt) || item.GroupID != share.GroupID || (share.GroupID == "" && item.OwnerID != share.OwnerID) {
		return AttachmentContent{}, domain.ErrNotFound
	}
	data, err := s.Stores.Attachments.Read(ctx, item.ID)
	if err != nil {
		return AttachmentContent{}, err
	}
	if err = s.audit(ctx, "", share.GroupID, action, "share", share.ID, "success", encodeAuditSummary(map[string]any{"targetType": item.TargetType}, nil)); err != nil {
		return AttachmentContent{}, err
	}
	return AttachmentContent{Data: data, FileName: item.FileName, MediaType: item.MediaType}, nil
}

func (s *Service) requireShareAttachmentScope(ctx context.Context, share *domain.Share, typ string) error {
	if share.GroupID == "" {
		return nil
	}
	permission := map[string]string{"expense": "ledger.expenses.read", "subscription_period": "ledger.subscriptions.read", "income": "ledger.incomes.read", "subscription": "ledger.subscriptions.read", "settlement": "ledger.settlements.read"}[typ]
	if permission == "" {
		return domain.ErrNotFound
	}
	if err := s.groupPermission(ctx, share.AttachmentOwnerID, share.GroupID, permission); err != nil {
		share.ShowAttachments, share.AttachmentOwnerID = false, ""
		if auditErr := s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
			if updateErr := s.Stores.Shares.Update(tx, share); updateErr != nil {
				return updateErr
			}
			return s.audit(tx, "", share.GroupID, "share.attachments_revoked", "share", share.ID, "success", encodeAuditSummary(map[string]any{"reason": "record_permission_revoked"}, nil))
		}); auditErr != nil {
			return errors.Join(domain.ErrAuditUnavailable, auditErr)
		}
		return domain.ErrForbidden
	}
	return nil
}

func shareAttachmentTypeVisible(share *domain.Share, typ string) bool {
	switch strings.TrimSpace(typ) {
	case "expense":
		return share.ShowExpenses
	case "subscription_period":
		return share.ShowExpenses || share.ShowSubscriptions
	case "income":
		return share.ShowIncomes
	case "subscription":
		return share.ShowSubscriptions
	case "settlement":
		return share.ShowSettlements
	default:
		return false
	}
}

func (s *Service) AuditShareAttachmentFailure(ctx context.Context, share *domain.Share, action, reason string) error {
	return s.audit(ctx, "", share.GroupID, action, "share", share.ID, "failure", encodeAuditSummary(map[string]any{"reason": reason}, nil))
}

func shareAttachmentCount(ctx context.Context, s *Service, share *domain.Share, typ, id string, billing time.Time) (int, error) {
	if !share.ShowAttachments {
		return 0, nil
	}
	items, err := s.Stores.Attachments.List(ctx, typ, id, billing)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, item := range items {
		if item.GroupID == share.GroupID && (share.GroupID != "" || item.OwnerID == share.OwnerID) {
			count++
		}
	}
	if typ == "subscription" {
		periods, periodErr := s.Stores.Attachments.List(ctx, "subscription_period", id, time.Time{})
		if periodErr != nil {
			return 0, periodErr
		}
		start, end, _, rangeErr := s.shareRange(ctx, share)
		if rangeErr != nil {
			return 0, rangeErr
		}
		for _, item := range periods {
			if inRange(item.BillingAt, start, end) && item.GroupID == share.GroupID && (share.GroupID != "" || item.OwnerID == share.OwnerID) {
				count++
			}
		}
	}
	return count, nil
}

func (s *Service) addShareAttachmentReference(ctx context.Context, share *domain.Share, row map[string]any, typ, id string, billing time.Time) error {
	if !share.ShowAttachments {
		return nil
	}
	count, err := shareAttachmentCount(ctx, s, share, typ, id, billing)
	if err != nil {
		return err
	}
	if count == 0 {
		return nil
	}
	if err := s.requireShareAttachmentScope(ctx, share, typ); err != nil {
		return err
	}
	ref, err := s.makeShareAttachmentReference(share, shareAttachmentLocator{Type: typ, TargetID: id, BillingAt: billing})
	if err != nil {
		return err
	}
	row["attachmentReference"], row["attachmentCount"] = ref, count
	return nil
}
