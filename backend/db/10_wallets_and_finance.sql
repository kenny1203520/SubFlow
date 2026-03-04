-- Exchange Rates table
CREATE TABLE IF NOT EXISTS exchange_rates (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    from_currency TEXT NOT NULL,
    to_currency TEXT NOT NULL,
    rate DECIMAL(15, 6) NOT NULL,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(from_currency, to_currency, date)
);

-- User Wallets table (Hierarchical)
CREATE TABLE IF NOT EXISTS user_wallets (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    group_id TEXT REFERENCES groups(id) ON DELETE CASCADE,
    -- Nullable (for Global/Root wallet)
    service_id TEXT REFERENCES services(id),
    -- Nullable (for Group wallet)
    currency TEXT NOT NULL DEFAULT 'TWD',
    balance DECIMAL(15, 2) DEFAULT 0 CHECK (balance >= 0),
    -- Prevent negative balance at DB level
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, group_id, service_id, currency) -- Ensure only one wallet per context per currency
);

-- Deposits table
CREATE TABLE IF NOT EXISTS deposits (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES users(id),
    target_wallet_id TEXT REFERENCES user_wallets(id),
    -- If null, maybe default to root wallet? Better to be explicit.
    amount DECIMAL(15, 2) NOT NULL CHECK (amount > 0),
    currency TEXT NOT NULL,
    method TEXT NOT NULL,
    -- 'transfer', 'credit_card', 'cash', etc.
    proof_url TEXT,
    -- Image URL for proof
    transaction_ref TEXT,
    -- External transaction ID
    status TEXT CHECK (
        status IN ('pending', 'verified', 'rejected', 'voided')
    ) DEFAULT 'pending',
    recorded_by TEXT REFERENCES users(id),
    -- User or Admin who created the record
    verified_by TEXT REFERENCES users(id),
    -- Admin who verified
    verified_at TIMESTAMP WITH TIME ZONE,
    note TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Wallet Transactions (Ledger)
CREATE TABLE IF NOT EXISTS wallet_transactions (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id TEXT NOT NULL REFERENCES user_wallets(id),
    type TEXT CHECK (
        type IN (
            'deposit',
            'withdrawal',
            'payment',
            'transfer_in',
            'transfer_out',
            'refund',
            'adjustment'
        )
    ) NOT NULL,
    amount DECIMAL(15, 2) NOT NULL,
    -- Positive for increase, Negative for decrease
    currency TEXT NOT NULL,
    status TEXT CHECK (
        status IN ('pending', 'completed', 'failed', 'voided')
    ) DEFAULT 'completed',
    related_id TEXT,
    -- Ref to deposit_id, bill_split_id, or transfer_id
    related_type TEXT,
    -- 'deposit', 'bill_split', 'wallet_transfer'
    description TEXT,
    void_ref_id TEXT REFERENCES wallet_transactions(id),
    -- If voided, reference the original transaction
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Wallet Transfers (Internal)
CREATE TABLE IF NOT EXISTS wallet_transfers (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    from_wallet_id TEXT REFERENCES user_wallets(id),
    to_wallet_id TEXT REFERENCES user_wallets(id),
    amount DECIMAL(15, 2) NOT NULL CHECK (amount > 0),
    currency TEXT NOT NULL,
    status TEXT CHECK (status IN ('pending', 'completed', 'failed')) DEFAULT 'pending',
    created_by TEXT REFERENCES users(id),
    executed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Auto-Debit Rules
CREATE TABLE IF NOT EXISTS auto_debit_rules (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    target_wallet_id TEXT REFERENCES user_wallets(id) ON DELETE CASCADE,
    -- The wallet that needs money (e.g., Service Wallet)
    backup_wallet_id TEXT REFERENCES user_wallets(id) ON DELETE CASCADE,
    -- The wallet to pull money from (e.g., Root Wallet)
    priority INTEGER DEFAULT 1,
    -- Lower number = Higher priority
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(target_wallet_id, backup_wallet_id)
);

-- Triggers for 'updated_at'
DO $$ BEGIN
    -- Exchange Rates
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_exchange_rates_timestamp'
    ) THEN
        CREATE TRIGGER update_exchange_rates_timestamp
        BEFORE UPDATE ON exchange_rates
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- User Wallets
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_user_wallets_timestamp'
    ) THEN
        CREATE TRIGGER update_user_wallets_timestamp
        BEFORE UPDATE ON user_wallets
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- Deposits
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_deposits_timestamp'
    ) THEN
        CREATE TRIGGER update_deposits_timestamp
        BEFORE UPDATE ON deposits
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- Wallet Transactions
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_wallet_transactions_timestamp'
    ) THEN
        CREATE TRIGGER update_wallet_transactions_timestamp
        BEFORE UPDATE ON wallet_transactions
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- Wallet Transfers
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_wallet_transfers_timestamp'
    ) THEN
        CREATE TRIGGER update_wallet_transfers_timestamp
        BEFORE UPDATE ON wallet_transfers
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;

    -- Auto Debit Rules
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'update_auto_debit_rules_timestamp'
    ) THEN
        CREATE TRIGGER update_auto_debit_rules_timestamp
        BEFORE UPDATE ON auto_debit_rules
        FOR EACH ROW
        EXECUTE FUNCTION update_timestamp();
    END IF;
END $$;