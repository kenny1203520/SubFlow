-- Create functions to check role hierarchy

-- Function to get a user's highest role level in a specific group
CREATE OR REPLACE FUNCTION get_user_max_role_level_in_group(
    user_id_param TEXT,
    group_id_param TEXT
) RETURNS INTEGER AS $$
DECLARE
    max_level INTEGER;
BEGIN
    SELECT MIN(gr.role_level) INTO max_level
    FROM group_members gm
    JOIN group_member_roles gmr ON gmr.member_id = gm.id
    JOIN group_roles gr ON gr.id = gmr.role_id
    WHERE gm.user_id = user_id_param
    AND gm.group_id = group_id_param;
    
    -- Return a very high number(no permission) if user has no roles
    RETURN COALESCE(max_level, 999);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to get a user's highest system role level
CREATE OR REPLACE FUNCTION get_user_max_system_role_level(user_id_param TEXT) RETURNS INTEGER AS $$
DECLARE
    max_level INTEGER;
BEGIN
    SELECT MIN(sr.role_level) INTO max_level
    FROM user_roles ur
    JOIN system_roles sr ON sr.id = ur.role_id
    WHERE ur.user_id = user_id_param;
    
    -- Return a very high number (no permission) if user has no system roles
    RETURN COALESCE(max_level, 999);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to check if user can manage target role in a group
-- Returns true if:
-- 1. User's highest group role level is strictly lower than target role's level (can only manage lower levels)
-- 2. User is Group Owner or can manage roles at target's level
CREATE OR REPLACE FUNCTION can_manage_role_in_group(
    actor_user_id TEXT,
    actor_group_id TEXT,
    target_role_id TEXT
) RETURNS BOOLEAN AS $$
DECLARE
    actor_max_level INTEGER;
    target_level INTEGER;
    group_creator TEXT;
BEGIN
    -- Get actor's max role level in group (minimum value = highest actual privilege)
    actor_max_level := get_user_max_role_level_in_group(actor_user_id, actor_group_id);
    
    -- Get target role's level
    SELECT role_level INTO target_level
    FROM group_roles
    WHERE id = target_role_id;
    
    -- Get group creator
    SELECT created_by INTO group_creator
    FROM groups
    WHERE id = actor_group_id;
    
    -- Group creator (owner) can always manage roles
    IF actor_user_id = group_creator THEN
        RETURN TRUE;
    END IF;
    
    -- Actor must have strictly lower level (higher privilege) than target role
    -- Lower numeric value = higher privilege
    RETURN actor_max_level < target_level;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to check if user can assign a specific role to another member
CREATE OR REPLACE FUNCTION can_assign_role_to_member(
    actor_user_id TEXT,
    group_id TEXT,
    target_role_id TEXT
) RETURNS BOOLEAN AS $$
BEGIN
    -- Must have manage roles permission AND target role must be below actor's level
    RETURN can_manage_group_members(actor_user_id, group_id)
    AND can_manage_role_in_group(actor_user_id, group_id, target_role_id);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to check if user can remove a specific role from a member
CREATE OR REPLACE FUNCTION can_remove_role_from_member(
    actor_user_id TEXT,
    group_id TEXT,
    target_role_id TEXT
) RETURNS BOOLEAN AS $$
BEGIN
    -- Must have manage members permission AND target role must be below actor's level
    RETURN can_manage_group_members(actor_user_id, group_id)
    AND can_manage_role_in_group(actor_user_id, group_id, target_role_id);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
