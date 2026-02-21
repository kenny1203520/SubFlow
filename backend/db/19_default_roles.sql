-- Insert Default System Roles
INSERT INTO system_roles (name, description, is_system_role)
VALUES 
  ('Administrator', 'Full access to all system resources - has every permission automatically', true),
  ('Support Agent', 'Can view user accounts and revoke sessions, but cannot modify security settings or bans', true)
ON CONFLICT (name) DO NOTHING;

-- Seed default permissions for all scopes
INSERT INTO permissions (scope, action, resource, description)
VALUES
  ('system', 'create', 'users',    'Create new users'),
  ('system', 'read',   'users',    'View user list and details'),
  ('system', 'update', 'users',    'Update any user account'),
  ('system', 'delete', 'users',    'Delete any user account'),
  ('system', 'read',   'sessions', 'View all active sessions'),
  ('system', 'delete', 'sessions', 'Revoke any user session'),
  ('system', 'read',   'settings', 'View system settings'),
  ('system', 'update', 'settings', 'Modify system settings'),
  ('system', 'read',   'roles',    'View system roles'),
  ('system', 'create', 'roles',    'Create system roles'),
  ('system', 'update', 'roles',    'Update system roles'),
  ('system', 'delete', 'roles',    'Delete system roles'),
  ('system', 'create', 'ip_blocks','Block an IP address'),
  ('system', 'delete', 'ip_blocks','Unblock an IP address'),
  ('system', 'read',   'logs',     'View system activity logs'),
  ('billing','read',   'all',      'View all billing records'),
  ('billing','update', 'all',      'Modify billing records')
ON CONFLICT (scope, action, resource) DO NOTHING;

-- Grant ALL permissions to the Administrator role
INSERT INTO permissions_system_role (role_id, permission_id)
SELECT sr.id, p.id
FROM system_roles sr
CROSS JOIN permissions p
WHERE sr.name = 'Administrator'
ON CONFLICT DO NOTHING;
