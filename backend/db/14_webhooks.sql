-- WebHooks Configurations
CREATE TABLE IF NOT EXISTS webhooks (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Owner/Context
    created_by TEXT NOT NULL REFERENCES users(id),
    group_id TEXT REFERENCES groups(id) ON DELETE CASCADE,
    -- Optional: Webhook for specific group events
    -- Target
    url TEXT NOT NULL,
    secret TEXT,
    -- Shared secret for HMAC signature (e.g. "whsec_...")
    -- Configuration
    events TEXT [] NOT NULL,
    -- List of subscribed events: ['bill.created', 'payment.received']
    is_active BOOLEAN DEFAULT TRUE,
    description TEXT,
    -- Status
    failure_count INTEGER DEFAULT 0,
    last_failure_at TIMESTAMP WITH TIME ZONE,
    last_success_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- WebHook Delivery Logs (For debugging and history)
CREATE TABLE IF NOT EXISTS webhook_logs (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id TEXT REFERENCES webhooks(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload JSONB,
    -- The data sent
    response_status INTEGER,
    -- HTTP Status Code (200, 404, 500)
    response_body TEXT,
    -- Truncated response from server
    duration_ms INTEGER,
    attempt_count INTEGER DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Trigger for 'updated_at'
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM pg_trigger
    WHERE tgname = 'update_webhooks_timestamp'
) THEN CREATE TRIGGER update_webhooks_timestamp BEFORE
UPDATE ON webhooks FOR EACH ROW EXECUTE FUNCTION update_timestamp();
END IF;
END $$;
