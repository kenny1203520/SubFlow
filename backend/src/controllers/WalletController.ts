import { BaseController } from './BaseController';
import { WalletService } from '../services/WalletService';

export class WalletController extends BaseController {
    private walletService = new WalletService();

    register() {
        this.socket.on("wallet:list", (cb) => this.getWallets(cb));
        this.socket.on("wallet:deposit", (payload, cb) => this.deposit(payload, cb));
        this.socket.on("wallet:transfer", (payload, cb) => this.transfer(payload, cb));
    }

    async getWallets(cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const wallets = await this.walletService.getWallets(userId);
            this.success(cb, { wallets });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to fetch wallets");
        }
    }

    async deposit(payload: { walletId: string, amount: number, proofUrl?: string }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            await this.walletService.deposit(userId, payload.walletId, payload.amount, payload.proofUrl);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Deposit failed");
        }
    }

    async transfer(payload: { fromWalletId: string, toWalletId: string, amount: number }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            await this.walletService.transfer(userId, payload.fromWalletId, payload.toWalletId, payload.amount);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Transfer failed");
        }
    }
}
