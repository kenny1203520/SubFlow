-- System Settings Table
-- Stores global configuration for the application (e.g., Captcha settings, password policies, 2FA enforcement)
CREATE TABLE IF NOT EXISTS system_settings (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_by TEXT REFERENCES users(id) ON DELETE SET NULL
);

-- IP Blocks Table
-- Manages blocked IP addresses for security and abuse prevention
CREATE TABLE IF NOT EXISTS ip_blocks (
    ip_address INET PRIMARY KEY,
    reason TEXT,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Default System Settings Initialization
-- You can seed default settings here if they do not exist
INSERT INTO system_settings (key, value, description)
VALUES 
  ('auth.captcha', '{"enabled": false, "provider": "none", "siteKey": null, "secretKey": null}'::jsonb, 'CAPTCHA configuration for authentication endpoints'),
  ('auth.password_policy', '{"minLength": 8, "requireUppercase": true, "requireLowercase": true, "requireNumbers": true, "requireSymbols": true}'::jsonb, 'Global password complexity requirements'),
  ('auth.require_2fa', '{"enabled": false}'::jsonb, 'Whether 2FA is globally enforced for all users'),
  ('security.auth_lockout', '{"maxFailedAttempts": 5, "lockoutDurationMins": 720}'::jsonb, 'Account lockout policy after failed login attempts'),
  ('security.rate_limit', '{"authWindowMs": 900000, "authMax": 5, "apiWindowMs": 300000, "apiMax": 100}'::jsonb, 'Rate limiting configuration for auth and API endpoints'),
  ('groups.group_limit', '{"max": 100}'::jsonb, 'Maximum number of groups a user can create or be a member of'),
  ('groups.role_limit', '{"max": 20}'::jsonb, 'Maximum number of roles allowed per group'),
  ('groups.member_limit', '{"max": 1000}'::jsonb, 'Maximum number of members allowed per group')
ON CONFLICT (key) DO NOTHING;

-- Triggers for 'updated_at'
DO $$ BEGIN 
    -- System Settings
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_system_settings_timestamp'
    ) THEN 
        CREATE TRIGGER update_system_settings_timestamp 
        BEFORE UPDATE ON system_settings 
        FOR EACH ROW EXECUTE FUNCTION update_timestamp();
    END IF;

    -- IP Blocks
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_ip_blocks_timestamp'
    ) THEN 
        CREATE TRIGGER update_ip_blocks_timestamp 
        BEFORE UPDATE ON ip_blocks 
        FOR EACH ROW EXECUTE FUNCTION update_timestamp();
    END IF;
END $$;
