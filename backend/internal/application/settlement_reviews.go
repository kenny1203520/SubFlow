package application

import (
	"context"
	"errors"
	"strings"

	"subflow/internal/domain"
)

func (s *Service) notifySettlementReview(ctx context.Context, userID string, settlement domain.Settlement, reviewID, kind string) error {
	if s.Stores.Notifications == nil || userID == "" {
		return nil
	}
	return s.Stores.Notifications.Create(ctx, &domain.Notification{
		UserID: userID, Type: kind, Category: domain.NotificationCategoryTask,
		EventKey: "settlement-review:" + reviewID + ":" + kind,
		GroupID:  settlement.GroupID, ResourceID: settlement.ID,
		Status: domain.NotificationStatusPending,
	})
}

func (s *Service) ListSettlementReviews(ctx context.Context, userID, groupID string) ([]domain.SettlementReview, error) {
	if err := s.groupPermission(ctx, userID, groupID, "ledger.settlements.read"); err != nil {
		return nil, err
	}
	if s.Stores.SettlementReviews == nil {
		return []domain.SettlementReview{}, nil
	}
	items, err := s.Stores.SettlementReviews.ListPending(ctx, groupID)
	if err != nil {
		return nil, err
	}
	manageErr := s.groupPermission(ctx, userID, groupID, "ledger.settlements.manage")
	if manageErr != nil && !errors.Is(manageErr, domain.ErrForbidden) {
		return nil, manageErr
	}
	canManage := manageErr == nil
	groupName := ""
	if group, groupErr := s.Stores.Groups.Get(ctx, groupID); groupErr == nil {
		groupName = group.Name
	}
	visible := make([]domain.SettlementReview, 0, len(items))
	for _, item := range items {
		item.CanManage = canManage
		item.CanRespond = item.Proposal.ToUserID == userID || canManage
		if item.Proposal.ToUserID == userID || item.SubmittedBy == userID || canManage {
			item.GroupName = groupName
			if from, userErr := s.Stores.Users.Get(ctx, item.Proposal.FromUserID); userErr == nil {
				item.FromName = from.Name
				if item.FromName == "" {
					item.FromName = from.Email
				}
			}
			if to, userErr := s.Stores.Users.Get(ctx, item.Proposal.ToUserID); userErr == nil {
				item.ToName = to.Name
				if item.ToName == "" {
					item.ToName = to.Email
				}
			}
			visible = append(visible, item)
		}
	}
	return visible, nil
}

func (s *Service) RequestSettlementVoid(ctx context.Context, userID, settlementID, reason string) (*domain.SettlementReview, error) {
	settlement, err := s.Stores.Settlements.Get(ctx, settlementID)
	if err != nil {
		return nil, s.settlementReviewFailure(ctx, userID, "", settlementID, "settlement.void_requested", "settlement_not_found", err)
	}
	if err = s.groupPermission(ctx, userID, settlement.GroupID, "ledger.settlements.delete"); err != nil {
		return nil, s.settlementReviewFailure(ctx, userID, settlement.GroupID, settlementID, "settlement.void_requested", "missing_delete_permission", err)
	}
	if settlement.CreatedBy != userID {
		manageErr := s.groupPermission(ctx, userID, settlement.GroupID, "ledger.settlements.manage")
		if manageErr != nil && !errors.Is(manageErr, domain.ErrForbidden) {
			return nil, s.settlementReviewFailure(ctx, userID, settlement.GroupID, settlementID, "settlement.void_requested", "manager_permission_unavailable", manageErr)
		}
		if manageErr != nil {
			return nil, s.settlementReviewFailure(ctx, userID, settlement.GroupID, settlementID, "settlement.void_requested", "not_creator_or_manager", domain.ErrForbidden)
		}
	}
	if !settlement.ApprovalRequired || settlement.Status != "confirmed" {
		return nil, s.settlementReviewFailure(ctx, userID, settlement.GroupID, settlementID, "settlement.void_requested", "settlement_not_approval_managed", domain.ErrConflict)
	}
	if s.Stores.SettlementReviews == nil {
		return nil, s.settlementReviewFailure(ctx, userID, settlement.GroupID, settlementID, "settlement.void_requested", "review_store_unavailable", domain.ErrAuditUnavailable)
	}
	if _, pendingErr := s.Stores.SettlementReviews.FindPending(ctx, settlementID); pendingErr == nil {
		return nil, s.settlementReviewFailure(ctx, userID, settlement.GroupID, settlementID, "settlement.void_requested", "review_already_pending", domain.ErrConflict)
	} else if pendingErr != domain.ErrNotFound {
		return nil, s.settlementReviewFailure(ctx, userID, settlement.GroupID, settlementID, "settlement.void_requested", "pending_review_lookup_failed", pendingErr)
	}
	value := domain.SettlementReview{GroupID: settlement.GroupID, SettlementID: settlementID, Action: "void", Proposal: *settlement, SubmittedBy: userID, Status: "pending", ExpectedVersion: settlement.Version, Reason: strings.TrimSpace(reason)}
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if createErr := s.Stores.SettlementReviews.Create(tx, &value); createErr != nil {
			return createErr
		}
		if notifyErr := s.notifySettlementReview(tx, settlement.ToUserID, *settlement, value.ID, "settlement_review_pending"); notifyErr != nil {
			return notifyErr
		}
		return s.audit(tx, userID, settlement.GroupID, "settlement.void_requested", "settlement", settlementID, "success", encodeAuditSummary(map[string]any{"reason_provided": value.Reason != ""}, nil))
	})
	if err != nil {
		return nil, s.settlementReviewFailure(ctx, userID, settlement.GroupID, settlementID, "settlement.void_requested", "transaction_failed", err)
	}
	return &value, nil
}

