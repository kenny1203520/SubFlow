import { Pool, PoolClient } from 'pg';
import { pool } from '../db';

export abstract class BaseRepository {
    protected pool: Pool = pool;

    protected async query(text: string, params?: any[]) {
        return await this.pool.query(text, params);
    }

    protected async getClient(): Promise<PoolClient> {
        return await this.pool.connect();
    }
}
