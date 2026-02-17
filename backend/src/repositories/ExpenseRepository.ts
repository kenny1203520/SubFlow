import { BaseRepository } from './BaseRepository';

export interface ExpenseRow {
    id: string;
    group_id: string;
    paid_by: string;
    amount: number;
    description: string;
    date: Date;
    created_at: Date;
}

export interface ExpenseSplitRow {
    id: string;
    expense_id: string;
    user_id: string;
    amount_owed: number;
    is_paid: boolean;
}

export class ExpenseRepository extends BaseRepository {
    async create(data: Partial<ExpenseRow>): Promise<ExpenseRow> {
        const res = await this.query(
            "INSERT INTO expenses (group_id, paid_by, amount, description) VALUES ($1, $2, $3, $4) RETURNING *",
            [data.group_id, data.paid_by, data.amount, data.description]
        );
        return res.rows[0];
    }

    async createSplits(splits: Partial<ExpenseSplitRow>[]): Promise<void> {
        for (const split of splits) {
            await this.query(
                "INSERT INTO expense_splits (expense_id, user_id, amount_owed) VALUES ($1, $2, $3)",
                [split.expense_id, split.user_id, split.amount_owed]
            );
        }
    }

    async findByGroupId(groupId: string): Promise<ExpenseRow[]> {
        const res = await this.query(
            "SELECT * FROM expenses WHERE group_id = $1 ORDER BY date DESC",
            [groupId]
        );
        return res.rows;
    }

    async findPendingSplitsByGroupId(groupId: string): Promise<any[]> {
        const res = await this.query(
            `SELECT es.*, e.description, e.date, u.username as payer_name 
             FROM expense_splits es
             JOIN expenses e ON es.expense_id = e.id
             JOIN users u ON e.paid_by = u.id
             WHERE e.group_id = $1 AND es.is_paid = false`,
            [groupId]
        );
        return res.rows;
    }

    async settleSplit(expenseId: string, userId: string): Promise<void> {
        await this.query(
            "UPDATE expense_splits SET is_paid = true WHERE expense_id = $1 AND user_id = $2",
            [expenseId, userId]
        );
    }
}