func (s *Service) RespondSettlementReview(ctx context.Context, userID, reviewID, decision, reason string) (*domain.SettlementReview, error) {
	if decision != "accept" && decision != "reject" {
		return nil, s.settlementReviewFailure(ctx, userID, "", "", "settlement.review.respond", "invalid_decision", domain.ErrInvalid)
	}
	review, err := s.Stores.SettlementReviews.Get(ctx, reviewID)
	if err != nil {
		return nil, s.settlementReviewFailure(ctx, userID, "", reviewID, "settlement.review.respond", "review_not_found", err)
	}
	settlement, err := s.Stores.Settlements.Get(ctx, review.SettlementID)
	if err != nil {
		return nil, s.settlementReviewFailure(ctx, userID, review.GroupID, review.SettlementID, "settlement.review.respond", "settlement_not_found", err)
	}
	action := "settlement.review." + decision
	if review.Status != "pending" || settlement.Version != review.ExpectedVersion {
		return nil, s.settlementReviewFailure(ctx, userID, review.GroupID, settlement.ID, action, "review_stale_or_closed", domain.ErrConflict)
	}
	manageErr := s.groupPermission(ctx, userID, review.GroupID, "ledger.settlements.manage")
	if manageErr != nil && !errors.Is(manageErr, domain.ErrForbidden) {
		return nil, s.settlementReviewFailure(ctx, userID, review.GroupID, review.SettlementID, action, "manager_permission_unavailable", manageErr)
	}
	manager := manageErr == nil
	recipient := userID == review.Proposal.ToUserID
	if !recipient && !manager {
		return nil, s.settlementReviewFailure(ctx, userID, review.GroupID, settlement.ID, action, "not_recipient_or_manager", domain.ErrForbidden)
	}
	reason = strings.TrimSpace(reason)
	if decision == "reject" && reason == "" {
		return nil, s.settlementReviewFailure(ctx, userID, review.GroupID, settlement.ID, action, "rejection_reason_required", domain.ErrInvalid)
	}
	if manager && !recipient && reason == "" {
		return nil, s.settlementReviewFailure(ctx, userID, review.GroupID, settlement.ID, action, "manager_override_reason_required", domain.ErrInvalid)
	}
	now := s.Now().UTC()
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		currentReview, getErr := s.Stores.SettlementReviews.Get(tx, reviewID)
		if getErr != nil {
			return getErr
		}
		currentSettlement, getErr := s.Stores.Settlements.Get(tx, review.SettlementID)
		if getErr != nil {
			return getErr
		}
		if currentReview.Status != "pending" || currentSettlement.Version != currentReview.ExpectedVersion {
			return domain.ErrConflict
		}
		if decision == "accept" {
			switch currentReview.Action {
			case "create":
				currentSettlement.Status = "confirmed"
			case "change":
				proposal := currentReview.Proposal
				currentSettlement.AmountMinor = proposal.AmountMinor
				currentSettlement.SettledOn = proposal.SettledOn
				currentSettlement.Notes = proposal.Notes
				currentSettlement.BaseAmountMinor = proposal.AmountMinor
				currentSettlement.ExchangeRateDate = proposal.SettledOn
				currentSettlement.Status = "confirmed"
			case "void":
				currentSettlement.Status = "voided"
			default:
				return domain.ErrInvalid
			}
			currentSettlement.Version++
		} else if currentReview.Action == "create" {
			currentSettlement.Status = "rejected"
			currentSettlement.Version++
		}
		currentReview.Status = map[bool]string{true: "accepted", false: "rejected"}[decision == "accept"]
		currentReview.DecidedBy = userID
		currentReview.Reason = reason
		if updateErr := s.Stores.Settlements.Update(tx, currentSettlement); updateErr != nil {
			return updateErr
		}
		if updateErr := s.Stores.SettlementReviews.Update(tx, currentReview); updateErr != nil {
			return updateErr
		}
		for _, participant := range []string{currentSettlement.FromUserID, currentSettlement.ToUserID} {
			if updateErr := s.Stores.Notifications.SetResourceStatus(tx, participant, currentSettlement.ID, domain.NotificationStatusResolved, currentReview.Status, now); updateErr != nil {
				return updateErr
			}
		}
		return s.audit(tx, userID, currentReview.GroupID, action, "settlement", currentSettlement.ID, "success", encodeAuditSummary(map[string]any{"review_id": reviewID, "action": currentReview.Action, "self_approval": userID == currentSettlement.FromUserID && manager, "reason_provided": reason != ""}, nil))
	})
	if err != nil {
		return nil, s.settlementReviewFailure(ctx, userID, review.GroupID, settlement.ID, action, "transaction_failed", err)
	}
	updated, err := s.Stores.SettlementReviews.Get(ctx, reviewID)
	return updated, err
}

