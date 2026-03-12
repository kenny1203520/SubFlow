import { Server, Socket } from 'socket.io';
import { SocketController } from './SocketController';
import { ExpenseService } from '../services/ExpenseService';
import { expenseSocketEvents } from '../socket/events';

export class ExpenseController extends SocketController {
    private expenseService = new ExpenseService();

    constructor(io: Server, socket: Socket) {
        super(io, socket);
    }

    register() {
        this.socket.on(expenseSocketEvents.ADD, (payload, cb) => this.addExpense(payload, cb));
        this.socket.on(expenseSocketEvents.LIST, (payload, cb) => this.listExpenses(payload, cb));
        this.socket.on(expenseSocketEvents.GET_SPLITS, (payload, cb) => this.getPendingSplits(payload, cb));
        this.socket.on(expenseSocketEvents.SETTLE, (payload, cb) => this.settleExpense(payload, cb));
    }

    async addExpense(payload: any, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const expense = await this.expenseService.addExpense(userId, payload);
            this.success(cb, { expense });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to add expense");
        }
    }

    async listExpenses(payload: { groupId: string }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const expenses = await this.expenseService.listExpenses(userId, payload.groupId);
            this.success(cb, { expenses });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to list expenses");
        }
    }

    async getPendingSplits(payload: { groupId: string }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const splits = await this.expenseService.getPendingSplits(userId, payload.groupId);
            this.success(cb, { splits });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to get splits");
        }
    }

    async settleExpense(payload: { expenseId: string }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            await this.expenseService.settleExpense(userId, payload.expenseId);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to settle expense");
        }
    }
}
