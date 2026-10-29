package application

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"subflow/internal/domain"
	"subflow/internal/ports"
	"subflow/internal/publicurl"
)

type CollaborationService struct {
	Base                *Service
	Events              ports.EventPublisher
	Mailer              ports.Mailer
	Environment, AppURL string
	ApplicationURL      func() string
}

func (s *CollaborationService) CreateInvitation(ctx context.Context, userID, groupID, email string) (*domain.Invitation, error) {
	return s.CreateInvitationBinding(ctx, userID, groupID, email, "")
}

// CreateInvitationBinding is CreateInvitation with an optional target: when
// targetPlaceholderID is set, accepting this invitation binds that
// placeholder "temp member" (see Service.CreateTempMember) to the accepting
// user instead of only creating a fresh membership — see accept() below.
func (s *CollaborationService) CreateInvitationBinding(ctx context.Context, userID, groupID, email, targetPlaceholderID string) (*domain.Invitation, error) {
	if err := s.Base.groupPermission(ctx, userID, groupID, "group.members.manage"); err != nil {
		return nil, err
	}
	if _, err := s.publicApplicationURL(); err != nil {
		return nil, err
	}
	email = domain.NormalizeEmail(email)
	if email == "" || !strings.Contains(email, "@") {
		return nil, domain.ErrInvalid
	}
	if _, err := s.Base.Stores.Invitations.FindPending(ctx, groupID, email); err == nil {
		return nil, domain.ErrConflict
	}
	if targetPlaceholderID != "" {
		placeholder, err := s.Base.Stores.Users.Get(ctx, targetPlaceholderID)
		if err != nil {
			return nil, err
		}
		if !placeholder.Placeholder || placeholder.LinkedUserID != "" {
			return nil, domain.ErrConflict
		}
		if role, err := s.Base.Stores.Memberships.GetRole(ctx, groupID, targetPlaceholderID); err != nil || role == "" {
			return nil, domain.ErrInvalid
		}
		if _, err := s.Base.Stores.Invitations.FindPendingByTarget(ctx, groupID, targetPlaceholderID); err == nil {
			return nil, domain.ErrConflict
		}
	}
	plain, hash, err := domain.NewInvitationToken()
	if err != nil {
		return nil, err
	}
	inv := &domain.Invitation{GroupID: groupID, Email: email, TokenHash: hash, Status: domain.InvitationPending, InvitedBy: userID, ExpiresAt: s.Base.Now().UTC().Add(7 * 24 * time.Hour), TargetPlaceholderID: targetPlaceholderID}
	notificationRecipientID := ""
	notificationID := ""
	err = s.Base.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if err := s.Base.Stores.Invitations.Create(tx, inv); err != nil {
			return err
		}
		if recipient, findErr := s.Base.Stores.Users.FindByEmail(tx, email); findErr == nil {
			notificationRecipientID = recipient.ID
			note := &domain.Notification{UserID: recipient.ID, Type: "group_invitation", Category: domain.NotificationCategoryTask, EventKey: "group_invitation:" + inv.ID, Title: "group_invitation", GroupID: groupID, ResourceID: inv.ID, Status: domain.NotificationStatusPending}
			if err := s.Base.Stores.Notifications.Create(tx, note); err != nil {
				return err
			}
			notificationID = note.ID
		} else if !errors.Is(findErr, domain.ErrNotFound) {
			return findErr
		}
		return s.Base.audit(tx, userID, groupID, "invitation.created", "invitation", inv.ID, "success", encodeAuditSummary(map[string]any{"email": email}, nil))
	})
	if err != nil {
		if auditErr := s.Base.audit(ctx, userID, groupID, "invitation.created", "invitation", inv.ID, "failure"); auditErr != nil {
			return nil, domain.ErrAuditUnavailable
		}
		return nil, err
	}
	if notificationRecipientID != "" {
		s.publishNotification(ctx, "created", notificationRecipientID, groupID, notificationID)
	}
	return s.deliver(ctx, inv, plain)
}
func (s *CollaborationService) deliver(ctx context.Context, inv *domain.Invitation, plain string) (*domain.Invitation, error) {
	group, err := s.Base.Stores.Groups.Get(ctx, inv.GroupID)
	if err != nil {
		return nil, err
	}
	parsedURL, err := s.publicApplicationURL()
	if err != nil {
		return nil, err
	}
	link := parsedURL.String() + "/invite/" + url.PathEscape(plain)
	if strings.EqualFold(s.Environment, "development") || s.Environment == "" {
		inv.DebugURL = link
	}
	if s.Mailer == nil || !s.Mailer.Configured() {
		if !strings.EqualFold(s.Environment, "development") && s.Environment != "" {
			inv.Status = domain.InvitationDeliveryFailed
			if saveErr := s.Base.Stores.Invitations.Update(ctx, inv); saveErr != nil {
				return nil, saveErr
			}
			if auditErr := s.Base.audit(ctx, inv.InvitedBy, inv.GroupID, "invitation.delivery_failed", "invitation", inv.ID, "failure"); auditErr != nil {
				return nil, domain.ErrAuditUnavailable
			}
			return nil, fmt.Errorf("smtp is required outside development")
		}
		return inv, nil
	}
	inviterName := ""
	if inviter, inviterErr := s.Base.Stores.Users.Get(ctx, inv.InvitedBy); inviterErr == nil {
		inviterName = inviter.Name
	}
	if err = s.Mailer.SendInvitation(ctx, *inv, *group, inviterName, link); err != nil {
		inv.Status = domain.InvitationDeliveryFailed
		if saveErr := s.Base.Stores.Invitations.Update(ctx, inv); saveErr != nil {
			return nil, saveErr
		}
		if auditErr := s.Base.audit(ctx, inv.InvitedBy, inv.GroupID, "invitation.delivery_failed", "invitation", inv.ID, "failure"); auditErr != nil {
			return nil, domain.ErrAuditUnavailable
		}
		return nil, err
	}
	s.publish(ctx, "created", *inv)
	return inv, nil
}

