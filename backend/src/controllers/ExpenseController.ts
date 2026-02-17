import { BaseController } from './BaseController';
import { ExpenseService } from '../services/ExpenseService';

export class ExpenseController extends BaseController {
    private expenseService = new ExpenseService();

    register() {
        this.socket.on("expense:add", (payload, cb) => this.addExpense(payload, cb));
        this.socket.on("expense:list", (payload, cb) => this.listExpenses(payload, cb));
        this.socket.on("expense:get_splits", (payload, cb) => this.getPendingSplits(payload, cb));
        this.socket.on("expense:settle", (payload, cb) => this.settleExpense(payload, cb));
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
