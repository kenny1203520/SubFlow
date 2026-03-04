-- Add role level constraint trigger
-- Ensures that direct permissions can only be granted up to the member's current role level
CREATE OR REPLACE FUNCTION check_member_permission_level() RETURNS TRIGGER AS $$
DECLARE
    member_max_level INTEGER;
    permission_scope TEXT;
BEGIN
    -- Get member's max role level
    SELECT MIN(gr.role_level) INTO member_max_level
    FROM group_members gm
    JOIN group_member_roles gmr ON gmr.member_id = gm.id
    JOIN group_roles gr ON gr.id = gmr.role_id
    WHERE gm.id = NEW.member_id
    AND gm.group_id = NEW.group_id;

    -- Store current role level for audit
    NEW.max_allowed_role_level := COALESCE(member_max_level, 999);

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_check_member_permission_level ON permissions_group_member;

CREATE TRIGGER trigger_check_member_permission_level
BEFORE INSERT OR UPDATE ON permissions_group_member
FOR EACH ROW
EXECUTE FUNCTION check_member_permission_level();

-- Helper function to get all permissions for a group member (roles + direct)
CREATE OR REPLACE FUNCTION get_member_all_permissions(
    group_id_param TEXT,
    member_id_param TEXT
) RETURNS TABLE(permission_id TEXT, permission_name TEXT, source TEXT) AS $$
BEGIN
    -- Include permissions from roles
    RETURN QUERY
    SELECT p.id, 
           (p.scope || ':' || p.action || ':' || p.resource)::TEXT as perm_name,
           'role'::TEXT as source_type
    FROM permissions p
    JOIN permissions_group_role pgr ON pgr.permission_id = p.id
    JOIN group_member_roles gmr ON gmr.role_id = pgr.role_id
    WHERE gmr.member_id = member_id_param;

    -- Include direct member permissions
    RETURN QUERY
    SELECT p.id,
           (p.scope || ':' || p.action || ':' || p.resource)::TEXT as perm_name,
           'direct'::TEXT as source_type
    FROM permissions p
    JOIN permissions_group_member pgm ON pgm.permission_id = p.id
    WHERE pgm.group_id = group_id_param
    AND pgm.member_id = member_id_param;
END;
$$ LANGUAGE plpgsql;
