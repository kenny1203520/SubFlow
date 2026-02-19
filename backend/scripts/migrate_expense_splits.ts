import { pool } from '../src/db';

async function runMigration() {
    const client = await pool.connect();
    try {
        console.log('Starting migration: expense_splits member_id refactor...');
        await client.query('BEGIN');

        // 1. Add member_id column
        await client.query(`
            ALTER TABLE expense_splits 
            ADD COLUMN IF NOT EXISTS member_id TEXT REFERENCES group_members(id) ON DELETE CASCADE;
        `);

        // 2. Populate member_id based on existing user_id and group context (complex join needed)
        // We know expense -> group_id. 
        // We know split -> user_id. 
        // We need group_members where group_id = expense.group_id AND user_id = split.user_id.
        await client.query(`
            UPDATE expense_splits es
            SET member_id = gm.id
            FROM expenses e, group_members gm
            WHERE es.expense_id = e.id 
            AND e.group_id = gm.group_id 
            AND es.user_id = gm.user_id
        `);

        // 3. Handle cases where member_id might still be null (if user left group/hard deleted?)
        // For now, we assume data integrity holds. If member_id is null, it's orphan data.
        // We'll delete splits that couldn't be mapped.
        await client.query(`DELETE FROM expense_splits WHERE member_id IS NULL`);

        // 4. Alter constraints
        // Drop old PK
        await client.query(`ALTER TABLE expense_splits DROP CONSTRAINT expense_splits_pkey`);

        // Make user_id nullable
        await client.query(`ALTER TABLE expense_splits ALTER COLUMN user_id DROP NOT NULL`);

        // Make member_id NOT NULL
        await client.query(`ALTER TABLE expense_splits ALTER COLUMN member_id SET NOT NULL`);

        // Add new PK
        await client.query(`ALTER TABLE expense_splits ADD PRIMARY KEY (expense_id, member_id)`);

        await client.query('COMMIT');
        console.log('Migration completed successfully.');
    } catch (error) {
        await client.query('ROLLBACK');
        console.error('Migration failed:', error);
    } finally {
        client.release();
        process.exit();
    }
}

runMigration();
