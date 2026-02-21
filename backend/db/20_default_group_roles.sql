-- Insert Default Group Roles
INSERT INTO group_roles (name, description, is_system_role)
VALUES 
    ('Group Owner', 'Full control over the group, including billing and deletion.', true),
    ('Group Admin', 'Can manage group settings and members.', true),
    ('Group Treasurer', 'Can manage expenses and group balances.', true)
ON CONFLICT (name) DO NOTHING;
