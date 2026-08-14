package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

func (s *Service) audit(ctx context.Context, actor, groupID, action, resource, resourceID, outcome string, summary ...string) {
	key := os.Getenv("SUBFLOW_AUDIT_HMAC_KEY")
	if key == "" {
		key = "subflow-audit-local"
	}
	summaryText := ""
	if len(summary) > 0 {
		summaryText = strings.TrimSpace(summary[0])
	}
	meta := auditRequestMetaFrom(ctx)
	ip, userAgent := meta.IP, meta.UserAgent
	// A context that never passed through the HTTP layer's middleware (a
	// background job like PostDueSubscriptions, running from
	// context.Background()) has no caller to record; say so explicitly
	// rather than leaving the column blank and indistinguishable from a
	// bug that failed to capture it.
	if ip == "" {
		ip = "system"
	}
	if userAgent == "" {
		userAgent = "system"
	}
	// The "v2" prefix distinguishes this payload shape from any pre-IP/UA
	// row, should a verifier ever need to tell them apart; nothing in this
	// codebase re-derives or checks the hash today (grep confirms), so this
	// change can't invalidate anything that currently depends on it.
	payload := strings.Join([]string{"v2", actor, groupID, action, resource, resourceID, outcome, summaryText, ip, userAgent}, "|")
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(payload))
	_ = s.Stores.Audits.Create(ctx, &domain.AuditLog{ActorID: actor, GroupID: groupID, Scope: map[bool]string{true: "group", false: "system"}[groupID != ""], Action: action, Resource: resource, ResourceID: resourceID, Outcome: outcome, Summary: summaryText, IP: ip, UserAgent: userAgent, Hash: hex.EncodeToString(mac.Sum(nil))})
}

func generatedRoleKey(name string) string { sum := sha256.Sum256([]byte(name + time.Now().UTC().String())); return fmt.Sprintf("custom-%x", sum[:6]) }

func (s *Service) ListGroupRoles(ctx context.Context, userID, groupID string) ([]domain.Role, error) {
	if err := s.role(ctx, groupID, userID, false); err != nil {
		return nil, err
	}
	return s.Stores.Roles.List(ctx, "group", groupID)
}
func (s *Service) systemPermission(ctx context.Context, userID, permission string) error {
	user, err := s.Stores.Users.Get(ctx, userID)
	if err != nil || user.SystemRoleID == "" {
		return domain.ErrForbidden
	}
	role, err := s.Stores.Roles.Get(ctx, "system", user.SystemRoleID)
	if err != nil {
		return domain.ErrForbidden
	}
	for _, value := range role.Permissions {
		if value == permission || value == "*" {
			return nil
		}
	}
	return domain.ErrForbidden
}
func (s *Service) SystemPermissions(ctx context.Context, userID string) ([]string, error) {
	user, err := s.Stores.Users.Get(ctx, userID)
	if err != nil || user.SystemRoleID == "" {
		return []string{}, nil
	}
	role, err := s.Stores.Roles.Get(ctx, "system", user.SystemRoleID)
	if err != nil {
		return []string{}, nil
	}
	return role.Permissions, nil
}

// GroupPermissions returns the permissions of the caller's assigned group
// role.  Membership.Role is kept for backwards compatibility, while RoleID
// is the authoritative RBAC relation for new and migrated memberships.
func (s *Service) GroupPermissions(ctx context.Context, userID, groupID string) ([]string, error) {
	members, err := s.Stores.Memberships.List(ctx, groupID, ports.PageRequest{Page: 1, PerPage: 100})
	if err != nil {
		return nil, err
	}
	for _, member := range members.Items {
		if member.UserID != userID {
			continue
		}
		if member.RoleID != "" {
			role, roleErr := s.Stores.Roles.Get(ctx, "group", member.RoleID)
			if roleErr != nil {
				return nil, roleErr
			}
			return role.Permissions, nil
		}
		// Old records can briefly exist without role_ref during an upgrade.
		// Resolve their protected owner/member role by key instead of denying
		// a valid existing member while the schema backfill completes.
		roles, roleErr := s.Stores.Roles.List(ctx, "group", groupID)
		if roleErr != nil {
			return nil, roleErr
		}
		for _, role := range roles {
			if role.Key == string(member.Role) {
				return role.Permissions, nil
			}
		}
		return nil, domain.ErrForbidden
	}
	return nil, domain.ErrForbidden
}

