import { WalletRepository } from '../repositories/WalletRepository';
import { pool } from '../db'; // Need direct pool access for transactions

export class WalletService {
    private walletRepo = new WalletRepository();

    // Ensure a Global (Root) Wallet exists for the user
    async getOrCreateGlobalWallet(userId: string, currency: string = 'TWD') {
        let wallet = await this.walletRepo.findWallet(userId, currency);
        if (!wallet) {
            wallet = await this.walletRepo.createWallet(userId, currency);
        }
        return wallet;
    }

    async getUserWallets(userId: string) {
        console.log('[WalletService] getUserWallets', userId);
        // Ensure global wallet exists before returning list
        await this.getOrCreateGlobalWallet(userId);
        const wallets = await this.walletRepo.getWalletsByUserId(userId);
        console.log('[WalletService] getUserWallets found', wallets.length);
        return wallets;
    }

    async getWalletDetails(walletId: string) {
        const wallet = await this.walletRepo.getWalletById(walletId);
        if (!wallet) throw new Error("Wallet not found");

        const transactions = await this.walletRepo.getTransactions(walletId);
        return { wallet, transactions };
    }

    async deposit(userId: string, amount: number, currency: string = 'TWD', method: string = 'manual') {
        if (amount <= 0) throw new Error("Deposit amount must be positive");

        const client = await pool.connect();
        try {
            await client.query('BEGIN');

            // 1. Get or Create Global Wallet
            let wallet = await this.walletRepo.findWallet(userId, currency);
            if (!wallet) {
                wallet = await this.walletRepo.createWallet(userId, currency);
            }

            // 2. Update Balance
            const updatedWallet = await this.walletRepo.updateBalance(wallet.id, amount, client);

            // 3. Log Transaction
            await this.walletRepo.createTransaction({
                wallet_id: wallet.id,
                type: 'deposit',
                amount: amount,
                currency: currency,
                description: `Deposit via ${method}`,
                status: 'completed'
            }, client);

            await client.query('COMMIT');
            return updatedWallet;
        } catch (e) {
            await client.query('ROLLBACK');
            throw e;
        } finally {
            client.release();
        }
    }

    async transfer(userId: string, fromWalletId: string, toWalletId: string, amount: number) {
        if (amount <= 0) throw new Error("Transfer amount must be positive");
        if (fromWalletId === toWalletId) throw new Error("Cannot transfer to same wallet");

        const client = await pool.connect();
        try {
            await client.query('BEGIN');

            const fromWallet = await this.walletRepo.getWalletById(fromWalletId);
            const toWallet = await this.walletRepo.getWalletById(toWalletId);

            if (!fromWallet || !toWallet) throw new Error("Wallet not found");
            if (fromWallet.user_id !== userId) throw new Error("Unauthorized"); // Simple check
            if (fromWallet.currency !== toWallet.currency) throw new Error("Currency mismatch");
            if (fromWallet.balance < amount) throw new Error("Insufficient funds");

            // Deduct from Source
            await this.walletRepo.updateBalance(fromWalletId, -amount, client);
            await this.walletRepo.createTransaction({
                wallet_id: fromWalletId,
                type: 'transfer_out',
                amount: -amount,
                currency: fromWallet.currency,
                description: `Transfer to wallet ${toWalletId}`,
                related_id: toWalletId,
                related_type: 'wallet'
            }, client);

            // Add to Destination
            await this.walletRepo.updateBalance(toWalletId, amount, client);
            await this.walletRepo.createTransaction({
                wallet_id: toWalletId,
                type: 'transfer_in',
                amount: amount,
                currency: toWallet.currency,
                description: `Transfer from wallet ${fromWalletId}`,
                related_id: fromWalletId,
                related_type: 'wallet'
            }, client);

            await client.query('COMMIT');
            return { from: fromWalletId, to: toWalletId, amount };
        } catch (e) {
            await client.query('ROLLBACK');
            throw e;
        } finally {
            client.release();
        }
    }
}
