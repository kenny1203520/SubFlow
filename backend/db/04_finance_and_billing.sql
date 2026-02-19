-- Expenses table
CREATE TABLE IF NOT EXISTS expenses (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    paid_by TEXT NOT NULL REFERENCES users(id),
    amount DECIMAL(10, 2) NOT NULL CHECK (amount >= 0),
    description TEXT,
    date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    currency TEXT DEFAULT 'TWD',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Index for expenses
CREATE INDEX IF NOT EXISTS idx_expenses_group_id ON expenses(group_id);

-- Expense Splits table
CREATE TABLE IF NOT EXISTS expense_splits (
    expense_id TEXT NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
    member_id TEXT NOT NULL REFERENCES group_members(id) ON DELETE CASCADE,
    amount_owed DECIMAL(10, 2) NOT NULL CHECK (amount_owed >= 0),
    status TEXT CHECK (status IN ('pending', 'paid')) DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (expense_id, member_id)
);

-- Subscriptions table
CREATE TABLE IF NOT EXISTS subscriptions (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id TEXT NOT NULL REFERENCES users(id),
    service_name TEXT NOT NULL,
    amount DECIMAL(10, 2) NOT NULL CHECK (amount >= 0),
    cycle TEXT CHECK (cycle IN ('monthly', 'yearly')) NOT NULL,
    status TEXT CHECK (status IN ('active', 'paused', 'cancelled')) DEFAULT 'active',
    next_payment_date DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Subscription Splits table
CREATE TABLE IF NOT EXISTS subscription_splits (
    subscription_id TEXT NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    member_id TEXT NOT NULL REFERENCES group_members(id),
    share_percentage DECIMAL(5, 2) CHECK (
        share_percentage >= 0
        AND share_percentage <= 100
    ),
    fixed_amount DECIMAL(10, 2) CHECK (fixed_amount >= 0),
    status TEXT CHECK (status IN ('pending', 'paid')) DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (subscription_id, member_id)
);

-- Bills / Tickets table
CREATE TABLE IF NOT EXISTS bills (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id TEXT REFERENCES groups(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT,
    total_amount DECIMAL(10, 2) NOT NULL CHECK (total_amount >= 0),
    currency TEXT DEFAULT 'TWD',
    issue_date DATE DEFAULT CURRENT_DATE,
    due_date DATE,
    status TEXT CHECK (
        status IN ('draft', 'pending', 'paid', 'overdue')
    ) DEFAULT 'pending',
    created_by TEXT REFERENCES users(id),
    updated_by TEXT REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Index for group bills
CREATE INDEX IF NOT EXISTS idx_bills_group_id ON bills(group_id);

-- Bill Splits table
CREATE TABLE IF NOT EXISTS bill_splits (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    bill_id TEXT REFERENCES bills(id) ON DELETE CASCADE,
    member_id TEXT REFERENCES group_members(id) ON DELETE
    SET NULL,
    amount_owed DECIMAL(10, 2) NOT NULL CHECK (amount_owed >= 0),
    paid_amount DECIMAL(10, 2) DEFAULT 0 CHECK (paid_amount >= 0),
    status TEXT CHECK (status IN ('pending', 'paid')) DEFAULT 'pending',
    paid_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Index for bill splits
CREATE INDEX IF NOT EXISTS idx_bill_splits_bill_id ON bill_splits(bill_id);