package application

import (
	"context"
	"errors"

	"subflow/internal/domain"
)

// Logout records explicit sign-out before the client discards its bearer token.
// PocketBase bearer authentication remains stateless; this does not revoke
// other devices' sessions. A failed audit rejects confirmation to the client.
func (s *Service) Logout(ctx context.Context, userID string) error {
	actor, outcome, reason := userID, "success", "explicit_sign_out"
	var denied error
	if userID == "" {
		denied = domain.ErrForbidden
	} else if _, err := s.Stores.Users.Get(ctx, userID); err != nil {
		denied = domain.ErrForbidden
	}
	if denied != nil {
		actor, outcome, reason = "", "failure", "authentication_required"
	}
	if err := s.audit(ctx, actor, "", "auth.logout", "user", actor, outcome, encodeAuditSummary(map[string]any{"reason": reason}, nil)); err != nil {
		if outcome == "success" {
			failure := s.audit(ctx, actor, "", "auth.logout", "user", actor, "failure", encodeAuditSummary(map[string]any{"reason": "audit_unavailable"}, nil))
			return errors.Join(domain.ErrAuditUnavailable, err, failure)
		}
		return errors.Join(domain.ErrAuditUnavailable, err)
	}
	return denied
}
