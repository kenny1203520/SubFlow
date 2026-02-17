import { WalletRepository } from '../repositories/WalletRepository';
import { pool } from '../db';

export class WalletService {
    private walletRepo = new WalletRepository();

    async getWallets(userId: string) {
        return await this.walletRepo.findByUserId(userId);
    }

    async deposit(userId: string, walletId: string, amount: number, proofUrl?: string) {
        const client = await pool.connect();
        try {
            await client.query("BEGIN");

            // In a real system, deposit would be 'pending' until approved.
            // For now, let's implement the core balance update logic.

            await this.walletRepo.updateBalance(walletId, amount);
            await this.walletRepo.createTransaction({
                wallet_id: walletId,
                amount: amount,
                type: 'deposit',
                description: 'User deposit'
            });

            await client.query("COMMIT");
        } catch (error) {
            await client.query("ROLLBACK");
            throw error;
        } finally {
            client.release();
        }
    }

    async transfer(userId: string, fromWalletId: string, toWalletId: string, amount: number) {
        const client = await pool.connect();
        try {
            await client.query("BEGIN");

            const balance = await this.walletRepo.getBalance(fromWalletId);
            if (balance < amount) throw new Error("Insufficient balance");

            await this.walletRepo.updateBalance(fromWalletId, -amount);
            await this.walletRepo.updateBalance(toWalletId, amount);

            await this.walletRepo.createTransaction({
                wallet_id: fromWalletId,
                amount: -amount,
                type: 'transfer_out',
                description: `Transfer to wallet ${toWalletId}`
            });

            await this.walletRepo.createTransaction({
                wallet_id: toWalletId,
                amount: amount,
                type: 'transfer_in',
                description: `Transfer from wallet ${fromWalletId}`
            });

            await client.query("COMMIT");
        } catch (error) {
            await client.query("ROLLBACK");
            throw error;
        } finally {
            client.release();
        }
    }
}
