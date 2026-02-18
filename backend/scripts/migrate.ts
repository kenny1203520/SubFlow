import { pool } from '../src/db';
import fs from 'fs';
import path from 'path';

async function migrate() {
    const client = await pool.connect();
    try {
        const dbDir = path.join(__dirname, '../db');
        const files = fs.readdirSync(dbDir)
            .filter(file => file.endsWith('.sql'))
            .sort();

        // console.log(`Found ${files.length} migration files.`);

        await client.query("BEGIN");

        for (const file of files) {
            const filePath = path.join(dbDir, file);
            const sql = fs.readFileSync(filePath, 'utf8');
            // console.log(`Executing ${file}...`);
            await client.query(sql);
        }

        await client.query("COMMIT");
        // console.log('Migration completed successfully.');
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

