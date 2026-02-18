import { pool } from './src/db';

async function check() {
    try {
        // console.log("Checking subscriptions table...");
        const res = await pool.query("SELECT column_name FROM information_schema.columns WHERE table_name = 'subscriptions'");
        // console.log("Subscriptions columns:", res.rows.map(r => r.column_name));

        // console.log("Checking user_wallets table...");
        const res2 = await pool.query("SELECT column_name FROM information_schema.columns WHERE table_name = 'user_wallets'");
        // console.log("UserWallets columns:", res2.rows.map(r => r.column_name));
    } catch (e) {
        console.error("Error:", e);
    } finally {
        await pool.end();
    }
}
check();
