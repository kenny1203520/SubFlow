-- Groups table (Identity only)
CREATE TABLE IF NOT EXISTS groups (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    max_members INTEGER NOT NULL DEFAULT 1,
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'paused', 'cancelled')),
    created_by TEXT REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Group Services table (Billing/Subscription details)
CREATE TABLE IF NOT EXISTS group_services (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    service_id TEXT REFERENCES services(id),
    service_name TEXT, -- Fallback if not using service_id
    website TEXT,
    plan_name TEXT,
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'paused', 'cancelled')),

    -- Billing Amount & Currency
    amount DECIMAL(15, 2) DEFAULT 0 CHECK (amount >= 0),
    service_currency TEXT DEFAULT 'TWD', -- Original currency of the service
    payment_currency TEXT DEFAULT 'TWD', -- Currency members pay in
    
    exchange_rate_mode TEXT CHECK (exchange_rate_mode IN ('fixed_custom', 'fixed_current', 'floating')) DEFAULT 'floating',
    fixed_rate_value DECIMAL(15, 6), -- Used when mode is fixed_*

    -- Rounding Rules
    rounding_method TEXT CHECK (rounding_method IN ('round', 'ceil', 'floor')) DEFAULT 'round',
    rounding_precision INTEGER DEFAULT 0, -- 0=integer, 2=2 decimal places

    -- Billing Cycle Configuration
    billing_type TEXT CHECK (billing_type IN ('once', 'recurring')) DEFAULT 'recurring',
    interval_unit TEXT CHECK (interval_unit IN ('minute', 'hour', 'day', 'week', 'month', 'quarter', 'year')),
    interval_value INTEGER DEFAULT 1,
    days_of_week INTEGER[], -- 1=Mon, 7=Sun (for weekly/recurring)
    start_date TIMESTAMP WITH TIME ZONE,
    end_condition TEXT CHECK (end_condition IN ('never', 'occurrences', 'on_date')) DEFAULT 'never',
    end_value TEXT, -- Stores occurrences count or date string

    -- Billing Method
    billing_method TEXT CHECK (billing_method IN ('equal', 'fixed', 'percentage')) DEFAULT 'equal',
    
    -- Fees & Taxes
    extra_fee_percentage DECIMAL(5, 2) DEFAULT 0, -- e.g., 1.5 for 1.5%
    fixed_fee_amount DECIMAL(15, 2) DEFAULT 0, -- Fixed fee per bill

    next_payment_date DATE,
    
    created_by TEXT REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Index for group services
CREATE INDEX IF NOT EXISTS idx_group_services_group_id ON group_services(group_id);
CREATE INDEX IF NOT EXISTS idx_group_services_service_id ON group_services(service_id);

-- Group Members table (Roster only)
CREATE TABLE IF NOT EXISTS group_members (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id TEXT REFERENCES groups(id) ON DELETE CASCADE,
    
    -- Member Binding Logic:
    -- 1. user_id can be NULL (Top-level placeholder)
    -- 2. temp_name represents the "Slot Name"
    -- 3. If user_id is set, it's a bound user.
    -- 4. To "transfer" binding: Update user_id.
    user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    temp_name TEXT, -- Nickname within this specific group
    
    display_name TEXT,
    can_self_edit_nickname BOOLEAN DEFAULT TRUE,
    
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    invited_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraint: Either user_id OR temp_name must exist to identify the slot
    CONSTRAINT member_identity CHECK (user_id IS NOT NULL OR temp_name IS NOT NULL)
);
-- General query index for group members
CREATE INDEX IF NOT EXISTS idx_group_members_group_id ON group_members(group_id);
-- Unique index for members
CREATE UNIQUE INDEX IF NOT EXISTS idx_group_user_unique ON group_members (group_id, user_id) WHERE user_id IS NOT NULL;

-- Group Service Members table (Splits per service)
CREATE TABLE IF NOT EXISTS group_service_members (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    group_service_id TEXT REFERENCES group_services(id) ON DELETE CASCADE,
    member_id TEXT REFERENCES group_members(id) ON DELETE CASCADE,
    
    share_ratio DECIMAL(5, 2) CHECK (share_ratio >= 0 AND share_ratio <= 100), -- For percentage billing
    fixed_amount DECIMAL(15, 2) CHECK (fixed_amount >= 0), -- For fixed billing
    
    role TEXT CHECK (role IN ('owner', 'member')) DEFAULT 'member', -- 'owner' here might mean who is "using" this slot strictly? Or just mostly 'member'

    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'paused', 'cancelled')),

    started_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    ended_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(group_service_id, member_id)
);
-- Index for service members
CREATE INDEX IF NOT EXISTS idx_group_service_members_service_id ON group_service_members(group_service_id);
CREATE INDEX IF NOT EXISTS idx_group_service_members_member_id ON group_service_members(member_id);

-- Triggers for 'updated_at'
DO $$ BEGIN
    -- Groups
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_groups_timestamp'
    ) THEN
        CREATE TRIGGER update_groups_timestamp
        BEFORE UPDATE ON groups
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- Group Services
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_group_services_timestamp'
    ) THEN
        CREATE TRIGGER update_group_services_timestamp
        BEFORE UPDATE ON group_services
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- Group Members
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_group_members_timestamp'
    ) THEN
        CREATE TRIGGER update_group_members_timestamp
        BEFORE UPDATE ON group_members
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- Group Service Members
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_group_service_members_timestamp'
    ) THEN
        CREATE TRIGGER update_group_service_members_timestamp
        BEFORE UPDATE ON group_service_members
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;
END $$;