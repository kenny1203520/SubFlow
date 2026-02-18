import { BaseController } from './BaseController';
import { WalletService } from '../services/WalletService';

export class WalletController extends BaseController {
    private walletService = new WalletService();

    register() {
        this.socket.on("wallet:list", (cb) => this.listWallets(cb));
        this.socket.on("wallet:details", (payload, cb) => this.getWalletDetails(payload, cb));
        this.socket.on("wallet:deposit", (payload, cb) => this.deposit(payload, cb));
        this.socket.on("wallet:transfer", (payload, cb) => this.transfer(payload, cb));
    }

    async listWallets(cb: (res: any) => void) {
        // console.log('[WalletController] listWallets called', this.socket.data.user?.id);
        try {
            const userId = this.socket.data.user.id;
            const wallets = await this.walletService.getUserWallets(userId);
            // console.log('[WalletController] listWallets success', wallets.length);
            this.success(cb, { wallets });
        } catch (error: any) {
            console.error('[WalletController] listWallets error', error);
            this.error(cb, error.message || "Failed to list wallets");
        }
    }

    async getWalletDetails(payload: { walletId: string }, cb: (res: any) => void) {
        try {
            const { wallet, transactions } = await this.walletService.getWalletDetails(payload.walletId);
            this.success(cb, { wallet, transactions });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to get wallet details");
        }
    }

    async deposit(payload: { amount: number, currency?: string }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const wallet = await this.walletService.deposit(userId, payload.amount, payload.currency);
            this.success(cb, { wallet });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to deposit");
        }
    }

    async transfer(payload: { fromWalletId: string, toWalletId: string, amount: number }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const result = await this.walletService.transfer(userId, payload.fromWalletId, payload.toWalletId, payload.amount);
            this.success(cb, { result });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to transfer");
        }
    }
}
