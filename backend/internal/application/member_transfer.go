package application

import (
	"context"

	"subflow/internal/domain"
)

// repointUserReferences moves every group-scoped reference -- expenses,
// expense_splits, subscriptions and every one of their revisions,
// settlements, notifications, group-scope categories -- from fromUserID to
// toUserID within groupID, then, if removeFromMembership, removes
// fromUserID's membership in that group. Used by both the temp-member bind
// path (CollaborationService.accept) and the member-identity-transfer
// feature below (RespondMemberTransfer), so the FK-rewriting logic lives in
// exactly one place. Must run inside an already-open
// Stores.Transactions.Within(tx) block -- it never opens its own transaction.
//
// audit_logs and personal-scope fields (expenses.owner, subscriptions.owner,
// categories.owner) are intentionally left untouched: an audit entry's
// summary is a free-text JSON blob, not a queryable/rewritable relation, and
// personal data sits outside the group-scoped boundary this operation works
// within.
func (s *Service) repointUserReferences(ctx context.Context, groupID, fromUserID, toUserID string, removeFromMembership bool) error {
	if err := s.Stores.Expenses.ReassignUser(ctx, groupID, fromUserID, toUserID); err != nil {
		return err
	}
	if err := s.Stores.Subscriptions.ReassignUser(ctx, groupID, fromUserID, toUserID); err != nil {
		return err
	}
	if err := s.Stores.Settlements.ReassignUser(ctx, groupID, fromUserID, toUserID); err != nil {
		return err
	}
	if err := s.Stores.Notifications.ReassignUser(ctx, groupID, fromUserID, toUserID); err != nil {
		return err
	}
	if err := s.Stores.Categories.ReassignUser(ctx, groupID, fromUserID, toUserID); err != nil {
		return err
	}
	if removeFromMembership {
		if err := s.Stores.Memberships.Delete(ctx, groupID, fromUserID); err != nil {
			return err
		}
	}
	return nil
}

// CreateMemberTransfer proposes handing fromUserID's entire group-scoped
// history over to toUserID -- the same repointing a temp-member bind does,
// just between two already-real members and requiring toUserID's consent
// (see RespondMemberTransfer). Gated on group.members.manage, the same
// permission RemoveMember/CreateTempMember use, since this is a member
// management action; the group owner can never be fromUserID here (use
// CreateOwnershipTransfer for that instead), matching RemoveMember's own
// owner guard.
func (s *Service) CreateMemberTransfer(ctx context.Context, userID, groupID, fromUserID, toUserID string) (*domain.MemberTransfer, error) {
	if err := s.groupPermission(ctx, userID, groupID, "group.members.manage"); err != nil {
		s.audit(ctx, userID, groupID, "member_transfer.created", "group", groupID, "failure", encodeAuditSummary(map[string]any{"from_user_id": fromUserID, "to_user_id": toUserID}, nil))
		return nil, err
	}
	if fromUserID == "" || toUserID == "" || fromUserID == toUserID {
		return nil, domain.ErrInvalid
	}
	fromRole, err := s.Stores.Memberships.GetRole(ctx, groupID, fromUserID)
	if err != nil {
		return nil, domain.ErrInvalid
	}
	if fromRole == domain.RoleOwner {
		return nil, domain.ErrForbidden
	}
	if _, err = s.Stores.Memberships.GetRole(ctx, groupID, toUserID); err != nil {
		return nil, domain.ErrInvalid
	}
	from, err := s.Stores.Users.Get(ctx, fromUserID)
	if err != nil {
		return nil, err
	}
	to, err := s.Stores.Users.Get(ctx, toUserID)
	if err != nil {
		return nil, err
	}
	if from.Placeholder || to.Placeholder {
		return nil, domain.ErrInvalid
	}
	if existing, findErr := s.Stores.MemberTransfers.FindPendingByFromUser(ctx, groupID, fromUserID); findErr == nil && existing != nil {
		return nil, domain.ErrConflict
	}
	transfer := &domain.MemberTransfer{GroupID: groupID, FromUserID: fromUserID, ToUserID: toUserID, Status: domain.MemberTransferPending}
	if err = s.Stores.MemberTransfers.Create(ctx, transfer); err != nil {
		return nil, err
	}
	s.audit(ctx, userID, groupID, "member_transfer.created", "member_transfer", transfer.ID, "success", encodeAuditSummary(map[string]any{"from_user_id": fromUserID, "to_user_id": toUserID}, nil))
	return transfer, nil
}

