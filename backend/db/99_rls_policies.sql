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
-- Group creators/admins can update group
CREATE POLICY "Admins can update groups" ON groups FOR UPDATE USING (
    EXISTS (
        SELECT 1 FROM group_members
        WHERE group_members.group_id = groups.id
        AND group_members.user_id = current_setting('app.current_user_id', true)::text
        AND group_members.role IN ('owner', 'admin')
    )
    OR created_by = current_setting('app.current_user_id', true)::text
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

-- Group admins can manage services (Insert/Update/Delete)
CREATE POLICY "Group admins can manage services" ON group_services FOR ALL USING (
    EXISTS (
        SELECT 1 FROM group_members
        WHERE group_members.group_id = group_services.group_id
        AND group_members.user_id = current_setting('app.current_user_id', true)::text
        AND group_members.role IN ('owner', 'admin')
    )
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
-- Let's say: Users can be added if the actor is an admin of the group.
CREATE POLICY "Group admins can add members" ON group_members FOR INSERT WITH CHECK (
    EXISTS (
        SELECT 1 FROM group_members gm
        WHERE gm.group_id = group_members.group_id
        AND gm.user_id = current_setting('app.current_user_id', true)::text
        AND gm.role IN ('owner', 'admin')
    )
    OR 
    EXISTS ( -- Or if creator is adding (for first member)
        SELECT 1 FROM groups
        WHERE groups.id = group_members.group_id
        AND groups.created_by = current_setting('app.current_user_id', true)::text
    )
);
-- Group admins can update/delete members
CREATE POLICY "Group admins can update/delete members" ON group_members FOR ALL USING (
    EXISTS (
        SELECT 1 FROM group_members gm
        WHERE gm.group_id = group_members.group_id
        AND gm.user_id = current_setting('app.current_user_id', true)::text
        AND gm.role IN ('owner', 'admin')
    )
);

-- GROUP SERVICES
ALTER TABLE group_services ENABLE ROW LEVEL SECURITY;

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
        WHERE gs.id = group_service_members.group_service_id
        AND gm.user_id = current_setting('app.current_user_id', true)::text
        AND gm.role IN ('owner', 'admin')
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
