
import { pool } from '../src/db';

async function migrate() {
    const client = await pool.connect();
    try {
        console.log('Starting migration...');
        await client.query('BEGIN');

        // Add status column
        console.log('Adding status column...');
        await client.query(`
            ALTER TABLE group_members 
            ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'active' CHECK (status IN ('active', 'invited', 'suspended'));
        `);

        // Add permissions column
        console.log('Adding permissions column...');
        await client.query(`
            ALTER TABLE group_members 
            ADD COLUMN IF NOT EXISTS permissions JSONB DEFAULT '{}';
        `);
        
        // Update existing rows ensuring they are active (Potentially redundant with DEFAULT but good for safety if default is dropped later)
        await client.query(`
            UPDATE group_members SET status = 'active' WHERE status IS NULL;
        `);

        await client.query('COMMIT');
        console.log('Migration completed successfully.');
    } catch (error) {
        await client.query('ROLLBACK');
        console.error('Migration failed:', error);
        process.exit(1);
    } finally {
        client.release();
        await pool.end();
    }
}

migrate();
