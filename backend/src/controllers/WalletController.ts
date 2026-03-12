import { Server, Socket } from 'socket.io';
import { SocketController } from './SocketController';
import { WalletService } from '../services/WalletService';
import { walletSocketEvents } from '../socket/events';

export class WalletController extends SocketController {
    private walletService = new WalletService();

    constructor(io: Server, socket: Socket) {
        super(io, socket);
    }

    register() {
        this.socket.on(walletSocketEvents.LIST, (...args: any[]) => this.listWallets(this.resolveAck(...args) as any));
        this.socket.on(walletSocketEvents.DETAILS, (payload, cb) => this.getWalletDetails(payload, cb));
        this.socket.on(walletSocketEvents.DEPOSIT, (payload, cb) => this.deposit(payload, cb));
        this.socket.on(walletSocketEvents.TRANSFER, (payload, cb) => this.transfer(payload, cb));
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
