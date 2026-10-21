package application

import (
	"context"

	"subflow/internal/domain"
)

// CreateOwnershipTransfer starts an invite-and-accept handoff of a group's
// single owner slot. Only the current owner may propose a target, the
// target must already be a real (non-placeholder) member, and a group can
// have at most one pending transfer outstanding at a time.
func (s *Service) CreateOwnershipTransfer(ctx context.Context, userID, groupID, toUserID string) (*domain.OwnershipTransfer, error) {
	group, err := s.Stores.Groups.Get(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group.OwnerID != userID {
		s.audit(ctx, userID, groupID, "ownership_transfer.created", "group", groupID, "failure", encodeAuditSummary(map[string]any{"to_user_id": toUserID}, nil))
		return nil, domain.ErrForbidden
	}
	if toUserID == "" || toUserID == userID {
		return nil, domain.ErrInvalid
	}
	if _, err = s.Stores.Memberships.GetRole(ctx, groupID, toUserID); err != nil {
		return nil, domain.ErrInvalid
	}
	target, err := s.Stores.Users.Get(ctx, toUserID)
	if err != nil {
		return nil, err
	}
	if target.Placeholder {
		return nil, domain.ErrInvalid
	}
	if existing, findErr := s.Stores.OwnershipTransfers.FindPending(ctx, groupID); findErr == nil && existing != nil {
		return nil, domain.ErrConflict
	}
	transfer := &domain.OwnershipTransfer{GroupID: groupID, FromUserID: userID, ToUserID: toUserID, Status: domain.OwnershipTransferPending}
	var notification *domain.Notification
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if err := s.Stores.OwnershipTransfers.Create(tx, transfer); err != nil {
			return err
		}
		var err error
		notification, err = s.createNotification(tx, toUserID, domain.NotificationCategoryTask, "ownership_transfer:"+transfer.ID, "ownership_transfer", groupID, transfer.ID)
		if err != nil {
			return err
		}
		return s.audit(tx, userID, groupID, "ownership_transfer.created", "ownership_transfer", transfer.ID, "success", encodeAuditSummary(map[string]any{"to_user_id": toUserID}, nil))
	})
	if err != nil {
		if auditErr := s.audit(ctx, userID, groupID, "ownership_transfer.created", "ownership_transfer", transfer.ID, "failure"); auditErr != nil {
			return nil, domain.ErrAuditUnavailable
		}
		return nil, err
	}
	s.publishNotification(ctx, notification, "created")
	return transfer, nil
}

