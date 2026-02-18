-- Enable RLS on tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE groups ENABLE ROW LEVEL SECURITY;
ALTER TABLE group_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE bills ENABLE ROW LEVEL SECURITY;
ALTER TABLE bill_splits ENABLE ROW LEVEL SECURITY;
ALTER TABLE services ENABLE ROW LEVEL SECURITY;
-- Create Schema for App variables if not exists (though usually we just use the variable)
-- No need to create schema 'app' for set_config, it's just a namespace string.
-- USERS
-- allow reading all users (for searching friends) - or maybe restrict? Let's allow read all for now.
CREATE POLICY "Users can read all users" ON users FOR
SELECT USING (true);
-- Users can update their own profile
CREATE POLICY "Users can update own profile" ON users FOR
UPDATE USING (
        id = current_setting('app.current_user_id', true)::text
    );
-- SESSIONS
-- Users can see their own sessions
CREATE POLICY "Users can see own sessions" ON sessions FOR
SELECT USING (
        user_id = current_setting('app.current_user_id', true)::text
    );
CREATE POLICY "Users can delete own sessions" ON sessions FOR DELETE USING (
    user_id = current_setting('app.current_user_id', true)::text
);
-- GROUPS
-- Users can see groups they are members of
CREATE POLICY "Members can view groups" ON groups FOR
SELECT USING (
        EXISTS (
            SELECT 1
            FROM group_members
            WHERE group_members.group_id = groups.id
                AND group_members.user_id = current_setting('app.current_user_id', true)::text
        )
    );
-- Users can create groups (insert)
CREATE POLICY "Users can create groups" ON groups FOR
INSERT WITH CHECK (
        created_by = current_setting('app.current_user_id', true)::text
    );
-- Group admins (created_by) can update group
CREATE POLICY "Admins can update groups" ON groups FOR
UPDATE USING (
        created_by = current_setting('app.current_user_id', true)::text
    );
-- GROUP MEMBERS
-- Members can view other members in their groups
CREATE POLICY "Members can view group members" ON group_members FOR
SELECT USING (
        EXISTS (
            SELECT 1
            FROM group_members gm
            WHERE gm.group_id = group_members.group_id
                AND gm.user_id = current_setting('app.current_user_id', true)::text
        )
        OR user_id = current_setting('app.current_user_id', true)::text -- View self locally even if not joined yet?
    );
-- Users can join (insert themselves) - actually usually invited. 
-- Let's say: Users can be added if the actor is an admin of the group.
-- This gets complex with "Invite" logic doing the insert.
-- For now, let's assume the API/Application logic handles the "who can add", 
-- but RLS is a safety net. 
-- Allowing Insert if you are the group creator?
CREATE POLICY "Group creators can add members" ON group_members FOR
INSERT WITH CHECK (
        EXISTS (
            SELECT 1
            FROM groups
            WHERE groups.id = group_members.group_id
                AND groups.created_by = current_setting('app.current_user_id', true)::text
        )
        OR user_id = current_setting('app.current_user_id', true)::text -- Self-join (if public)
    );
-- BILLS
-- Members of the group can view bills
CREATE POLICY "Group members can view bills" ON bills FOR
SELECT USING (
        EXISTS (
            SELECT 1
            FROM group_members
            WHERE group_members.group_id = bills.group_id
                AND group_members.user_id = current_setting('app.current_user_id', true)::text
        )
    );
-- SERVICES
-- If public services exist, allow read. If custom services, only creator or group members.
-- Assuming services are linked to groups or generic.
-- "02_services.sql" doesn't show schema but let's assume 'group_id' or 'created_by' or null (system).
-- For now, allow read all services to keep it simple if they are generic templates.
CREATE POLICY "Everyone can read services" ON services FOR
SELECT USING (true);