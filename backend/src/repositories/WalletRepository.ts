import { BaseRepository } from './BaseRepository';

export interface WalletRow {
    id: string;
    user_id: string;
    group_id?: string;
    service_id?: string;
    balance: number;
    currency: string;
    wallet_type: 'root' | 'group' | 'service';
}

export class WalletRepository extends BaseRepository {
    async findByUserId(userId: string): Promise<WalletRow[]> {
        const res = await this.query('SELECT * FROM user_wallets WHERE user_id = $1', [userId]);
        return res.rows;
    }

    async getBalance(walletId: string): Promise<number> {
        const res = await this.query('SELECT balance FROM user_wallets WHERE id = $1', [walletId]);
        return parseFloat(res.rows[0]?.balance || 0);
    }

    async updateBalance(walletId: string, amount: number): Promise<void> {
        await this.query('UPDATE user_wallets SET balance = balance + $1 WHERE id = $2', [amount, walletId]);
    }

    async createTransaction(data: any): Promise<void> {
        await this.query(`
            INSERT INTO wallet_transactions (
                wallet_id, amount, transaction_type, reference_type, reference_id, description
            ) VALUES ($1, $2, $3, $4, $5, $6)
        `, [data.wallet_id, data.amount, data.type, data.ref_type, data.ref_id, data.description]);
    }
}
