-- Roles (Dynamic definitions)
CREATE TABLE IF NOT EXISTS roles (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    -- e.g., 'super_admin', 'group_owner', 'treasurer'
    description TEXT,
    is_system_role BOOLEAN DEFAULT FALSE,
    -- Cannot be deleted if true
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
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
    UNIQUE(scope, action, resource)
);
-- RolePermissions (Many-to-Many)
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id TEXT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);
-- Group Member Roles (Replacing simple role column in group_members if needed, or enhancing it)
-- We will keep group_members.role as a "Primary Role" for simplicity in code, 
-- but this table allows assigning custom roles to members.
CREATE TABLE IF NOT EXISTS group_member_roles (
    member_id TEXT NOT NULL REFERENCES group_members(id) ON DELETE CASCADE,
    role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    assigned_by TEXT REFERENCES users(id),
    PRIMARY KEY (member_id, role_id)
);
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
-- Triggers
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM pg_trigger
    WHERE tgname = 'update_roles_timestamp'
) THEN CREATE TRIGGER update_roles_timestamp BEFORE
UPDATE ON roles FOR EACH ROW EXECUTE FUNCTION update_timestamp();
END IF;
IF NOT EXISTS (
    SELECT 1
    FROM pg_trigger
    WHERE tgname = 'update_member_service_status_timestamp'
) THEN CREATE TRIGGER update_member_service_status_timestamp BEFORE
UPDATE ON member_service_status FOR EACH ROW EXECUTE FUNCTION update_timestamp();
END IF;
END $$;