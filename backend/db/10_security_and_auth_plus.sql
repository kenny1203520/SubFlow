-- User Security Settings
CREATE TABLE IF NOT EXISTS user_security (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    two_factor_secret TEXT,
    -- Encrypted TOTP secret
    backup_codes TEXT [],
    -- Hashed backup codes
    -- Account Status
    is_suspended BOOLEAN DEFAULT FALSE,
    suspended_at TIMESTAMP WITH TIME ZONE,
    suspended_until TIMESTAMP WITH TIME ZONE,
    -- For temporary suspension
    suspension_reason TEXT,
    is_blocked BOOLEAN DEFAULT FALSE,
    -- Permanent block
    blocked_at TIMESTAMP WITH TIME ZONE,
    block_reason TEXT,
    -- SSO / LDAP info
    auth_provider TEXT DEFAULT 'local',
    -- 'local', 'google', 'ldap', 'sso'
    provider_id TEXT,
    -- UID from external provider
    last_password_change TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Login History
CREATE TABLE IF NOT EXISTS login_history (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT REFERENCES users(id) ON DELETE
    SET NULL,
        session_id TEXT,
        -- Optional link to session
        login_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        ip_address TEXT,
        user_agent TEXT,
        device_fingerprint TEXT,
        status TEXT CHECK (status IN ('success', 'failed', '2fa_challenge')),
        failure_reason TEXT
);
-- User Devices (For session management and security)
CREATE TABLE IF NOT EXISTS user_devices (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_name TEXT,
    -- e.g., "Chrome on Windows"
    device_fingerprint TEXT,
    last_active_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_trusted BOOLEAN DEFAULT FALSE,
    is_blocked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Triggers
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM pg_trigger
    WHERE tgname = 'update_user_security_timestamp'
) THEN CREATE TRIGGER update_user_security_timestamp BEFORE
UPDATE ON user_security FOR EACH ROW EXECUTE FUNCTION update_timestamp();
END IF;
IF NOT EXISTS (
    SELECT 1
    FROM pg_trigger
    WHERE tgname = 'update_user_devices_timestamp'
) THEN CREATE TRIGGER update_user_devices_timestamp BEFORE
UPDATE ON user_devices FOR EACH ROW EXECUTE FUNCTION update_timestamp();
END IF;
END $$;