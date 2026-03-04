-- Insert Default System Roles
INSERT INTO system_roles (name, description, is_system_role, role_level)
VALUES 
  ('Administrator', 'Full access to all system resources - has every permission automatically', true, 1),
  ('Support Agent', 'Can view user accounts and revoke sessions, but cannot modify security settings or bans', true, 5),
  ('User', 'Can view user accounts and revoke sessions, but cannot modify security settings or bans', true, 10),
  ('Guest', 'Can view user accounts and revoke sessions, but cannot modify security settings or bans', true, 50)
ON CONFLICT (name) DO NOTHING;

-- Seed default permissions for all scopes

-- System Permissions
INSERT INTO permissions (scope, action, resource, description)
VALUES
  ('system', 'read',   'admin',    'admin.permissions.system.read.admin'),
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
  ('system', 'manage', 'permissions_user', 'admin.permissions.system.manage.permissions_user'),

-- Audit Permissions
  ('audit', 'read',   'user_activity', 'audit.permissions.audit.read.user_activity'),

-- Users Permissions
  ('users', 'read',   'profile', 'users.permissions.users.read.profile'),
  ('users', 'update', 'profile', 'users.permissions.users.update.profile'),

-- Billing Permissions
  ('billing','read',   'all',      'billing.permissions.billing.read.all'),
  ('billing','update', 'all',      'billing.permissions.billing.update.all'),

-- Group Permissions
  ('groups', 'create', 'groups',   'groups.permissions.groups.create.groups'),
  ('groups', 'query',  'groups',   'groups.permissions.groups.query.groups'),
  ('groups', 'read',   'groups',   'groups.permissions.groups.read.groups'),
  ('groups', 'update', 'groups',   'groups.permissions.groups.update.groups'),
  ('groups', 'delete', 'groups',   'groups.permissions.groups.delete.groups'),
  ('groups', 'create', 'expenses', 'groups.permissions.groups.create.expenses'),
  ('groups', 'read',   'expenses', 'groups.permissions.groups.read.expenses'),
  ('groups', 'update', 'expenses', 'groups.permissions.groups.update.expenses'),
  ('groups', 'delete', 'expenses', 'groups.permissions.groups.delete.expenses'),

-- User Permissions
  ('user', 'read',   'profile', 'user.permissions.user.read.profile'),
  ('user', 'update', 'profile', 'user.permissions.user.update.profile'),
  ('user', 'upload', 'avatar', 'user.permissions.user.upload.avatar'),
  ('user', 'upload', 'file',   'user.permissions.user.upload.file')
ON CONFLICT (scope, action, resource) DO UPDATE SET description = EXCLUDED.description;

-- Grant ALL permissions to the Administrator role
INSERT INTO permissions_system_role (role_id, permission_id)
SELECT sr.id, p.id
FROM system_roles sr
CROSS JOIN permissions p
WHERE sr.name = 'Administrator'
ON CONFLICT DO NOTHING;

-- Grant permissions to Support Agent role
INSERT INTO permissions_system_role (role_id, permission_id)
SELECT sr.id, p.id
FROM system_roles sr
CROSS JOIN permissions p
WHERE sr.name = 'Support Agent'
AND (
  (p.scope = 'system' AND p.action = 'read' AND p.resource IN ('stats', 'users', 'sessions', 'roles', 'logs')) OR
  (p.scope = 'system' AND p.action = 'delete' AND p.resource = 'sessions') OR
  (p.scope = 'audit' AND p.action = 'read' AND p.resource = 'user_activity') OR
  (p.scope = 'billing' AND p.action = 'read' AND p.resource = 'all') OR
  (p.scope = 'groups' AND p.action = 'read' AND p.resource IN ('groups', 'expenses'))
)
ON CONFLICT DO NOTHING;

-- Grant permissions to User role
INSERT INTO permissions_system_role (role_id, permission_id)
SELECT sr.id, p.id
FROM system_roles sr
CROSS JOIN permissions p
WHERE sr.name = 'User'
AND (
  (p.scope = 'groups') OR
  (p.scope = 'user' AND p.action IN ('read', 'update', 'upload'))
)
ON CONFLICT DO NOTHING;

-- Grant permissions to Guest role
INSERT INTO permissions_system_role (role_id, permission_id)
SELECT sr.id, p.id
FROM system_roles sr
CROSS JOIN permissions p
WHERE sr.name = 'Guest'
AND (
  (p.scope = 'user' AND p.action IN ('read', 'update') AND p.resource = 'profile') OR
  (p.scope = 'groups' AND p.action = 'query' AND p.resource = 'groups')
)
ON CONFLICT DO NOTHING;