-- Billing Ledger
-- Tracks the generation of bills for recurring groups to prevent duplicate/missed bills
CREATE TABLE IF NOT EXISTS billing_ledger (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    cycle_start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    cycle_end_date TIMESTAMP WITH TIME ZONE NOT NULL,
    billing_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    status TEXT CHECK (status IN ('pending', 'generated', 'failed')) DEFAULT 'pending',
    bill_id TEXT REFERENCES bills(id),
    -- Link to the generated bill
    note TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(group_id, cycle_start_date, cycle_end_date)
);
-- Note: We assume 'bills' and 'bill_splits' tables exist in 04_finance_and_billing.sql
-- We might need to alter them to support Multi-Currency if they don't already.
-- Detailed checks on 'bills' table:
-- It currently has: total_amount, currency.
-- We need to ensure 'currency' in bills matches 'payment_currency' or 'service_currency' logic.
-- Ideally, a bill should record specific exchange rate used at generation time.
-- Altering 'bills' to add exchange rate info is recommended.
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'bills'
        AND column_name = 'exchange_rate'
) THEN
ALTER TABLE bills
ADD COLUMN exchange_rate DECIMAL(15, 6) DEFAULT 1,
    ADD COLUMN service_amount DECIMAL(15, 2),
    -- Original amount in service currency
ADD COLUMN service_currency TEXT;
END IF;
END $$;
-- Triggers
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM pg_trigger
    WHERE tgname = 'update_billing_ledger_timestamp'
) THEN CREATE TRIGGER update_billing_ledger_timestamp BEFORE
UPDATE ON billing_ledger FOR EACH ROW EXECUTE FUNCTION update_timestamp();
END IF;
END $$;