import { BaseRepository } from './BaseRepository';
import { PoolClient } from 'pg';

export interface WalletRow {
    id: string;
    user_id: string;
    group_id?: string;
    service_id?: string;
    currency: string;
    balance: number;
    created_at: Date;
    updated_at: Date;
}

export interface TransactionRow {
    id: string;
    wallet_id: string;
    type: string;
    amount: number;
    currency: string;
    status: string;
    related_id?: string;
    related_type?: string;
    description?: string;
    created_at: Date;
}

export class WalletRepository extends BaseRepository {

    async createWallet(userId: string, currency: string = 'TWD', groupId?: string, serviceId?: string): Promise<WalletRow> {
        const res = await this.query(
            `INSERT INTO user_wallets (user_id, currency, group_id, service_id, balance)
             VALUES ($1, $2, $3, $4, 0)
             RETURNING *`,
            [userId, currency, groupId, serviceId]
        );
        return res.rows[0];
    }

    async getWalletsByUserId(userId: string): Promise<WalletRow[]> {
        const res = await this.query(
            `SELECT * FROM user_wallets WHERE user_id = $1 ORDER BY created_at ASC`,
            [userId]
        );
        return res.rows;
    }

    async getWalletById(walletId: string): Promise<WalletRow | null> {
        const res = await this.query(
            `SELECT * FROM user_wallets WHERE id = $1`,
            [walletId]
        );
        return res.rows[0] || null;
    }

    async findWallet(userId: string, currency: string, groupId?: string, serviceId?: string): Promise<WalletRow | null> {
        // Build dynamic query based on nullability
        let query = `SELECT * FROM user_wallets WHERE user_id = $1 AND currency = $2`;
        const params: any[] = [userId, currency];

        if (groupId) {
            query += ` AND group_id = $3`;
            params.push(groupId);
        } else {
            query += ` AND group_id IS NULL`;
        }

        if (serviceId) {
            query += ` AND service_id = $${params.length + 1}`;
            params.push(serviceId);
        } else {
            query += ` AND service_id IS NULL`;
        }

        const res = await this.query(query, params);
        return res.rows[0] || null;
    }

    // Transactional: Update Balance
    async updateBalance(walletId: string, amount: number, client: PoolClient): Promise<WalletRow> {
        const res = await client.query(
            `UPDATE user_wallets 
             SET balance = balance + $1, updated_at = NOW() 
             WHERE id = $2 
             RETURNING *`,
            [amount, walletId]
        );
        return res.rows[0];
    }

    // Transactional: Create Transaction Record
    async createTransaction(data: {
        wallet_id: string;
        type: string;
        amount: number;
        currency: string;
        status?: string;
        related_id?: string;
        related_type?: string;
        description?: string;
    }, client: PoolClient): Promise<TransactionRow> {
        const res = await client.query(
            `INSERT INTO wallet_transactions 
             (wallet_id, type, amount, currency, status, related_id, related_type, description)
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
             RETURNING *`,
            [
                data.wallet_id,
                data.type,
                data.amount,
                data.currency,
                data.status || 'completed',
                data.related_id,
                data.related_type,
                data.description
            ]
        );
        return res.rows[0];
    }

    async getTransactions(walletId: string, limit: number = 20, offset: number = 0): Promise<TransactionRow[]> {
        const res = await this.query(
            `SELECT * FROM wallet_transactions 
             WHERE wallet_id = $1 
             ORDER BY created_at DESC 
             LIMIT $2 OFFSET $3`,
            [walletId, limit, offset]
        );
        return res.rows;
    }
}
