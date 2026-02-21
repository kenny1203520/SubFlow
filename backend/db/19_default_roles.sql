-- Insert Default System Roles
INSERT INTO system_roles (name, description, is_system_role)
VALUES 
  ('Administrator', 'Full access to all system resources - has every permission automatically', true),
  ('Support Agent', 'Can view user accounts and revoke sessions, but cannot modify security settings or bans', true),
  ('User', 'Can view user accounts and revoke sessions, but cannot modify security settings or bans', true),
  ('Guest', 'Can view user accounts and revoke sessions, but cannot modify security settings or bans', true)
ON CONFLICT (name) DO NOTHING;

-- Seed default permissions for all scopes

-- System Permissions
INSERT INTO permissions (scope, action, resource, description)
VALUES
  ('system', 'read',   'stats',    'admin.permissions.system.read.stats'),
  ('system', 'create', 'users',    'admin.permissions.system.create.users'),
  ('system', 'read',   'users',    'admin.permissions.system.read.users'),
  ('system', 'update', 'users',    'admin.permissions.system.update.users'),
  ('system', 'delete', 'users',    'admin.permissions.system.delete.users'),
  ('system', 'read',   'sessions', 'admin.permissions.system.read.sessions'),
  ('system', 'delete', 'sessions', 'admin.permissions.system.delete.sessions'),
  ('system', 'read',   'settings', 'admin.permissions.system.read.settings'),
  ('system', 'update', 'settings', 'admin.permissions.system.update.settings'),
  ('system', 'create', 'roles',    'admin.permissions.system.create.roles'),
  ('system', 'read',   'roles',    'admin.permissions.system.read.roles'),
  ('system', 'update', 'roles',    'admin.permissions.system.update.roles'),
  ('system', 'delete', 'roles',    'admin.permissions.system.delete.roles'),
  ('system', 'create', 'ip_blocks','admin.permissions.system.create.ip_blocks'),
  ('system', 'read',   'ip_blocks','admin.permissions.system.read.ip_blocks'),
  ('system', 'delete', 'ip_blocks','admin.permissions.system.delete.ip_blocks'),
  ('system', 'read',   'logs',     'admin.permissions.system.read.logs'),
  ('system', 'manage', 'roles',            'admin.permissions.system.manage.roles'),
  ('system', 'manage', 'user_roles',       'admin.permissions.system.manage.user_roles'),
  ('system', 'manage', 'permissions_user', 'admin.permissions.system.manage.permissions_user')
ON CONFLICT (scope, action, resource) DO UPDATE SET description = EXCLUDED.description;

-- Billing Permissions
INSERT INTO permissions (scope, action, resource, description)
VALUES
  ('billing','read',   'all',      'admin.permissions.billing.read.all'),
  ('billing','update', 'all',      'admin.permissions.billing.update.all')
ON CONFLICT (scope, action, resource) DO UPDATE SET description = EXCLUDED.description;

-- Grant ALL permissions to the Administrator role
INSERT INTO permissions_system_role (role_id, permission_id)
SELECT sr.id, p.id
FROM system_roles sr
CROSS JOIN permissions p
WHERE sr.name = 'Administrator'
ON CONFLICT DO NOTHING;