// RespondOwnershipTransfer lets only the proposed target accept or decline.
// Accepting moves Group.OwnerID and swaps both memberships' RBAC role in one
// transaction, so the enum-based CanManageGroup gate and the permission-based
// GroupPermissions resolution agree on who the owner is throughout.
func (s *Service) RespondOwnershipTransfer(ctx context.Context, userID, id string, accept bool) (*domain.OwnershipTransfer, error) {
	transfer, err := s.Stores.OwnershipTransfers.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if transfer.Status != domain.OwnershipTransferPending {
		return nil, domain.ErrConflict
	}
	if transfer.ToUserID != userID {
		s.audit(ctx, userID, transfer.GroupID, "ownership_transfer.responded", "ownership_transfer", id, "failure")
		return nil, domain.ErrForbidden
	}
	if !accept {
		transfer.Status = domain.OwnershipTransferDeclined
		err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
			if err := s.Stores.OwnershipTransfers.Update(tx, transfer); err != nil {
				return err
			}
			if err := s.Stores.Notifications.SetResourceStatus(tx, userID, transfer.ID, domain.NotificationStatusResolved, "declined", s.now()); err != nil {
				return err
			}
			if err := s.Stores.Notifications.MarkReadForResource(tx, userID, transfer.ID, s.now()); err != nil {
				return err
			}
			return s.audit(tx, userID, transfer.GroupID, "ownership_transfer.declined", "ownership_transfer", id, "success")
		})
		if err != nil {
			if auditErr := s.audit(ctx, userID, transfer.GroupID, "ownership_transfer.declined", "ownership_transfer", id, "failure"); auditErr != nil {
				return nil, domain.ErrAuditUnavailable
			}
			return nil, err
		}
		s.publishNotification(ctx, &domain.Notification{ID: transfer.ID, UserID: userID, GroupID: transfer.GroupID}, "updated")
		return transfer, nil
	}
	roles, err := s.Stores.Roles.List(ctx, "group", transfer.GroupID)
	if err != nil {
		return nil, err
	}
	var ownerRoleID, memberRoleID string
	for _, role := range roles {
		switch role.Key {
		case "owner":
			ownerRoleID = role.ID
		case "member":
			memberRoleID = role.ID
		}
	}
	if ownerRoleID == "" || memberRoleID == "" {
		return nil, domain.ErrConflict
	}
	if err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		group, groupErr := s.Stores.Groups.Get(tx, transfer.GroupID)
		if groupErr != nil {
			return groupErr
		}
		if group.OwnerID != transfer.FromUserID {
			// The owner changed since this transfer was proposed (e.g. a
			// second, since-cancelled transfer already moved it) — refuse
			// rather than silently overwriting whoever owns it now.
			return domain.ErrConflict
		}
		group.OwnerID = transfer.ToUserID
		if updateErr := s.Stores.Groups.Update(tx, group); updateErr != nil {
			return updateErr
		}
		if updateErr := s.Stores.Memberships.UpdateRole(tx, transfer.GroupID, transfer.FromUserID, memberRoleID); updateErr != nil {
			return updateErr
		}
		if updateErr := s.Stores.Memberships.UpdateRole(tx, transfer.GroupID, transfer.ToUserID, ownerRoleID); updateErr != nil {
			return updateErr
		}
		transfer.Status = domain.OwnershipTransferAccepted
		if err := s.Stores.OwnershipTransfers.Update(tx, transfer); err != nil {
			return err
		}
		if err := s.Stores.Notifications.SetResourceStatus(tx, userID, transfer.ID, domain.NotificationStatusResolved, "accepted", s.now()); err != nil {
			return err
		}
		if err := s.Stores.Notifications.MarkReadForResource(tx, userID, transfer.ID, s.now()); err != nil {
			return err
		}
		return s.audit(tx, userID, transfer.GroupID, "ownership_transfer.accepted", "ownership_transfer", id, "success", encodeAuditSummary(map[string]any{"from_user_id": transfer.FromUserID, "to_user_id": transfer.ToUserID}, nil))
	}); err != nil {
		if auditErr := s.audit(ctx, userID, transfer.GroupID, "ownership_transfer.accepted", "ownership_transfer", id, "failure", encodeAuditSummary(map[string]any{"from_user_id": transfer.FromUserID, "to_user_id": transfer.ToUserID}, nil)); auditErr != nil {
			return nil, domain.ErrAuditUnavailable
		}
		return nil, err
	}
	s.publishNotification(ctx, &domain.Notification{ID: transfer.ID, UserID: transfer.ToUserID, GroupID: transfer.GroupID}, "updated")
	return transfer, nil
}

// CancelOwnershipTransfer lets only the initiating (current) owner withdraw
// a still-pending transfer.
func (s *Service) CancelOwnershipTransfer(ctx context.Context, userID, id string) error {
	transfer, err := s.Stores.OwnershipTransfers.Get(ctx, id)
	if err != nil {
		return err
	}
	if transfer.Status != domain.OwnershipTransferPending {
		return domain.ErrConflict
	}
	if transfer.FromUserID != userID {
		s.audit(ctx, userID, transfer.GroupID, "ownership_transfer.cancelled", "ownership_transfer", id, "failure")
		return domain.ErrForbidden
	}
	transfer.Status = domain.OwnershipTransferCancelled
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if err := s.Stores.OwnershipTransfers.Update(tx, transfer); err != nil {
			return err
		}
		if err := s.Stores.Notifications.SetResourceStatus(tx, transfer.ToUserID, transfer.ID, domain.NotificationStatusInvalid, "cancelled", s.now()); err != nil {
			return err
		}
		return s.audit(tx, userID, transfer.GroupID, "ownership_transfer.cancelled", "ownership_transfer", id, "success")
	})
	if err != nil {
		if auditErr := s.audit(ctx, userID, transfer.GroupID, "ownership_transfer.cancelled", "ownership_transfer", id, "failure"); auditErr != nil {
			return domain.ErrAuditUnavailable
		}
		return err
	}
	s.publishNotification(ctx, &domain.Notification{ID: transfer.ID, UserID: transfer.ToUserID, GroupID: transfer.GroupID}, "updated")
	return nil
}

// PendingOwnershipTransfer returns the group's outstanding transfer, if any,
// for the owner-transfer UI to show its status. Any current member may view
// it; only the initiator/target act on it, which the other methods enforce.
func (s *Service) PendingOwnershipTransfer(ctx context.Context, userID, groupID string) (*domain.OwnershipTransfer, error) {
	if err := s.groupPermission(ctx, userID, groupID, "group.view"); err != nil {
		return nil, err
	}
	return s.Stores.OwnershipTransfers.FindPending(ctx, groupID)
}
