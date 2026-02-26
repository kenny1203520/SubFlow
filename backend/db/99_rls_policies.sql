-- Helper Functions for Flexible Permission Checking
-- These functions check both role-based and direct permission assignments

CREATE OR REPLACE FUNCTION can_manage_group(user_id_param TEXT, group_id_param TEXT) RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        -- Check if user is Group Owner or Admin (via roles)
        SELECT 1 FROM group_members gm
        JOIN group_member_roles gmr ON gmr.member_id = gm.id
        JOIN group_roles gr ON gr.id = gmr.role_id
        WHERE gm.group_id = group_id_param
        AND gm.user_id = user_id_param
        AND gr.name IN ('Group Owner', 'Group Admin')
    )
    OR 
    -- Check if user is the group creator
    EXISTS (
        SELECT 1 FROM groups
        WHERE groups.id = group_id_param
        AND groups.created_by = user_id_param
    )
    OR
    -- Check if user has group:update permission through their roles
    EXISTS (
        SELECT 1 FROM group_members gm
        JOIN group_member_roles gmr ON gmr.member_id = gm.id
        JOIN permissions_group_role pgr ON pgr.role_id = gmr.role_id
        JOIN permissions p ON pgr.permission_id = p.id
        WHERE gm.group_id = group_id_param
        AND gm.user_id = user_id_param
        AND p.scope = 'group'
        AND (p.action IN ('update', 'manage', 'admin') OR p.resource IN ('group', 'settings', '*'))
    );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE OR REPLACE FUNCTION can_manage_group_members(user_id_param TEXT, group_id_param TEXT) RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        -- Check if user is Group Owner or Admin (via roles)
        SELECT 1 FROM group_members gm
        JOIN group_member_roles gmr ON gmr.member_id = gm.id
        JOIN group_roles gr ON gr.id = gmr.role_id
        WHERE gm.group_id = group_id_param
        AND gm.user_id = user_id_param
        AND gr.name IN ('Group Owner', 'Group Admin')
    )
    OR 
    -- Check if user is the group creator
    EXISTS (
        SELECT 1 FROM groups
        WHERE groups.id = group_id_param
        AND groups.created_by = user_id_param
    )
    OR
    -- Check if user has permissions to manage members through their roles
    EXISTS (
        SELECT 1 FROM group_members gm
        JOIN group_member_roles gmr ON gmr.member_id = gm.id
        JOIN permissions_group_role pgr ON pgr.role_id = gmr.role_id
        JOIN permissions p ON pgr.permission_id = p.id
        WHERE gm.group_id = group_id_param
        AND gm.user_id = user_id_param
        AND p.scope = 'group'
        AND (p.action IN ('update', 'manage', 'admin') OR p.resource IN ('members', 'group', '*'))
    );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- USER SECURITY
ALTER TABLE user_security ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Users can view own security settings" ON user_security FOR SELECT USING (user_id = current_setting('app.current_user_id', true)::text);
CREATE POLICY "Users can update own security settings" ON user_security FOR UPDATE USING (user_id = current_setting('app.current_user_id', true)::text);
CREATE POLICY "Users can insert own security settings" ON user_security FOR INSERT WITH CHECK (user_id = current_setting('app.current_user_id', true)::text);

-- USER DEVICES
ALTER TABLE user_devices ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Users can view own devices" ON user_devices FOR SELECT USING (user_id = current_setting('app.current_user_id', true)::text);
CREATE POLICY "Users can manage own devices" ON user_devices FOR ALL USING (user_id = current_setting('app.current_user_id', true)::text);

-- LOGIN HISTORY
ALTER TABLE login_history ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Users can view own login history" ON login_history FOR SELECT USING (user_id = current_setting('app.current_user_id', true)::text);
-- Insert is usually done by system (bypassing RLS or as admin), but if we want strictly:
-- CREATE POLICY "System can insert login history" ... (depends on how we handle system role)


-- Create Schema for App variables if not exists (though usually we just use the variable)
-- No need to create schema 'app' for set_config, it's just a namespace string.

-- USERS
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- allow reading all users (for searching friends) - or maybe restrict? Let's allow read all for now.
CREATE POLICY "Users can read all users" ON users FOR SELECT USING (true);
-- Users can update their own profile
CREATE POLICY "Users can update own profile" ON users FOR UPDATE USING (id = current_setting('app.current_user_id', true)::text);


-- SESSIONS
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
-- Users can see their own sessions
CREATE POLICY "Users can see own sessions" ON sessions FOR SELECT USING (user_id = current_setting('app.current_user_id', true)::text);
CREATE POLICY "Users can delete own sessions" ON sessions FOR DELETE USING (user_id = current_setting('app.current_user_id', true)::text);

-- GROUPS
ALTER TABLE groups ENABLE ROW LEVEL SECURITY;
-- Users can see groups they are members of
CREATE POLICY "Members can view groups" ON groups FOR SELECT USING (
    EXISTS (
        SELECT 1 FROM group_members 
        WHERE group_members.group_id = groups.id 
        AND group_members.user_id = current_setting('app.current_user_id', true)::text
    )
    OR created_by = current_setting('app.current_user_id', true)::text
);
-- Users can create groups (insert)
CREATE POLICY "Users can create groups" ON groups FOR INSERT WITH CHECK (created_by = current_setting('app.current_user_id', true)::text);
-- Group creators/admins can update group - now uses helper function for flexible permissions
CREATE POLICY "Admins can update groups" ON groups FOR UPDATE USING (
    can_manage_group(current_setting('app.current_user_id', true)::text, id)
);

