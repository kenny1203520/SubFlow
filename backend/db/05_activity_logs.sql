-- Activity Logs (Replacing basic Audit Logs)
-- This table is designed for high-volume, append-only security logging.
CREATE TABLE IF NOT EXISTS activity_logs (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT REFERENCES users(id) ON DELETE
    SET NULL,
        -- Keep log even if user is deleted
        risk_level TEXT CHECK (
            risk_level IN ('low', 'medium', 'high', 'critical')
        ) DEFAULT 'low',
        behavior_type TEXT NOT NULL,
        -- 'auth', 'finance', 'config', 'data_access'
        action TEXT NOT NULL,
        -- 'login', 'logout', 'deposit_verified', 'bill_paid', 'schema_change'
        description TEXT,
        details JSONB,
        -- Full request payload or diff
        -- Security Context
        ip_address TEXT,
        user_agent TEXT,
        device_fingerprint TEXT,
        -- Computed frontend hash
        geo_location TEXT,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Index for quick security audits
CREATE INDEX IF NOT EXISTS idx_activity_logs_user_id ON activity_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_activity_logs_created_at ON activity_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_activity_logs_risk_level ON activity_logs(risk_level)
WHERE risk_level IN ('high', 'critical');