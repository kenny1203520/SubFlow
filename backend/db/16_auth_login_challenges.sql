-- 2FA Login Challenges
-- Purpose:
--   Bind /signin and /signin/2fa into one short-lived, one-time challenge flow.
-- Security notes:
--   1) Store only challenge hash in DB (never plaintext token).
--   2) Mark challenge as consumed immediately after successful 2FA.
--   3) Enforce short TTL and retry limits to reduce brute-force risk.

CREATE TABLE IF NOT EXISTS auth_login_challenges (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Hash of opaque challenge token sent to client (cookie/header/body)
    challenge_hash TEXT NOT NULL UNIQUE,

    -- Current use case: password passed, waiting for 2FA verification
    challenge_type TEXT NOT NULL CHECK (challenge_type IN ('signin_2fa')),

    -- Optional binding to request context from the first-factor step
    ip_address TEXT,
    user_agent TEXT,
    device_fingerprint TEXT,

    -- Replay/bruteforce controls
    max_attempts INT NOT NULL DEFAULT 5 CHECK (max_attempts > 0),
    attempt_count INT NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    last_attempt_at TIMESTAMP WITH TIME ZONE,

    -- Lifecycle
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    consumed_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoke_reason TEXT,

    -- Traceability
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CHECK (expires_at > created_at),
    CHECK (
        NOT (consumed_at IS NOT NULL AND revoked_at IS NOT NULL)
    )
);

-- Fast lookup by user/type in 2FA verification flow
CREATE INDEX IF NOT EXISTS idx_auth_login_challenges_user_type
    ON auth_login_challenges(user_id, challenge_type);

-- Cleanup job / TTL sweep
CREATE INDEX IF NOT EXISTS idx_auth_login_challenges_expires
    ON auth_login_challenges(expires_at);

-- Active challenge lookup for conflict checks and revocation
CREATE INDEX IF NOT EXISTS idx_auth_login_challenges_active
    ON auth_login_challenges(user_id, challenge_type, created_at DESC)
    WHERE consumed_at IS NULL AND revoked_at IS NULL;

-- Trigger for updated_at
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_auth_login_challenges_timestamp'
    ) THEN
        CREATE TRIGGER update_auth_login_challenges_timestamp
        BEFORE UPDATE ON auth_login_challenges
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;
END $$;
