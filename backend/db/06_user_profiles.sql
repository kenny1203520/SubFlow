-- User Profiles table
CREATE TABLE IF NOT EXISTS user_profiles (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    first_name TEXT,
    middle_name TEXT,
    last_name TEXT,
    nickname TEXT,
    birthday DATE,
    id_number TEXT,
    -- Consider encryption for sensitive data
    passport_number TEXT,
    -- Consider encryption for sensitive data
    mobile_phone TEXT,
    address TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Trigger for update timestamp
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM pg_trigger
    WHERE tgname = 'update_user_profiles_timestamp'
) THEN CREATE TRIGGER update_user_profiles_timestamp BEFORE
UPDATE ON user_profiles FOR EACH ROW EXECUTE FUNCTION update_timestamp();
END IF;
END $$;