-- API Keys (Personal Access Tokens)
CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    -- e.g. "Mobile App", "GitHub Action"
    key_hash TEXT NOT NULL UNIQUE,
    -- SHA256 hash of the transparent key
    key_prefix TEXT NOT NULL,
    -- First few chars for identification (e.g. "sf_...")
    scopes TEXT [] NOT NULL DEFAULT '{}',
    -- ['read:groups', 'write:bills']
    is_active BOOLEAN DEFAULT TRUE,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    last_used_ip TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Index for fast lookup by hash
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);

-- Trigger for 'updated_at'
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM pg_trigger
    WHERE tgname = 'update_api_keys_timestamp'
) THEN CREATE TRIGGER update_api_keys_timestamp BEFORE
UPDATE ON api_keys FOR EACH ROW EXECUTE FUNCTION update_timestamp();
END IF;
END $$;