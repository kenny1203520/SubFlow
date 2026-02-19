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
    expense_id: string;
    member_id: string;
    amount_owed: number;
    status: 'pending' | 'paid';
}

export class ExpenseRepository extends BaseRepository {
    async create(data: Partial<ExpenseRow>): Promise<ExpenseRow> {
        const res = await this.query(
            "INSERT INTO expenses (group_id, paid_by, amount, description) VALUES ($1, $2, $3, $4) RETURNING *",
            [data.group_id, data.paid_by, data.amount, data.description]
        );
        return res.rows[0];
    }

    async createSplits(splits: { expense_id: string, member_id: string, amount_owed: number }[]): Promise<void> {
        for (const split of splits) {
            await this.query(
                "INSERT INTO expense_splits (expense_id, member_id, amount_owed) VALUES ($1, $2, $3)",
                [split.expense_id, split.member_id, split.amount_owed]
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
            `SELECT es.*, e.description, e.date, 
                    COALESCE(u.username, gm.temp_name) as payer_name,
                    u.username as real_username,
                    gm.temp_name
             FROM expense_splits es
             JOIN expenses e ON es.expense_id = e.id
             JOIN group_members gm ON es.member_id = gm.id
             LEFT JOIN users u ON gm.user_id = u.id
             WHERE e.group_id = $1 AND es.status = 'pending'`,
            [groupId]
        );
        return res.rows;
    }

    async settleSplit(expenseId: string, userId: string): Promise<void> {
        await this.query(
            `UPDATE expense_splits es
             SET status = 'paid' 
             FROM group_members gm
             WHERE es.member_id = gm.id 
             AND es.expense_id = $1 
             AND gm.user_id = $2`,
            [expenseId, userId]
        );
    }
}
