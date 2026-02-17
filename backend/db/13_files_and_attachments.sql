-- Attached Files / File Metadata
-- Centralized table for managing all uploaded files (proofs, invoices, avatars)
CREATE TABLE IF NOT EXISTS attached_files (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid(),
    uploader_id TEXT REFERENCES users(id) ON DELETE
    SET NULL,
        file_name TEXT NOT NULL,
        original_name TEXT,
        mime_type TEXT,
        size_bytes BIGINT,
        storage_path TEXT NOT NULL,
        -- S3 key or local path
        public_url TEXT,
        -- If publicly accessible
        context_type TEXT,
        -- 'deposit_proof', 'expense_receipt', 'avatar'
        context_id TEXT,
        -- ID of the related record (deposit_id, expense_id)
        access_control TEXT CHECK (
            access_control IN ('public', 'private', 'group_shared')
        ) DEFAULT 'private',
        expires_at TIMESTAMP WITH TIME ZONE,
        -- For temporary files
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- Trigger for updated_at
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM pg_trigger
    WHERE tgname = 'update_attached_files_timestamp'
) THEN CREATE TRIGGER update_attached_files_timestamp BEFORE
UPDATE ON attached_files FOR EACH ROW EXECUTE FUNCTION update_timestamp();
END IF;
END $$;