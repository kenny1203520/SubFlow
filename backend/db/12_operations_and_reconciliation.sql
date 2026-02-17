-- Reconciliation Logs
-- Auto-generated reports comparing Wallet Balance vs. Transaction Sum
CREATE TABLE IF NOT EXISTS reconciliation_logs (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id TEXT REFERENCES user_wallets(id) ON DELETE CASCADE,
    snapshot_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    recorded_balance DECIMAL(15, 2) NOT NULL,
    -- The balance in user_wallets
    calculated_balance DECIMAL(15, 2) NOT NULL,
    -- Sum of wallet_transactions
    discrepancy DECIMAL(15, 2) GENERATED ALWAYS AS (recorded_balance - calculated_balance) STORED,
    status TEXT CHECK (status IN ('balanced', 'discrepancy', 'fixed')) DEFAULT 'balanced',
    note TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Exchange Rate Sync Logs
-- To track if the external API is updating rates correctly
CREATE TABLE IF NOT EXISTS exchange_rate_sync_logs (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL,
    -- e.g. 'OpenExchangeRates', 'Fixer', 'Manual'
    sync_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    status TEXT CHECK (status IN ('success', 'failed', 'partial')) NOT NULL,
    base_currency TEXT DEFAULT 'USD',
    rates_count INTEGER DEFAULT 0,
    error_message TEXT,
    duration_ms INTEGER -- How long the sync took
);
-- Triggers (if any needed, mostly log tables don't need update triggers)