func (s *Service) WithdrawSettlementReview(ctx context.Context, userID, reviewID string) error {
	review, err := s.Stores.SettlementReviews.Get(ctx, reviewID)
	if err != nil {
		return s.settlementReviewFailure(ctx, userID, "", reviewID, "settlement.review.withdraw", "review_not_found", err)
	}
	settlement, err := s.Stores.Settlements.Get(ctx, review.SettlementID)
	if err != nil {
		return s.settlementReviewFailure(ctx, userID, review.GroupID, review.SettlementID, "settlement.review.withdraw", "settlement_not_found", err)
	}
	if review.SubmittedBy != userID || review.Status != "pending" {
		return s.settlementReviewFailure(ctx, userID, review.GroupID, settlement.ID, "settlement.review.withdraw", "not_submitter_or_closed", domain.ErrForbidden)
	}
	now := s.Now().UTC()
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		current, getErr := s.Stores.SettlementReviews.Get(tx, reviewID)
		if getErr != nil {
			return getErr
		}
		if current.Status != "pending" {
			return domain.ErrConflict
		}
		current.Status = "withdrawn"
		current.DecidedBy = userID
		if current.Action == "create" {
			settlement.Status = "withdrawn"
			settlement.Version++
			if updateErr := s.Stores.Settlements.Update(tx, settlement); updateErr != nil {
				return updateErr
			}
		}
		if updateErr := s.Stores.SettlementReviews.Update(tx, current); updateErr != nil {
			return updateErr
		}
		for _, participant := range []string{settlement.FromUserID, settlement.ToUserID} {
			if updateErr := s.Stores.Notifications.SetResourceStatus(tx, participant, settlement.ID, domain.NotificationStatusResolved, "withdrawn", now); updateErr != nil {
				return updateErr
			}
		}
		return s.audit(tx, userID, current.GroupID, "settlement.review.withdraw", "settlement", settlement.ID, "success", encodeAuditSummary(map[string]any{"review_id": reviewID, "action": current.Action}, nil))
	})
	if err != nil {
		return s.settlementReviewFailure(ctx, userID, review.GroupID, settlement.ID, "settlement.review.withdraw", "transaction_failed", err)
	}
	return nil
}

func (s *Service) settlementReviewFailure(ctx context.Context, userID, groupID, settlementID, action, reason string, cause error) error {
	if auditErr := s.audit(ctx, userID, groupID, action, "settlement", settlementID, "failure", encodeAuditSummary(map[string]any{"reason": reason}, nil)); auditErr != nil {
		return auditErr
	}
	return cause
}