func (s *Service) groupPermission(ctx context.Context, userID, groupID, permission string) error {
	permissions, err := s.GroupPermissions(ctx, userID, groupID)
	if err != nil {
		return err
	}
	for _, value := range permissions {
		if value == permission || value == "*" {
			return nil
		}
	}
	return domain.ErrForbidden
}
func (s *Service) ListSystemUsers(ctx context.Context, userID string, page ports.PageRequest, query string) (ports.Page[domain.User], error) {
	if err := s.systemPermission(ctx, userID, "system.users.assign"); err != nil {
		return ports.Page[domain.User]{}, err
	}
	return s.Stores.Users.List(ctx, page, query)
}
func (s *Service) ListSystemRoles(ctx context.Context, userID string) ([]domain.Role, error) {
	if err := s.systemPermission(ctx, userID, "system.roles.manage"); err != nil {
		return nil, err
	}
	return s.Stores.Roles.List(ctx, "system", "")
}
func (s *Service) CreateSystemRole(ctx context.Context, userID string, value domain.Role) (*domain.Role, error) {
	if err := s.systemPermission(ctx, userID, "system.roles.manage"); err != nil {
		return nil, err
	}
	value.Scope, value.GroupID, value.CreatedBy = "system", "", userID
	value.Name = strings.TrimSpace(value.Name)
	value.Category = strings.TrimSpace(value.Category)
	if value.Key == "" { value.Key = generatedRoleKey(value.Name) }
	value.Key = strings.ToLower(strings.TrimSpace(value.Key))
	if value.Name == "" || value.Key == "" || value.Protected {
		return nil, domain.ErrInvalid
	}
	roles, err := s.Stores.Roles.List(ctx, "system", "")
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		if role.Key == value.Key {
			return nil, domain.ErrConflict
		}
	}
	if err = s.Stores.Roles.Create(ctx, &value); err != nil {
		return nil, err
	}
	s.audit(ctx, userID, "", "system_role.created", "system_role", value.ID, "success", encodeAuditSummary(map[string]any{"name": value.Name, "key": value.Key, "permissions": value.Permissions}, nil))
	return &value, nil
}
func (s *Service) UpdateSystemRole(ctx context.Context, userID string, value domain.Role) (*domain.Role, error) {
	if err := s.systemPermission(ctx, userID, "system.roles.manage"); err != nil {
		return nil, err
	}
	current, err := s.Stores.Roles.Get(ctx, "system", value.ID)
	if err != nil {
		return nil, err
	}
	if current.Protected {
		return nil, domain.ErrForbidden
	}
	before := *current
	current.Name = strings.TrimSpace(value.Name)
	current.Category = strings.TrimSpace(value.Category)
	current.Permissions = value.Permissions
	if current.Name == "" {
		return nil, domain.ErrInvalid
	}
	if err = s.Stores.Roles.Update(ctx, current); err != nil {
		return nil, err
	}
	var roleChanges changeSet
	roleChanges.addString("name", before.Name, current.Name)
	roleChanges.addString("category", before.Category, current.Category)
	roleChanges.addStrings("permissions", before.Permissions, current.Permissions)
	s.audit(ctx, userID, "", "system_role.updated", "system_role", current.ID, "success", encodeAuditSummary(nil, roleChanges))
	return current, nil
}
func (s *Service) DeleteSystemRole(ctx context.Context, userID, id string) error {
	if err := s.systemPermission(ctx, userID, "system.roles.manage"); err != nil {
		return err
	}
	role, err := s.Stores.Roles.Get(ctx, "system", id)
	if err != nil {
		return err
	}
	if role.Protected {
		return domain.ErrForbidden
	}
	err = s.Stores.Roles.Delete(ctx, "system", id)
	if err == nil {
		s.audit(ctx, userID, "", "system_role.deleted", "system_role", id, "success", encodeAuditSummary(map[string]any{"name": role.Name, "key": role.Key}, nil))
	}
	return err
}
func (s *Service) AssignSystemRole(ctx context.Context, userID, targetID, roleID string) error {
	if err := s.systemPermission(ctx, userID, "system.users.assign"); err != nil {
		return err
	}
	role, err := s.Stores.Roles.Get(ctx, "system", roleID)
	if err != nil {
		return err
	}
	if role.Scope != "system" {
		return domain.ErrInvalid
	}
	if err = s.Stores.Users.SetSystemRole(ctx, targetID, roleID); err != nil {
		return err
	}
	s.audit(ctx, userID, "", "system_role.assigned", "user", targetID, "success", encodeAuditSummary(map[string]any{"role": role.Name}, nil))
	return nil
}
func (s *Service) RecoverSystemAdmin(ctx context.Context, targetID string) error {
	settings, err := s.Stores.Settings.Get(ctx)
	if err != nil {
		return err
	}
	if !settings.Initialized {
		return domain.ErrSetupDisabled
	}
	roles, err := s.Stores.Roles.List(ctx, "system", "")
	if err != nil {
		return err
	}
	for _, role := range roles {
		if role.Key == "admin" {
			count, e := s.Stores.Users.CountBySystemRole(ctx, role.ID)
			if e != nil {
				return e
			}
			if count > 0 {
				return domain.ErrConflict
			}
			if err = s.Stores.Users.SetSystemRole(ctx, targetID, role.ID); err == nil {
				s.audit(ctx, targetID, "", "system_role.recovered", "user", targetID, "success", encodeAuditSummary(map[string]any{"role": role.Name}, nil))
			}
			return err
		}
	}
	return domain.ErrNotFound
}
func (s *Service) ListSystemAudit(ctx context.Context, userID string, query ports.AuditQuery) (ports.Page[domain.AuditLog], error) {
	if err := s.systemPermission(ctx, userID, "system.audit.read"); err != nil {
		return ports.Page[domain.AuditLog]{}, err
	}
	return s.Stores.Audits.List(ctx, "", query)
}
func (s *Service) CreateGroupRole(ctx context.Context, userID string, value domain.Role) (*domain.Role, error) {
	if err := s.groupPermission(ctx, userID, value.GroupID, "group.roles.manage"); err != nil {
		return nil, err
	}
	value.Scope = "group"
	value.Name = strings.TrimSpace(value.Name)
	value.Category = strings.TrimSpace(value.Category)
	if value.Key == "" { value.Key = generatedRoleKey(value.Name) }
	value.Key = strings.ToLower(strings.TrimSpace(value.Key))
	value.CreatedBy = userID
	if value.Name == "" || value.Key == "" || value.Protected {
		return nil, domain.ErrInvalid
	}
	roles, err := s.Stores.Roles.List(ctx, "group", value.GroupID)
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		if role.Key == value.Key {
			return nil, domain.ErrConflict
		}
	}
	if err = s.Stores.Roles.Create(ctx, &value); err != nil {
		return nil, err
	}
	s.audit(ctx, userID, value.GroupID, "role.created", "group_role", value.ID, "success", encodeAuditSummary(map[string]any{"name": value.Name, "key": value.Key, "permissions": value.Permissions}, nil))
	return &value, nil
}
func (s *Service) UpdateGroupRole(ctx context.Context, userID string, value domain.Role) (*domain.Role, error) {
	current, err := s.Stores.Roles.Get(ctx, "group", value.ID)
	if err != nil {
		return nil, err
	}
	if err = s.groupPermission(ctx, userID, current.GroupID, "group.roles.manage"); err != nil {
		return nil, err
	}
	if current.Protected {
		return nil, domain.ErrForbidden
	}
	before := *current
	current.Name = strings.TrimSpace(value.Name)
	current.Category = strings.TrimSpace(value.Category)
	current.Permissions = value.Permissions
	if current.Name == "" {
		return nil, domain.ErrInvalid
	}
	if err = s.Stores.Roles.Update(ctx, current); err != nil {
		return nil, err
	}
	var groupRoleChanges changeSet
	groupRoleChanges.addString("name", before.Name, current.Name)
	groupRoleChanges.addString("category", before.Category, current.Category)
	groupRoleChanges.addStrings("permissions", before.Permissions, current.Permissions)
	s.audit(ctx, userID, current.GroupID, "role.updated", "group_role", current.ID, "success", encodeAuditSummary(nil, groupRoleChanges))
	return current, nil
}
func (s *Service) DeleteGroupRole(ctx context.Context, userID, id string) error {
	current, err := s.Stores.Roles.Get(ctx, "group", id)
	if err != nil {
		return err
	}
	if err = s.groupPermission(ctx, userID, current.GroupID, "group.roles.manage"); err != nil {
		return err
	}
	if current.Protected {
		return domain.ErrForbidden
	}
	members, err := s.Stores.Memberships.List(ctx, current.GroupID, ports.PageRequest{Page: 1, PerPage: 100})
	if err != nil {
		return err
	}
	for _, m := range members.Items {
		if m.RoleID == id {
			return domain.ErrConflict
		}
	}
	err = s.Stores.Roles.Delete(ctx, "group", id)
	if err == nil {
		s.audit(ctx, userID, current.GroupID, "role.deleted", "group_role", id, "success", encodeAuditSummary(map[string]any{"name": current.Name, "key": current.Key}, nil))
	}
	return err
}
func (s *Service) AssignGroupRole(ctx context.Context, userID, groupID, memberID, roleID string) error {
	if err := s.groupPermission(ctx, userID, groupID, "group.roles.manage"); err != nil {
		return err
	}
	role, err := s.Stores.Roles.Get(ctx, "group", roleID)
	if err != nil {
		return err
	}
	if role.GroupID != groupID {
		return domain.ErrInvalid
	}
	// The protected "owner" group_roles record can only ever be held by
	// exactly one member, and only via the ownership-transfer flow
	// (Service.RespondOwnershipTransfer), which updates the membership
	// directly. Granting or moving it through this general-purpose
	// role-assignment endpoint would let a second member end up with full
	// owner permissions while Group.OwnerID (and CanManageGroup's enum
	// check) still points at someone else.
	if role.Key == "owner" {
		s.audit(ctx, userID, groupID, "role.assigned", "membership", memberID, "failure", encodeAuditSummary(map[string]any{"role": role.Name}, nil))
		return domain.ErrForbidden
	}
	target, err := s.Stores.Memberships.GetRole(ctx, groupID, memberID)
	if err != nil {
		return err
	}
	if target == domain.RoleOwner {
		return domain.ErrForbidden
	}
	err = s.Stores.Memberships.UpdateRole(ctx, groupID, memberID, roleID)
	if err == nil {
		s.audit(ctx, userID, groupID, "role.assigned", "membership", memberID, "success", encodeAuditSummary(map[string]any{"role": role.Name}, nil))
	}
	return err
}
func (s *Service) ListGroupAudit(ctx context.Context, userID, groupID string, query ports.AuditQuery) (ports.Page[domain.AuditLog], error) {
	if err := s.groupPermission(ctx, userID, groupID, "group.audit.read"); err != nil {
		return ports.Page[domain.AuditLog]{}, err
	}
	return s.Stores.Audits.List(ctx, groupID, query)
}