func (s *CollaborationService) publicApplicationURL() (*url.URL, error) {
	value := s.AppURL
	if s.ApplicationURL != nil {
		value = s.ApplicationURL()
	}
	parsed, err := publicurl.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("%w: PocketBase Application URL must be a valid root HTTP or HTTPS URL", domain.ErrConfiguration)
	}
	return parsed, nil
}
func (s *CollaborationService) ListInvitations(ctx context.Context, userID, groupID string, page ports.PageRequest) (ports.Page[domain.Invitation], error) {
	if err := s.Base.groupPermission(ctx, userID, groupID, "group.members.manage"); err != nil {
		return ports.Page[domain.Invitation]{}, err
	}
	return s.Base.Stores.Invitations.List(ctx, groupID, page)
}
func (s *CollaborationService) ResendInvitation(ctx context.Context, userID, id string) (*domain.Invitation, error) {
	inv, err := s.Base.Stores.Invitations.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err = s.Base.groupPermission(ctx, userID, inv.GroupID, "group.members.manage"); err != nil {
		return nil, err
	}
	if inv.Status == domain.InvitationAccepted || inv.Status == domain.InvitationRevoked {
		return nil, domain.ErrConflict
	}
	if _, err = s.publicApplicationURL(); err != nil {
		return nil, err
	}
	plain, hash, err := domain.NewInvitationToken()
	if err != nil {
		return nil, err
	}
	inv.TokenHash = hash
	inv.Status = domain.InvitationPending
	inv.ExpiresAt = s.Base.Now().UTC().Add(7 * 24 * time.Hour)
	inv.AcceptedBy = ""
	if err = s.Base.Stores.Invitations.Update(ctx, inv); err != nil {
		return nil, err
	}
	return s.deliver(ctx, inv, plain)
}
func (s *CollaborationService) RevokeInvitation(ctx context.Context, userID, id string) error {
	inv, err := s.Base.Stores.Invitations.Get(ctx, id)
	if err != nil {
		return err
	}
	if err = s.Base.groupPermission(ctx, userID, inv.GroupID, "group.members.manage"); err != nil {
		return err
	}
	if inv.Status == domain.InvitationAccepted {
		return domain.ErrConflict
	}
	inv.Status = domain.InvitationRevoked
	notificationRecipientID := ""
	err = s.Base.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if err := s.Base.Stores.Invitations.Update(tx, inv); err != nil {
			return err
		}
		if recipient, findErr := s.Base.Stores.Users.FindByEmail(tx, inv.Email); findErr == nil {
			notificationRecipientID = recipient.ID
			if err := s.Base.Stores.Notifications.SetResourceStatus(tx, recipient.ID, inv.ID, domain.NotificationStatusInvalid, "revoked", s.Base.Now().UTC()); err != nil {
				return err
			}
		} else if !errors.Is(findErr, domain.ErrNotFound) {
			return findErr
		}
		return s.Base.audit(tx, userID, inv.GroupID, "invitation.revoked", "invitation", inv.ID, "success", encodeAuditSummary(map[string]any{"email": inv.Email}, nil))
	})
	if err != nil {
		if auditErr := s.Base.audit(ctx, userID, inv.GroupID, "invitation.revoked", "invitation", inv.ID, "failure"); auditErr != nil {
			return domain.ErrAuditUnavailable
		}
		return err
	}
	s.publish(ctx, "updated", *inv)
	if notificationRecipientID != "" {
		s.publishNotification(ctx, "updated", notificationRecipientID, inv.GroupID, inv.ID)
	}
	return nil
}
func (s *CollaborationService) AcceptInvitation(ctx context.Context, userID, token string) (*domain.Invitation, error) {
	inv, err := s.Base.Stores.Invitations.GetByTokenHash(ctx, domain.HashInvitationToken(token))
	if err != nil {
		return nil, err
	}
	return s.accept(ctx, userID, inv)
}
func (s *CollaborationService) accept(ctx context.Context, userID string, inv *domain.Invitation) (*domain.Invitation, error) {
	now := s.Base.Now().UTC()
	if inv.Status != domain.InvitationPending {
		return nil, domain.ErrConflict
	}
	if !domain.InvitationUsable(*inv, now) {
		inv.Status = domain.InvitationExpired
		_ = s.Base.Stores.Invitations.Update(ctx, inv)
		return nil, domain.ErrConflict
	}
	user, err := s.Base.Stores.Users.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if domain.NormalizeEmail(user.Email) != domain.NormalizeEmail(inv.Email) {
		return nil, domain.ErrForbidden
	}
	boundPlaceholderID := inv.TargetPlaceholderID
	auditDetails := map[string]any{"email": inv.Email}
	if boundPlaceholderID != "" {
		auditDetails["bound_placeholder_id"] = boundPlaceholderID
		auditDetails["bound_to_user_id"] = userID
	}
	err = s.Base.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if _, roleErr := s.Base.Stores.Memberships.GetRole(tx, inv.GroupID, userID); roleErr != nil {
			if err := s.Base.Stores.Memberships.Create(tx, &domain.Membership{GroupID: inv.GroupID, UserID: userID, Role: domain.RoleMember}); err != nil {
				return err
			}
		}
		if inv.TargetPlaceholderID != "" {
			// Eagerly repoint every historical expense_split/subscription/
			// settlement reference from the placeholder to this real
			// account, then delete the now-orphaned placeholder row --
			// unlike a member-to-member transfer, a bind never leaves a
			// membership behind to remove (the placeholder's own membership
			// dies along with it), so this repoints without going through
			// the membership-delete step.
			if err := s.Base.repointUserReferences(tx, inv.GroupID, inv.TargetPlaceholderID, userID, false); err != nil {
				return err
			}
			if err := s.Base.Stores.Users.Delete(tx, inv.TargetPlaceholderID); err != nil {
				return err
			}
			inv.TargetPlaceholderID = ""
		}
		inv.Status = domain.InvitationAccepted
		inv.AcceptedBy = userID
		if err := s.Base.Stores.Invitations.Update(tx, inv); err != nil {
			return err
		}
		if err := s.Base.Stores.Notifications.SetResourceStatus(tx, userID, inv.ID, domain.NotificationStatusResolved, "accepted", now); err != nil {
			return err
		}
		if err := s.Base.Stores.Notifications.MarkReadForResource(tx, userID, inv.ID, now); err != nil {
			return err
		}
		return s.Base.audit(tx, userID, inv.GroupID, "invitation.accepted", "invitation", inv.ID, "success", encodeAuditSummary(auditDetails, nil))
	})
	if err != nil {
		if auditErr := s.Base.audit(ctx, userID, inv.GroupID, "invitation.accepted", "invitation", inv.ID, "failure"); auditErr != nil {
			return nil, domain.ErrAuditUnavailable
		}
		return nil, err
	}
	s.publish(ctx, "accepted", *inv)
	s.publishNotification(ctx, "updated", userID, inv.GroupID, inv.ID)
	return inv, nil
}
func (s *CollaborationService) ListMyInvitations(ctx context.Context, userID string, page ports.PageRequest) (ports.Page[domain.Invitation], error) {
	user, err := s.Base.Stores.Users.Get(ctx, userID)
	if err != nil {
		return ports.Page[domain.Invitation]{}, err
	}
	return s.Base.Stores.Invitations.ListForEmail(ctx, domain.NormalizeEmail(user.Email), page)
}
func (s *CollaborationService) AcceptInvitationByID(ctx context.Context, userID, id string) (*domain.Invitation, error) {
	inv, err := s.Base.Stores.Invitations.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.accept(ctx, userID, inv)
}
func (s *CollaborationService) DeclineInvitation(ctx context.Context, userID, id string) (*domain.Invitation, error) {
	inv, err := s.Base.Stores.Invitations.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	user, err := s.Base.Stores.Users.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if inv.Status != domain.InvitationPending || domain.NormalizeEmail(user.Email) != domain.NormalizeEmail(inv.Email) {
		return nil, domain.ErrForbidden
	}
	inv.Status = domain.InvitationDeclined
	if err = s.Base.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if err := s.Base.Stores.Invitations.Update(tx, inv); err != nil {
			return err
		}
		if err := s.Base.Stores.Notifications.SetResourceStatus(tx, userID, inv.ID, domain.NotificationStatusResolved, "declined", s.Base.Now().UTC()); err != nil {
			return err
		}
		if err := s.Base.Stores.Notifications.MarkReadForResource(tx, userID, inv.ID, s.Base.Now().UTC()); err != nil {
			return err
		}
		return s.Base.audit(tx, userID, inv.GroupID, "invitation.declined", "invitation", inv.ID, "success", encodeAuditSummary(map[string]any{"email": inv.Email}, nil))
	}); err != nil {
		if auditErr := s.Base.audit(ctx, userID, inv.GroupID, "invitation.declined", "invitation", inv.ID, "failure"); auditErr != nil {
			return nil, domain.ErrAuditUnavailable
		}
		return nil, err
	}
	s.publish(ctx, "updated", *inv)
	s.publishNotification(ctx, "updated", userID, inv.GroupID, inv.ID)
	return inv, nil
}
func (s *CollaborationService) ListNotifications(ctx context.Context, userID string, page ports.PageRequest) (ports.Page[domain.Notification], error) {
	return s.notificationService().ListForUser(ctx, userID, page)
}
func (s *CollaborationService) ListNotificationsFiltered(ctx context.Context, userID string, query ports.NotificationQuery) (ports.Page[domain.Notification], error) {
	return s.notificationService().ListFiltered(ctx, userID, query)
}
func (s *CollaborationService) NotificationSummary(ctx context.Context, userID string) (ports.NotificationSummary, error) {
	return s.notificationService().Summary(ctx, userID)
}
func (s *CollaborationService) NotificationPreferences(ctx context.Context, userID string) (domain.NotificationPreferences, error) {
	return s.notificationService().Preferences(ctx, userID)
}
func (s *CollaborationService) UpdateNotificationPreferences(ctx context.Context, userID string, value domain.NotificationPreferences) error {
	return s.notificationService().UpdatePreferences(ctx, userID, value)
}
func (s *CollaborationService) MarkNotificationRead(ctx context.Context, userID, id string) error {
	return s.notificationService().MarkRead(ctx, userID, id)
}
func (s *CollaborationService) MarkNotificationUnread(ctx context.Context, userID, id string) error {
	return s.notificationService().MarkUnread(ctx, userID, id)
}
func (s *CollaborationService) MarkNotificationsRead(ctx context.Context, userID string) error {
	return s.notificationService().MarkReadAll(ctx, userID)
}
func (s *CollaborationService) MarkNotificationsViewed(ctx context.Context, userID string, ids []string) error {
	return s.notificationService().MarkViewed(ctx, userID, ids)
}

func (s *CollaborationService) notificationService() *NotificationService {
	return NewNotificationService(s.Base.Stores.Notifications, s.Base.Stores.Transactions, s.Base.Now,
		func(ctx context.Context, actor, groupID, action, resource, id, outcome string) error {
			return s.Base.audit(ctx, actor, groupID, action, resource, id, outcome)
		})
}
func (s *CollaborationService) publish(ctx context.Context, kind string, inv domain.Invitation) {
	if s.Events != nil {
		_ = s.Events.Publish(ctx, domain.Event{Type: kind, GroupID: inv.GroupID, Resource: "group_invitations", ResourceID: inv.ID, OccurredAt: s.Base.Now().UTC()})
	}
}

func (s *CollaborationService) publishNotification(ctx context.Context, kind, userID, groupID, notificationID string) {
	if s.Events != nil {
		_ = s.Events.Publish(ctx, domain.Event{Type: kind, UserID: userID, Resource: "notifications", ResourceID: notificationID, OccurredAt: s.Base.Now().UTC()})
	}
}