-- GROUP SERVICES
-- Members of the group can view services
CREATE POLICY "Group members can view services" ON group_services FOR SELECT USING (
    EXISTS (
        SELECT 1 FROM group_members 
        WHERE group_members.group_id = group_services.group_id 
        AND group_members.user_id = current_setting('app.current_user_id', true)::text
    )
);

-- Group admins can manage services (Insert/Update/Delete) - now uses helper function
CREATE POLICY "Group admins can manage services" ON group_services FOR ALL USING (
    can_manage_group(current_setting('app.current_user_id', true)::text, group_id)
);

-- GROUP MEMBERS
ALTER TABLE group_members ENABLE ROW LEVEL SECURITY;
-- Members can view other members in their groups
CREATE POLICY "Members can view group members" ON group_members FOR SELECT USING (
    EXISTS (
        SELECT 1 FROM group_members gm 
        WHERE gm.group_id = group_members.group_id 
        AND gm.user_id = current_setting('app.current_user_id', true)::text
    )
    OR user_id = current_setting('app.current_user_id', true)::text
);
-- Users can join (insert themselves) - actually usually invited. 
-- Let's say: Users can be added if the actor is an admin of the group or has proper permissions
CREATE POLICY "Group admins can add members" ON group_members FOR INSERT WITH CHECK (
    can_manage_group_members(current_setting('app.current_user_id', true)::text, group_id)
    OR 
    EXISTS ( -- Or if creator is adding (for first member)
        SELECT 1 FROM groups
        WHERE groups.id = group_members.group_id
        AND groups.created_by = current_setting('app.current_user_id', true)::text
    )
);
-- Group admins can update/delete members - now uses helper function
CREATE POLICY "Group admins can update/delete members" ON group_members FOR ALL USING (
    can_manage_group_members(current_setting('app.current_user_id', true)::text, group_id)
);

-- GROUP SERVICES
ALTER TABLE group_services ENABLE ROW LEVEL SECURITY;

-- GROUP MEMBER ROLES (for role assignments)
ALTER TABLE group_member_roles ENABLE ROW LEVEL SECURITY;
-- Members can view their own roles in the group
CREATE POLICY "Members can view group member roles" ON group_member_roles FOR SELECT USING (
    EXISTS (
        SELECT 1 FROM group_members gm
        WHERE gm.id = group_member_roles.member_id
        AND EXISTS (
            SELECT 1 FROM group_members gm2
            WHERE gm2.group_id = gm.group_id
            AND gm2.user_id = current_setting('app.current_user_id', true)::text
        )
    )
);
-- Only admins or those with proper role assignment permissions can modify roles
CREATE POLICY "Admins can manage member roles" ON group_member_roles FOR ALL USING (
    EXISTS (
        SELECT 1 FROM group_members gm_member
        JOIN group_member_roles gmr ON gmr.member_id = gm_member.id
        WHERE gm_member.group_id = (
            SELECT gm.group_id FROM group_members gm WHERE gm.id = group_member_roles.member_id LIMIT 1
        )
        AND gm_member.user_id = current_setting('app.current_user_id', true)::text
        AND can_manage_group(gm_member.user_id, gm_member.group_id)
    )
);

-- GROUP SERVICES

-- GROUP SERVICE MEMBERS
ALTER TABLE group_service_members ENABLE ROW LEVEL SECURITY;
-- Visible to group members
CREATE POLICY "Group members can view service splits" ON group_service_members FOR SELECT USING (
    EXISTS (
        SELECT 1 FROM group_services gs
        JOIN group_members gm ON gs.group_id = gm.group_id
        WHERE gs.id = group_service_members.group_service_id
        AND gm.user_id = current_setting('app.current_user_id', true)::text
    )
);
-- Admins can manage splits
CREATE POLICY "Group admins can manage service splits" ON group_service_members FOR ALL USING (
    EXISTS (
        SELECT 1 FROM group_services gs
        JOIN group_members gm ON gs.group_id = gm.group_id
        JOIN group_member_roles gmr ON gmr.member_id = gm.id
        JOIN group_roles gr ON gr.id = gmr.role_id
        WHERE gs.id = group_service_members.group_service_id
        AND gm.user_id = current_setting('app.current_user_id', true)::text
        AND gr.name IN ('Group Owner', 'Group Admin')
    )
);

-- BILLS
ALTER TABLE bills ENABLE ROW LEVEL SECURITY;
-- Members of the group can view bills
CREATE POLICY "Group members can view bills" ON bills FOR SELECT USING (
    EXISTS (
        SELECT 1 FROM group_members 
        WHERE group_members.group_id = bills.group_id 
        AND group_members.user_id = current_setting('app.current_user_id', true)::text
    )
);

-- Bill splits
ALTER TABLE bill_splits ENABLE ROW LEVEL SECURITY;

-- SERVICES
ALTER TABLE services ENABLE ROW LEVEL SECURITY;
-- Everyone can read services
CREATE POLICY "Everyone can read services" ON services FOR SELECT USING (true);
