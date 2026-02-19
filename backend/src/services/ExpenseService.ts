import { ExpenseRepository } from '../repositories/ExpenseRepository';
import { GroupMemberRepository } from '../repositories/GroupMemberRepository';
import { pool } from '../db';

export class ExpenseService {
    private expenseRepo = new ExpenseRepository();
    private memberRepo = new GroupMemberRepository();

    async addExpense(userId: string, payload: { groupId: string, amount: number, description: string, splits: { memberId: string, amount: number }[] }) {
        const role = await this.memberRepo.checkRole(payload.groupId, userId);
        if (!role) throw new Error("Not a member");

        const client = await pool.connect();
        try {
            await client.query("BEGIN");

            const expense = await this.expenseRepo.create({
                group_id: payload.groupId,
                paid_by: userId,
                amount: payload.amount,
                description: payload.description
            });

            const splits = payload.splits.map(s => ({
                expense_id: expense.id,
                member_id: s.memberId,
                amount_owed: s.amount
            }));

            await this.expenseRepo.createSplits(splits);

            await client.query("COMMIT");
            return expense;
        } catch (error) {
            await client.query("ROLLBACK");
            throw error;
        } finally {
            client.release();
        }
    }

    async listExpenses(userId: string, groupId: string) {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (!role) throw new Error("Not a member");

        return await this.expenseRepo.findByGroupId(groupId);
    }

    async getPendingSplits(userId: string, groupId: string) {
        const role = await this.memberRepo.checkRole(groupId, userId);
        if (!role) throw new Error("Not a member");

        return await this.expenseRepo.findPendingSplitsByGroupId(groupId);
    }

    async settleExpense(userId: string, expenseId: string) {
        // Logic: currently allows user to settle their own debt.
        // In future, might require receiver to confirm.
        await this.expenseRepo.settleSplit(expenseId, userId);
    }
}
