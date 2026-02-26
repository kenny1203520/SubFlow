-- System Roles (Global definitions)
CREATE TABLE IF NOT EXISTS system_roles (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    -- e.g., 'super_admin', 'support_agent'
    description TEXT,
    is_system_role BOOLEAN DEFAULT FALSE,
    -- Lower number = higher privilege; default to 999 for custom roles without defined level
    role_level INTEGER NOT NULL DEFAULT 999,
    -- Cannot be deleted if true
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Group Roles (Group-level definitions)
CREATE TABLE IF NOT EXISTS group_roles (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    -- e.g., 'group_owner', 'treasurer', 'auditor'
    description TEXT,
    is_system_role BOOLEAN DEFAULT FALSE,
    -- Lower number = higher privilege; default to 999 for custom roles without defined level
    role_level INTEGER NOT NULL DEFAULT 999,
    -- Cannot be deleted if true
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (group_id, name)
);
-- Index
CREATE INDEX IF NOT EXISTS idx_group_roles_group_id
ON group_roles(group_id);

-- Permissions (Granular actions)
CREATE TABLE IF NOT EXISTS permissions (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    scope TEXT NOT NULL,
    -- 'system', 'group', 'billing'
    action TEXT NOT NULL,
    -- 'create', 'read', 'update', 'delete', 'approve'
    resource TEXT NOT NULL,
    -- 'users', 'bills', 'settings'
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(scope, action, resource)
);

-- System Role Permissions (Many-to-Many)
CREATE TABLE IF NOT EXISTS permissions_system_role (
    role_id TEXT NOT NULL REFERENCES system_roles(id) ON DELETE CASCADE,
    permission_id TEXT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, permission_id)
);
-- Index
CREATE INDEX IF NOT EXISTS idx_permissions_system_role_role
ON permissions_system_role(role_id);
CREATE INDEX IF NOT EXISTS idx_permissions_system_role_permission
ON permissions_system_role(permission_id);

-- Group Role Permissions (Many-to-Many)
CREATE TABLE IF NOT EXISTS permissions_group_role (
    role_id TEXT NOT NULL REFERENCES group_roles(id) ON DELETE CASCADE,
    permission_id TEXT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, permission_id)
);
-- Index
CREATE INDEX IF NOT EXISTS idx_permissions_group_role_role
ON permissions_group_role(role_id);
CREATE INDEX IF NOT EXISTS idx_permissions_group_role_permission
ON permissions_group_role(permission_id);

-- User Permissions (Direct assignment of granular permissions to users)
CREATE TABLE IF NOT EXISTS permissions_user (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission_id TEXT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    granted_by TEXT REFERENCES users(id),
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, permission_id)
);
-- Index
CREATE INDEX IF NOT EXISTS idx_permissions_user_user
ON permissions_user(user_id);
CREATE INDEX IF NOT EXISTS idx_permissions_user_permission
ON permissions_user(permission_id);

-- Group Member Permissions (Direct permission assignment for members)
-- This allows granting specific permissions to members, constrained by their role hierarchy level
CREATE TABLE IF NOT EXISTS permissions_group_member (
    member_id TEXT NOT NULL REFERENCES group_members(id) ON DELETE CASCADE,
    permission_id TEXT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    granted_by TEXT NOT NULL REFERENCES users(id),
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(group_id, member_id, permission_id)
);
-- Index
CREATE INDEX IF NOT EXISTS idx_permissions_group_member_member 
ON permissions_group_member(member_id);
CREATE INDEX IF NOT EXISTS idx_permissions_group_member_permission 
ON permissions_group_member(permission_id);

-- User System Roles (Global role assignments)
CREATE TABLE IF NOT EXISTS user_roles (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id TEXT NOT NULL REFERENCES system_roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    assigned_by TEXT REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id)
);
-- Index
CREATE INDEX IF NOT EXISTS idx_user_roles_user
ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role
ON user_roles(role_id);

-- Group Member Roles (replaces group_members.role; all roles are managed here)
CREATE TABLE IF NOT EXISTS group_member_roles (
    member_id TEXT NOT NULL REFERENCES group_members(id) ON DELETE CASCADE,
    role_id TEXT NOT NULL REFERENCES group_roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    assigned_by TEXT REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (member_id, role_id)
);
-- Index
CREATE INDEX IF NOT EXISTS idx_group_member_roles_member
ON group_member_roles(member_id);
CREATE INDEX IF NOT EXISTS idx_group_member_roles_role
ON group_member_roles(role_id);

-- Member Service Status (Service-specific active periods within a group)
CREATE TABLE IF NOT EXISTS member_service_status (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id TEXT NOT NULL REFERENCES group_members(id) ON DELETE CASCADE,
    service_id TEXT REFERENCES services(id),
    -- Specific service in the group
    status TEXT CHECK (
        status IN ('active', 'paused', 'scheduled_stop', 'ended')
    ) DEFAULT 'active',
    start_date DATE NOT NULL DEFAULT CURRENT_DATE,
    end_date DATE,
    -- If scheduled_stop or ended
    -- For separate nickname in this specific service context (optional, usually group level is enough)
    -- But user asked for: "群組中的使用者可以存在一個群組內的單獨暱稱" -> This is on group_members
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Triggers for 'updated_at'
DO $$ BEGIN
    -- System Roles
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_system_roles_timestamp'
    ) THEN
        CREATE TRIGGER update_system_roles_timestamp
        BEFORE UPDATE ON system_roles
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- Group Roles
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_group_roles_timestamp'
    ) THEN
        CREATE TRIGGER update_group_roles_timestamp
        BEFORE UPDATE ON group_roles
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- Permissions
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_permissions_timestamp'
    ) THEN
        CREATE TRIGGER update_permissions_timestamp
        BEFORE UPDATE ON permissions
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- Permissions System Role
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_permissions_system_role_timestamp'
    ) THEN
        CREATE TRIGGER update_permissions_system_role_timestamp
        BEFORE UPDATE ON permissions_system_role
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- Permissions Group Role
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_permissions_group_role_timestamp'
    ) THEN
        CREATE TRIGGER update_permissions_group_role_timestamp
        BEFORE UPDATE ON permissions_group_role
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- Permissions User
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_permissions_user_timestamp'
    ) THEN
        CREATE TRIGGER update_permissions_user_timestamp
        BEFORE UPDATE ON permissions_user
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- User Roles
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_user_roles_timestamp'
    ) THEN
        CREATE TRIGGER update_user_roles_timestamp
        BEFORE UPDATE ON user_roles
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- Group Member Roles
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_group_member_roles_timestamp'
    ) THEN
        CREATE TRIGGER update_group_member_roles_timestamp
        BEFORE UPDATE ON group_member_roles
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- Member Service Status
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_member_service_status_timestamp'
    ) THEN
        CREATE TRIGGER update_member_service_status_timestamp
        BEFORE UPDATE ON member_service_status
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;
END $$;