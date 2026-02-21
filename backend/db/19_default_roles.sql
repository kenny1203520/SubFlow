-- Insert Default System Roles
INSERT INTO system_roles (name, description, is_system_role)
VALUES 
  ('Administrator', 'Full access to all system resources - has every permission automatically', true),
  ('Support Agent', 'Can view user accounts and revoke sessions, but cannot modify security settings or bans', true)
ON CONFLICT (name) DO NOTHING;

-- Seed default permissions for all scopes
INSERT INTO permissions (scope, action, resource, description)
VALUES
  ('system', 'read',   'stats',    'permissions.system.read.stats'),
  ('system', 'create', 'users',    'permissions.system.create.users'),
  ('system', 'read',   'users',    'permissions.system.read.users'),
  ('system', 'update', 'users',    'permissions.system.update.users'),
  ('system', 'delete', 'users',    'permissions.system.delete.users'),
  ('system', 'read',   'sessions', 'permissions.system.read.sessions'),
  ('system', 'delete', 'sessions', 'permissions.system.delete.sessions'),
  ('system', 'read',   'settings', 'permissions.system.read.settings'),
  ('system', 'update', 'settings', 'permissions.system.update.settings'),
  ('system', 'create', 'roles',    'permissions.system.create.roles'),
  ('system', 'read',   'roles',    'permissions.system.read.roles'),
  ('system', 'update', 'roles',    'permissions.system.update.roles'),
  ('system', 'delete', 'roles',    'permissions.system.delete.roles'),
  ('system', 'create', 'ip_blocks','permissions.system.create.ip_blocks'),
  ('system', 'read',   'ip_blocks','permissions.system.read.ip_blocks'),
  ('system', 'delete', 'ip_blocks','permissions.system.delete.ip_blocks'),
  ('system', 'read',   'logs',     'permissions.system.read.logs'),
  ('system', 'manage', 'roles',            'permissions.system.manage.roles'),
  ('system', 'manage', 'user_roles',       'permissions.system.manage.user_roles'),
  ('system', 'manage', 'permissions_user', 'permissions.system.manage.permissions_user'),
  ('billing','read',   'all',      'permissions.billing.read.all'),
  ('billing','update', 'all',      'permissions.billing.update.all')
ON CONFLICT (scope, action, resource) DO NOTHING;

-- Grant ALL permissions to the Administrator role
INSERT INTO permissions_system_role (role_id, permission_id)
SELECT sr.id, p.id
FROM system_roles sr
CROSS JOIN permissions p
WHERE sr.name = 'Administrator'
ON CONFLICT DO NOTHING;