// RespondMemberTransfer lets only the proposed target accept or decline.
// Accepting repoints every reference in one transaction and removes
// fromUserID from the group -- a full handover, same as binding a temp
// member retires the placeholder.
func (s *Service) RespondMemberTransfer(ctx context.Context, userID, id string, accept bool) (*domain.MemberTransfer, error) {
	transfer, err := s.Stores.MemberTransfers.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if transfer.Status != domain.MemberTransferPending {
		return nil, domain.ErrConflict
	}
	if transfer.ToUserID != userID {
		s.audit(ctx, userID, transfer.GroupID, "member_transfer.responded", "member_transfer", id, "failure")
		return nil, domain.ErrForbidden
	}
	if !accept {
		transfer.Status = domain.MemberTransferDeclined
		if err = s.Stores.MemberTransfers.Update(ctx, transfer); err != nil {
			return nil, err
		}
		s.audit(ctx, userID, transfer.GroupID, "member_transfer.declined", "member_transfer", id, "success")
		return transfer, nil
	}
	if err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if _, roleErr := s.Stores.Memberships.GetRole(tx, transfer.GroupID, transfer.FromUserID); roleErr != nil {
			// fromUserID already left the group since this transfer was
			// proposed (e.g. removed by an admin) -- refuse rather than
			// silently repointing nothing.
			return domain.ErrConflict
		}
		if repointErr := s.repointUserReferences(tx, transfer.GroupID, transfer.FromUserID, transfer.ToUserID, true); repointErr != nil {
			return repointErr
		}
		transfer.Status = domain.MemberTransferAccepted
		return s.Stores.MemberTransfers.Update(tx, transfer)
	}); err != nil {
		s.audit(ctx, userID, transfer.GroupID, "member_transfer.accepted", "member_transfer", id, "failure", encodeAuditSummary(map[string]any{"from_user_id": transfer.FromUserID, "to_user_id": transfer.ToUserID}, nil))
		return nil, err
	}
	s.audit(ctx, userID, transfer.GroupID, "member_transfer.accepted", "member_transfer", id, "success", encodeAuditSummary(map[string]any{"from_user_id": transfer.FromUserID, "to_user_id": transfer.ToUserID}, nil))
	return transfer, nil
}

// CancelMemberTransfer lets anyone holding group.members.manage withdraw a
// still-pending transfer -- unlike ownership transfer, the proposer doesn't
// have to be the same actor who's authorized to manage members, since either
// could reasonably change their mind about a pending handover.
func (s *Service) CancelMemberTransfer(ctx context.Context, userID, id string) error {
	transfer, err := s.Stores.MemberTransfers.Get(ctx, id)
	if err != nil {
		return err
	}
	if transfer.Status != domain.MemberTransferPending {
		return domain.ErrConflict
	}
	if err = s.groupPermission(ctx, userID, transfer.GroupID, "group.members.manage"); err != nil {
		s.audit(ctx, userID, transfer.GroupID, "member_transfer.cancelled", "member_transfer", id, "failure")
		return err
	}
	transfer.Status = domain.MemberTransferCancelled
	if err = s.Stores.MemberTransfers.Update(ctx, transfer); err != nil {
		return err
	}
	s.audit(ctx, userID, transfer.GroupID, "member_transfer.cancelled", "member_transfer", id, "success")
	return nil
}

// PendingMemberTransfers lists every transfer still awaiting a response in
// the group, for the members UI to show what's in flight. Any current member
// may view it; CreateMemberTransfer/RespondMemberTransfer/
// CancelMemberTransfer enforce who may act on any given one.
func (s *Service) PendingMemberTransfers(ctx context.Context, userID, groupID string) ([]domain.MemberTransfer, error) {
	if err := s.role(ctx, groupID, userID, false); err != nil {
		return nil, err
	}
	return s.Stores.MemberTransfers.ListPending(ctx, groupID)
}
