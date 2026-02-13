import { pool } from '../src/db';
import fs from 'fs';
import path from 'path';

async function migrate() {
    const client = await pool.connect();
    try {
        const schemaPath = path.join(__dirname, '../db/schema.sql');
        const schemaSql = fs.readFileSync(schemaPath, 'utf8');

        console.log('Running migration...');
        await client.query("BEGIN");
        await client.query(schemaSql);
        await client.query("COMMIT");
        console.log('Migration completed successfully.');
    } catch (error) {
        await client.query("ROLLBACK");
        console.error('Migration failed:', error);
        process.exit(1);
    } finally {
        client.release();
        await pool.end();
    }
}

migrate();
