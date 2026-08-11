package application

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

type CollaborationService struct {
	Base                *Service
	Events              ports.EventPublisher
	Mailer              ports.Mailer
	Environment, AppURL string
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
	}
	plain, hash, err := domain.NewInvitationToken()
	if err != nil {
		return nil, err
	}
	inv := &domain.Invitation{GroupID: groupID, Email: email, TokenHash: hash, Status: domain.InvitationPending, InvitedBy: userID, ExpiresAt: s.Base.Now().UTC().Add(7 * 24 * time.Hour), TargetPlaceholderID: targetPlaceholderID}
	if err = s.Base.Stores.Invitations.Create(ctx, inv); err != nil {
		return nil, err
	}
	if recipient, findErr := s.Base.Stores.Users.FindByEmail(ctx, email); findErr == nil {
		_ = s.Base.Stores.Notifications.Create(ctx, &domain.Notification{UserID: recipient.ID, Type: "group_invitation", GroupID: groupID, ResourceID: inv.ID})
	}
	s.Base.audit(ctx, userID, groupID, "invitation.created", "invitation", inv.ID, "success", encodeAuditSummary(map[string]any{"email": email}, nil))
	return s.deliver(ctx, inv, plain)
}
func (s *CollaborationService) deliver(ctx context.Context, inv *domain.Invitation, plain string) (*domain.Invitation, error) {
	group, err := s.Base.Stores.Groups.Get(ctx, inv.GroupID)
	if err != nil {
		return nil, err
	}
	link := strings.TrimRight(s.AppURL, "/") + "/invite/" + url.PathEscape(plain)
	if strings.EqualFold(s.Environment, "development") || s.Environment == "" {
		inv.DebugURL = link
	}
	if s.Mailer == nil || !s.Mailer.Configured() {
		if !strings.EqualFold(s.Environment, "development") && s.Environment != "" {
			inv.Status = domain.InvitationDeliveryFailed
			_ = s.Base.Stores.Invitations.Update(ctx, inv)
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
		_ = s.Base.Stores.Invitations.Update(ctx, inv)
		return nil, err
	}
	s.publish(ctx, "created", *inv)
	return inv, nil
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
	if err = s.Base.Stores.Invitations.Update(ctx, inv); err != nil {
		return err
	}
	s.publish(ctx, "updated", *inv)
	s.Base.audit(ctx, userID, inv.GroupID, "invitation.revoked", "invitation", inv.ID, "success", encodeAuditSummary(map[string]any{"email": inv.Email}, nil))
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
	if inv.Status != domain.InvitationPending { return nil, domain.ErrConflict }
	if !domain.InvitationUsable(*inv, now) { inv.Status = domain.InvitationExpired; _ = s.Base.Stores.Invitations.Update(ctx, inv); return nil, domain.ErrConflict }
	user, err := s.Base.Stores.Users.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if domain.NormalizeEmail(user.Email) != domain.NormalizeEmail(inv.Email) {
		return nil, domain.ErrForbidden
	}
	err = s.Base.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if _, roleErr := s.Base.Stores.Memberships.GetRole(tx, inv.GroupID, userID); roleErr != nil {
			if err := s.Base.Stores.Memberships.Create(tx, &domain.Membership{GroupID: inv.GroupID, UserID: userID, Role: domain.RoleMember}); err != nil {
				return err
			}
		}
		if inv.TargetPlaceholderID != "" {
			// Historical expense_splits/settlements/subscriptions keep
			// pointing at the placeholder's own ID rather than being
			// rewritten — see WorkspaceDashboard's alias resolution, which
			// folds a bound placeholder's balance into this linked account.
			if err := s.Base.Stores.Users.LinkPlaceholder(tx, inv.TargetPlaceholderID, userID); err != nil {
				return err
			}
		}
		inv.Status = domain.InvitationAccepted
		inv.AcceptedBy = userID
		if err := s.Base.Stores.Invitations.Update(tx, inv); err != nil { return err }
		return s.Base.Stores.Notifications.MarkReadForResource(tx, userID, inv.ID, now)
	})
	if err != nil {
		return nil, err
	}
	s.Base.audit(ctx, userID, inv.GroupID, "invitation.accepted", "invitation", inv.ID, "success", encodeAuditSummary(map[string]any{"email": inv.Email}, nil))
	s.publish(ctx, "accepted", *inv)
	return inv, nil
}
func (s *CollaborationService) ListMyInvitations(ctx context.Context, userID string, page ports.PageRequest) (ports.Page[domain.Invitation], error) { user, err := s.Base.Stores.Users.Get(ctx, userID); if err != nil { return ports.Page[domain.Invitation]{}, err }; return s.Base.Stores.Invitations.ListForEmail(ctx, domain.NormalizeEmail(user.Email), page) }
func (s *CollaborationService) AcceptInvitationByID(ctx context.Context, userID, id string) (*domain.Invitation, error) { inv, err := s.Base.Stores.Invitations.Get(ctx, id); if err != nil { return nil, err }; return s.accept(ctx, userID, inv) }
func (s *CollaborationService) DeclineInvitation(ctx context.Context, userID, id string) (*domain.Invitation, error) {
	inv, err := s.Base.Stores.Invitations.Get(ctx, id); if err != nil { return nil, err }
	user, err := s.Base.Stores.Users.Get(ctx, userID); if err != nil { return nil, err }
	if inv.Status != domain.InvitationPending || domain.NormalizeEmail(user.Email) != domain.NormalizeEmail(inv.Email) { return nil, domain.ErrForbidden }
	inv.Status = domain.InvitationDeclined
	if err = s.Base.Stores.Transactions.Within(ctx, func(tx context.Context) error { if err := s.Base.Stores.Invitations.Update(tx, inv); err != nil { return err }; return s.Base.Stores.Notifications.MarkReadForResource(tx, userID, inv.ID, s.Base.Now().UTC()) }); err != nil { return nil, err }
	s.Base.audit(ctx, userID, inv.GroupID, "invitation.declined", "invitation", inv.ID, "success", encodeAuditSummary(map[string]any{"email": inv.Email}, nil)); s.publish(ctx, "updated", *inv); return inv, nil
}
func (s *CollaborationService) ListNotifications(ctx context.Context, userID string, page ports.PageRequest) (ports.Page[domain.Notification], error) { return s.Base.Stores.Notifications.ListForUser(ctx, userID, page) }
func (s *CollaborationService) MarkNotificationRead(ctx context.Context, userID, id string) error { note, err := s.Base.Stores.Notifications.Get(ctx, id); if err != nil { return err }; if note.UserID != userID { return domain.ErrForbidden }; if note.ReadAt == nil { if err = s.Base.Stores.Notifications.MarkRead(ctx, id, s.Base.Now().UTC()); err == nil { s.Base.audit(ctx, userID, note.GroupID, "notification.read", "notification", id, "success", encodeAuditSummary(map[string]any{"type": note.Type}, nil)) } }; return err }
func (s *CollaborationService) publish(ctx context.Context, kind string, inv domain.Invitation) {
	if s.Events != nil {
		_ = s.Events.Publish(ctx, domain.Event{Type: kind, GroupID: inv.GroupID, Resource: "group_invitations", ResourceID: inv.ID, OccurredAt: s.Base.Now().UTC()})
	}
}
