-- Services table
CREATE TABLE IF NOT EXISTS services (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    domain TEXT,
    -- for fetching icons
    icon_url TEXT,
    is_system BOOLEAN DEFAULT FALSE,
    created_by TEXT REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Triggers for 'updated_at'
DO $$ BEGIN
    -- Services
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_services_timestamp'
    ) THEN 
        CREATE TRIGGER update_services_timestamp 
        BEFORE UPDATE ON services 
        FOR EACH ROW EXECUTE FUNCTION update_timestamp();
    END IF;
END $$;