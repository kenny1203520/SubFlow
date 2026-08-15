package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"subflow/internal/domain"
)

func TestCreateInvitationRequiresPermission(t *testing.T) {
	collab, stores, ownerID, groupID := newTempMemberFixture(t)
	ctx := context.Background()
	memberID := newRealUser(t, stores, "plain-member@example.com")
	if err := stores.Memberships.Create(ctx, &domain.Membership{GroupID: groupID, UserID: memberID, Role: domain.RoleMember}); err != nil {
		t.Fatal(err)
	}

	if _, err := collab.CreateInvitation(ctx, memberID, groupID, "someone@example.com"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for a member without group.members.manage, got %v", err)
	}
	if _, err := collab.CreateInvitation(ctx, ownerID, groupID, "someone@example.com"); err != nil {
		t.Fatalf("expected the owner to be allowed to invite: %v", err)
	}
}

func TestCreateInvitationRejectsInvalidEmail(t *testing.T) {
	collab, _, ownerID, groupID := newTempMemberFixture(t)
	ctx := context.Background()
	if _, err := collab.CreateInvitation(ctx, ownerID, groupID, "not-an-email"); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for a malformed email, got %v", err)
	}
}

func TestCreateInvitationRejectsDuplicatePending(t *testing.T) {
	collab, _, ownerID, groupID := newTempMemberFixture(t)
	ctx := context.Background()
	if _, err := collab.CreateInvitation(ctx, ownerID, groupID, "dup@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := collab.CreateInvitation(ctx, ownerID, groupID, "dup@example.com"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict for a second pending invite to the same email, got %v", err)
	}
}

func TestResendAndRevokeInvitationRejectAfterTerminalStatus(t *testing.T) {
	collab, stores, ownerID, groupID := newTempMemberFixture(t)
	ctx := context.Background()

	inv, err := collab.CreateInvitation(ctx, ownerID, groupID, "resend@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = collab.ResendInvitation(ctx, ownerID, inv.ID); err != nil {
		t.Fatalf("expected resending a pending invitation to succeed: %v", err)
	}

	realUserID := newRealUser(t, stores, "resend@example.com")
	if _, err = collab.AcceptInvitationByID(ctx, realUserID, inv.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = collab.ResendInvitation(ctx, ownerID, inv.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict resending an already-accepted invitation, got %v", err)
	}
	if err = collab.RevokeInvitation(ctx, ownerID, inv.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict revoking an already-accepted invitation, got %v", err)
	}
}

// A revoked invitation is a dead end, not a pausable one: it can never be
// resent (the caller must issue a fresh invitation instead).
func TestRevokeInvitationThenResendIsRejected(t *testing.T) {
	collab, _, ownerID, groupID := newTempMemberFixture(t)
	ctx := context.Background()
	inv, err := collab.CreateInvitation(ctx, ownerID, groupID, "revoke@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err = collab.RevokeInvitation(ctx, ownerID, inv.ID); err != nil {
		t.Fatalf("expected revoking a pending invitation to succeed: %v", err)
	}
	if _, err = collab.ResendInvitation(ctx, ownerID, inv.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict resending a revoked invitation, got %v", err)
	}
}

func TestAcceptInvitationRejectsEmailMismatch(t *testing.T) {
	collab, stores, ownerID, groupID := newTempMemberFixture(t)
	ctx := context.Background()
	inv, err := collab.CreateInvitation(ctx, ownerID, groupID, "intended@example.com")
	if err != nil {
		t.Fatal(err)
	}
	wrongUserID := newRealUser(t, stores, "someone-else@example.com")
	if _, err = collab.AcceptInvitationByID(ctx, wrongUserID, inv.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden accepting with a mismatched email, got %v", err)
	}
}

func TestAcceptInvitationRejectsExpiredAndMarksItExpired(t *testing.T) {
	collab, stores, ownerID, groupID := newTempMemberFixture(t)
	ctx := context.Background()
	inv, err := collab.CreateInvitation(ctx, ownerID, groupID, "expired@example.com")
	if err != nil {
		t.Fatal(err)
	}
	realUserID := newRealUser(t, stores, "expired@example.com")

	// Fast-forward past the 7-day expiry window.
	collab.Base.Now = func() time.Time { return inv.ExpiresAt.Add(time.Hour) }
	if _, err = collab.AcceptInvitationByID(ctx, realUserID, inv.ID); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict accepting an expired invitation, got %v", err)
	}
	stored, err := stores.Invitations.Get(ctx, inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != domain.InvitationExpired {
		t.Fatalf("expected the invitation to be marked expired, got %q", stored.Status)
	}
}

func TestDeclineInvitationRejectsEmailMismatchAndWrongStatus(t *testing.T) {
	collab, stores, ownerID, groupID := newTempMemberFixture(t)
	ctx := context.Background()
	inv, err := collab.CreateInvitation(ctx, ownerID, groupID, "decline@example.com")
	if err != nil {
		t.Fatal(err)
	}
	wrongUserID := newRealUser(t, stores, "not-invited@example.com")
	if _, err = collab.DeclineInvitation(ctx, wrongUserID, inv.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden declining with a mismatched email, got %v", err)
	}

	realUserID := newRealUser(t, stores, "decline@example.com")
	if _, err = collab.DeclineInvitation(ctx, realUserID, inv.ID); err != nil {
		t.Fatalf("expected declining a pending invitation to succeed: %v", err)
	}
	if _, err = collab.DeclineInvitation(ctx, realUserID, inv.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden declining an already-declined invitation, got %v", err)
	}
}

func TestListMyInvitationsFiltersByAcceptingEmail(t *testing.T) {
	collab, stores, ownerID, groupID := newTempMemberFixture(t)
	ctx := context.Background()
	if _, err := collab.CreateInvitation(ctx, ownerID, groupID, "mine@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := collab.CreateInvitation(ctx, ownerID, groupID, "someone-else@example.com"); err != nil {
		t.Fatal(err)
	}
	myUserID := newRealUser(t, stores, "mine@example.com")

	page, err := collab.ListMyInvitations(ctx, myUserID, defaultAuditQuery().PageRequest)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Email != "mine@example.com" {
		t.Fatalf("expected exactly the invitation addressed to this user's email, got %#v", page.Items)
	}
}

// CreateInvitation only notifies a user it can find by email, so the
// recipient must already have an account before the invite is sent.
func TestMarkNotificationReadRejectsOtherUsersNotification(t *testing.T) {
	collab, stores, ownerID, groupID := newTempMemberFixture(t)
	ctx := context.Background()
	recipientID := newRealUser(t, stores, "notify@example.com")
	if _, err := collab.CreateInvitation(ctx, ownerID, groupID, "notify@example.com"); err != nil {
		t.Fatal(err)
	}

	notifications, err := collab.ListNotifications(ctx, recipientID, defaultAuditQuery().PageRequest)
	if err != nil {
		t.Fatal(err)
	}
	if len(notifications.Items) != 1 {
		t.Fatalf("expected exactly 1 notification for the invited (pre-existing) user, got %#v", notifications.Items)
	}
	noteID := notifications.Items[0].ID

	otherUserID := newRealUser(t, stores, "unrelated@example.com")
	if err = collab.MarkNotificationRead(ctx, otherUserID, noteID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden marking someone else's notification read, got %v", err)
	}
	if err = collab.MarkNotificationRead(ctx, recipientID, noteID); err != nil {
		t.Fatalf("expected the recipient to be able to mark their own notification read: %v", err)
	}
}
