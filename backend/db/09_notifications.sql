-- Group Invites
CREATE TABLE IF NOT EXISTS group_invites (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    code TEXT NOT NULL UNIQUE,
    -- Invite code
    max_uses INTEGER DEFAULT 1,
    used_count INTEGER DEFAULT 0,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_by TEXT REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_usage CHECK (used_count <= max_uses)
);

-- System Notifications
CREATE TABLE IF NOT EXISTS notifications (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    -- 'payment_due', 'payment_received', 'invite', 'system', 'security'
    
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    data JSONB,
    -- Extra data for frontend navigation (e.g., { group_id: "..." })
    is_read BOOLEAN DEFAULT FALSE,
    is_solved BOOLEAN DEFAULT FALSE,
    is_deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Triggers
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM pg_trigger
    WHERE tgname = 'update_group_invites_timestamp'
) THEN CREATE TRIGGER update_group_invites_timestamp BEFORE
UPDATE ON group_invites FOR EACH ROW EXECUTE FUNCTION update_timestamp();
END IF;
END $$;