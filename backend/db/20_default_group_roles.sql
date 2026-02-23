-- Ensure group role names are unique within each group
ALTER TABLE group_roles DROP CONSTRAINT IF EXISTS group_roles_name_key;
ALTER TABLE group_roles ADD CONSTRAINT group_roles_group_id_name_key UNIQUE (group_id, name);

-- Seed default permissions for group roles

-- Group-specific Permissions (Granular control within a group context)
INSERT INTO permissions (scope, action, resource, description)
VALUES
  -- Group Management
  ('group', 'read',   'details',      'groups.permissions.group.read.details'),
  ('group', 'update', 'settings',     'groups.permissions.group.update.settings'),
  ('group', 'update', 'details',      'groups.permissions.group.update.details'),
  ('group', 'delete', 'group',        'groups.permissions.group.delete.group'),
  ('group', 'transfer', 'ownership',  'groups.permissions.group.transfer.ownership'),
  
  -- Member Management
  ('group', 'read',   'members',      'groups.permissions.group.read.members'),
  ('group', 'invite', 'members',      'groups.permissions.group.invite.members'),
  ('group', 'remove', 'members',      'groups.permissions.group.remove.members'),
  ('group', 'update', 'members',      'groups.permissions.group.update.members'),
  
  -- Role Management
  ('group', 'read',   'roles',        'groups.permissions.group.read.roles'),
  ('group', 'create', 'roles',        'groups.permissions.group.create.roles'),
  ('group', 'update', 'roles',        'groups.permissions.group.update.roles'),
  ('group', 'delete', 'roles',        'groups.permissions.group.delete.roles'),
  ('group', 'assign', 'roles',        'groups.permissions.group.assign.roles'),
  ('group', 'remove', 'role_assignment', 'groups.permissions.group.remove.role_assignment'),
  
  -- Permission Management
  ('group', 'read',   'permissions',  'groups.permissions.group.read.permissions'),
  ('group', 'grant',  'permissions',  'groups.permissions.group.grant.permissions'),
  ('group', 'revoke', 'permissions',  'groups.permissions.group.revoke.permissions'),
  
  -- Service Management
  ('group', 'read',   'services',     'groups.permissions.group.read.services'),
  ('group', 'create', 'services',     'groups.permissions.group.create.services'),
  ('group', 'update', 'services',     'groups.permissions.group.update.services'),
  ('group', 'delete', 'services',     'groups.permissions.group.delete.services'),
  
  -- Subscription Management
  ('group', 'read',   'subscriptions', 'groups.permissions.group.read.subscriptions'),
  ('group', 'create', 'subscriptions', 'groups.permissions.group.create.subscriptions'),
  ('group', 'update', 'subscriptions', 'groups.permissions.group.update.subscriptions'),
  ('group', 'delete', 'subscriptions', 'groups.permissions.group.delete.subscriptions'),
  ('group', 'manage', 'billing',       'groups.permissions.group.manage.billing'),
  
  -- Expense Management
  ('group', 'read',   'expenses',     'groups.permissions.group.read.expenses'),
  ('group', 'create', 'expenses',     'groups.permissions.group.create.expenses'),
  ('group', 'update', 'expenses',     'groups.permissions.group.update.expenses'),
  ('group', 'delete', 'expenses',     'groups.permissions.group.delete.expenses'),
  ('group', 'settle', 'expenses',     'groups.permissions.group.settle.expenses'),
  
  -- Financial Management
  ('group', 'read',   'finance',      'groups.permissions.group.read.finance'),
  ('group', 'manage', 'finance',      'groups.permissions.group.manage.finance'),
  ('group', 'export', 'finance',      'groups.permissions.group.export.finance'),
  
  -- File Management
  ('group', 'read',   'files',        'groups.permissions.group.read.files'),
  ('group', 'upload', 'files',        'groups.permissions.group.upload.files'),
  ('group', 'delete', 'files',        'groups.permissions.group.delete.files')
ON CONFLICT (scope, action, resource) DO UPDATE SET description = EXCLUDED.description;
