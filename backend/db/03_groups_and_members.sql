-- Groups table
CREATE TABLE IF NOT EXISTS groups (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    service_id TEXT REFERENCES services(id),
    service_name TEXT,
    -- Fallback if not using service_id
    website TEXT,
    plan_name TEXT,
    amount DECIMAL(10, 2) DEFAULT 0 CHECK (amount >= 0),
    currency TEXT DEFAULT 'TWD',
    billing_cycle TEXT CHECK (
        billing_cycle IN ('one-time', 'monthly', 'yearly')
    ),
    billing_method TEXT CHECK (
        billing_method IN ('equal', 'fixed', 'percentage')
    ) DEFAULT 'equal',
    max_members INTEGER NOT NULL DEFAULT 1,
    billing_day INTEGER CHECK (
        billing_day >= 1
        AND billing_day <= 31
    ),
    next_payment_date DATE,
    created_by TEXT REFERENCES users(id),
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Index for service groups
CREATE INDEX IF NOT EXISTS idx_groups_service_id ON groups(service_id);
-- Group Members table
CREATE TABLE IF NOT EXISTS group_members (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id TEXT REFERENCES groups(id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users(id),
    temp_name TEXT,
    -- For non-registered members
    role TEXT CHECK (role IN ('admin', 'member')) DEFAULT 'member',
    share_ratio DECIMAL(5, 2) CHECK (
        share_ratio >= 0
        AND share_ratio <= 100
    ),
    -- For percentage billing
    fixed_amount DECIMAL(10, 2) CHECK (fixed_amount >= 0),
    -- For fixed billing
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT member_identity CHECK (
        user_id IS NOT NULL
        OR temp_name IS NOT NULL
    )
);
-- General query index for group members
CREATE INDEX IF NOT EXISTS idx_group_members_group_id ON group_members(group_id);
-- Unique index for members
CREATE UNIQUE INDEX IF NOT EXISTS idx_group_user_unique ON group_members (group_id, user_id)
WHERE user_id IS NOT NULL;