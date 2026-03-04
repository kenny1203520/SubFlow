-- User Security Settings
CREATE TABLE IF NOT EXISTS user_security (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    two_factor_secret TEXT,
    -- Encrypted TOTP secret
    backup_codes TEXT [],
    -- Hashed backup codes
    passkey_enabled BOOLEAN DEFAULT FALSE,
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
    -- Failed login attempts
    failed_login_attempts INT DEFAULT 0,
    -- SSO / LDAP info
    auth_provider TEXT DEFAULT 'local',
    -- 'local', 'google', 'ldap', 'sso'
    provider_id TEXT,
    ldap_enabled BOOLEAN DEFAULT FALSE,
    sso_enabled BOOLEAN DEFAULT FALSE,
    -- Notification Preferences
    notify_new_device BOOLEAN DEFAULT TRUE,
    notify_new_location BOOLEAN DEFAULT TRUE,
    notify_suspicious_activity BOOLEAN DEFAULT TRUE,
    -- UID from external provider
    last_password_change TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- User Devices (For session management and security)
CREATE TABLE IF NOT EXISTS user_devices (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_name TEXT,
    -- e.g., "Chrome on Windows"
    user_agent TEXT,
    ip_address TEXT,
    device_fingerprint TEXT,
    device_token TEXT UNIQUE,
    last_active_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_trusted BOOLEAN DEFAULT FALSE,
    trusted_at TIMESTAMP WITH TIME ZONE,
    is_blocked BOOLEAN DEFAULT FALSE,
    blocked_at TIMESTAMP WITH TIME ZONE,
    block_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Index
CREATE INDEX IF NOT EXISTS idx_user_devices_token ON user_devices(device_token);
CREATE INDEX IF NOT EXISTS idx_user_devices_fingerprint ON user_devices(device_fingerprint);

-- WebAuthn Credentials Table (for PassKeys)
CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id TEXT NOT NULL UNIQUE, -- Base64URL encoded credential ID
    public_key TEXT NOT NULL, -- Base64URL encoded public key
    counter BIGINT NOT NULL DEFAULT 0, -- Signature counter for replay attack prevention
    device_type TEXT, -- 'platform' (built-in) or 'cross-platform' (external key)
    transports TEXT[], -- ['usb', 'nfc', 'ble', 'internal']
    backup_eligible BOOLEAN DEFAULT FALSE,
    backup_state BOOLEAN DEFAULT FALSE,
    attestation_object TEXT, -- Store for audit purposes
    aaguid TEXT, -- Authenticator Attestation GUID
    device_name TEXT, -- User-friendly name like "iPhone 15 Pro"
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Index
CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user_id ON webauthn_credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_credential_id ON webauthn_credentials(credential_id);

-- WebAuthn Challenges Table (for registration and authentication)
CREATE TABLE IF NOT EXISTS webauthn_challenges (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    challenge TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('registration', 'authentication')),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Index
CREATE INDEX IF NOT EXISTS idx_webauthn_challenges_user_id ON webauthn_challenges(user_id);
CREATE INDEX IF NOT EXISTS idx_webauthn_challenges_expires ON webauthn_challenges(expires_at);

-- LDAP User Mapping Table
CREATE TABLE IF NOT EXISTS ldap_users (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    ldap_dn TEXT NOT NULL, -- Distinguished Name
    ldap_uid TEXT, -- LDAP unique identifier
    ldap_email TEXT,
    ldap_display_name TEXT,
    ldap_groups TEXT[], -- LDAP group memberships
    last_sync_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Index
CREATE INDEX IF NOT EXISTS idx_ldap_users_user_id ON ldap_users(user_id);
CREATE INDEX IF NOT EXISTS idx_ldap_users_ldap_dn ON ldap_users(ldap_dn);
CREATE INDEX IF NOT EXISTS idx_ldap_users_ldap_uid ON ldap_users(ldap_uid);

-- SSO Providers Table (OAuth2/SAML)
CREATE TABLE IF NOT EXISTS sso_providers (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL, -- 'google', 'github', 'microsoft', 'okta', etc.
    type TEXT NOT NULL CHECK (type IN ('oauth2', 'saml', 'oidc')),
    enabled BOOLEAN DEFAULT TRUE,
    -- OAuth2/OIDC Configuration
    client_id TEXT,
    client_secret TEXT, -- Should be encrypted
    authorization_url TEXT,
    token_url TEXT,
    userinfo_url TEXT,
    scope TEXT,
    -- SAML Configuration
    saml_entity_id TEXT,
    saml_sso_url TEXT,
    saml_certificate TEXT,
    -- Common
    metadata_url TEXT,
    config JSONB, -- Additional provider-specific configuration
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- SSO User Mappings Table
CREATE TABLE IF NOT EXISTS sso_users (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_id TEXT NOT NULL REFERENCES sso_providers(id) ON DELETE CASCADE,
    external_id TEXT NOT NULL, -- User ID from the SSO provider
    external_email TEXT,
    external_username TEXT,
    access_token TEXT, -- Encrypted
    refresh_token TEXT, -- Encrypted
    token_expires_at TIMESTAMP WITH TIME ZONE,
    profile_data JSONB, -- Store additional profile information
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider_id, external_id)
);
-- Index
CREATE INDEX IF NOT EXISTS idx_sso_users_user_id ON sso_users(user_id);
CREATE INDEX IF NOT EXISTS idx_sso_users_provider_id ON sso_users(provider_id);
CREATE INDEX IF NOT EXISTS idx_sso_users_external_id ON sso_users(external_id);

-- Device Login Notifications Table
CREATE TABLE IF NOT EXISTS device_notifications (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id TEXT REFERENCES user_devices(id) ON DELETE CASCADE,
    notification_type TEXT NOT NULL CHECK (notification_type IN ('new_device', 'new_location', 'suspicious_activity')),
    message TEXT NOT NULL,
    ip_address TEXT,
    location TEXT,
    is_read BOOLEAN DEFAULT FALSE,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Index
CREATE INDEX IF NOT EXISTS idx_device_notifications_user_id ON device_notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_device_notifications_device_id ON device_notifications(device_id);

-- Triggers for 'updated_at'
DO $$ BEGIN
    -- User Security
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_user_security_timestamp'
    ) THEN
        CREATE TRIGGER update_user_security_timestamp
        BEFORE UPDATE ON user_security
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- User Devices
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_user_devices_timestamp'
    ) THEN
        CREATE TRIGGER update_user_devices_timestamp
        BEFORE UPDATE ON user_devices
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- WebAuthn Credentials
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_webauthn_credentials_timestamp'
    ) THEN
        CREATE TRIGGER update_webauthn_credentials_timestamp
        BEFORE UPDATE ON webauthn_credentials
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- LDAP Users
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_ldap_users_timestamp'
    ) THEN
        CREATE TRIGGER update_ldap_users_timestamp
        BEFORE UPDATE ON ldap_users
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- SSO Providers
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_sso_providers_timestamp'
    ) THEN
        CREATE TRIGGER update_sso_providers_timestamp
        BEFORE UPDATE ON sso_providers
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- SSO Users
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_sso_users_timestamp'
    ) THEN
        CREATE TRIGGER update_sso_users_timestamp
        BEFORE UPDATE ON sso_users
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;
END $$;