-- System Settings Table
-- Stores global configuration for the application (e.g., Captcha settings, password policies, 2FA enforcement)
CREATE TABLE IF NOT EXISTS system_settings (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_by TEXT REFERENCES users(id) ON DELETE SET NULL
);

-- Default System Settings Initialization
-- You can seed default settings here if they do not exist
INSERT INTO system_settings (key, value, description)
VALUES 
  ('auth.captcha', '{"enabled": false, "provider": "none",  "version": null, "siteKey": null, "secretKey": null}'::jsonb, 'CAPTCHA configuration for authentication endpoints'),
  ('auth.password_policy', '{"minLength": 8, "requireUppercase": true, "requireLowercase": true, "requireNumbers": true, "requireSymbols": true}'::jsonb, 'Global password complexity requirements'),
  ('auth.require_2fa', '{"enabled": false}'::jsonb, 'Whether 2FA is globally enforced for all users'),
  ('security.auth_lockout', '{"maxFailedAttempts": 5, "lockoutDurationMins": 720}'::jsonb, 'Account lockout policy after failed login attempts'),
  ('security.rate_limit', '{"authWindowMs": 900000, "authMax": 6, "apiWindowMs": 300000, "apiMax": 1000}'::jsonb, 'Rate limiting configuration for auth and API endpoints'),
  ('groups.settings', '{"groupsLimit": 100, "roleLimitPerGroup": 10, "memberLimitPerGroup": 1000}'::jsonb, 'Maximum number of groups a user can create or be a member of')
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
END $$;
