-- Audit Logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT REFERENCES users(id),
    target_type TEXT NOT NULL,
    -- 'group', 'bill', 'member'
    target_id TEXT NOT NULL,
    action TEXT NOT NULL,
    -- 'create', 'update', 'delete'
    changes JSONB,
    -- Store before/after values
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_timestamp() RETURNS TRIGGER AS $$ BEGIN NEW.updated_at = NOW();
RETURN NEW;
END;
$$ language 'plpgsql';
-- Applying triggers to all tables with updated_at column
DO $$
DECLARE t TEXT;
BEGIN FOR t IN
SELECT table_name
FROM information_schema.columns
WHERE column_name = 'updated_at'
    AND table_schema = 'public' LOOP EXECUTE format(
        'DROP TRIGGER IF EXISTS update_%I_timestamp ON %I',
        t,
        t
    );
EXECUTE format(
    'CREATE TRIGGER update_%I_timestamp BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION update_timestamp()',
    t,
    t
);
END LOOP;
END $$;