import { Pool, PoolClient } from 'pg';
import { pool } from '../db';
import { getContext } from '../utils/context';

export abstract class BaseRepository {
    protected pool: Pool = pool;

    protected async query(text: string, params?: any[]) {
        const client = await this.pool.connect();
        try {
            const context = getContext();
            const userId = context?.userId;

            if (userId) {
                await client.query('BEGIN');
                await client.query(`SET LOCAL app.current_user_id = $1`, [userId]);
                const res = await client.query(text, params);
                await client.query('COMMIT');
                return res;
            } else {
                // For public access or system tasks without user context
                // You might want to default to a system user or allow public access depending on policy
                // For now, we execute without setting the variable, which means policies relying on it might block access (default deny)
                // OR we can set it to null explicitly if policies handle it.
                // await client.query(`SET LOCAL app.current_user_id = NULL`); 
                return await client.query(text, params);
            }
        } catch (e) {
            await client.query('ROLLBACK');
            throw e;
        } finally {
            client.release();
        }
    }

    protected async getClient(): Promise<PoolClient> {
        // If getting a raw client, the caller is responsible for setting context if needed!
        // Or we can wrap it here too, but it's trickier with transactions.
        return await this.pool.connect();
    }
}
