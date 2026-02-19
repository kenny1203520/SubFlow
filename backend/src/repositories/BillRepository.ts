import { BaseRepository } from './BaseRepository';

export interface BillRow {
    id: string;
    group_id: string;
    total_amount: number;
    currency: string;
    issue_date: Date;
    due_date?: Date;
    status: 'pending' | 'paid' | 'overdue';
    created_at: Date;
}

export interface BillSplitRow {
    id: string;
    bill_id: string;
    member_id: string;
    amount_owed: number;
    paid_amount: number;
    status: 'pending' | 'paid';
    paid_at?: Date;
}

export class BillRepository extends BaseRepository {
    async findByGroupId(groupId: string): Promise<BillRow[]> {
        const res = await this.query(
            `SELECT * FROM bills WHERE group_id = $1 ORDER BY issue_date DESC`,
            [groupId]
        );
        return res.rows;
    }

    async getBillWithGroupInfo(billId: string): Promise<any> {
        const res = await this.query(`
            SELECT b.*, g.name as group_name, g.currency 
            FROM bills b
            JOIN groups g ON b.group_id = g.id
            WHERE b.id = $1
        `, [billId]);
        return res.rows[0];
    }

    async getSplitsByBillId(billId: string): Promise<any[]> {
        const res = await this.query(`
            SELECT bs.*, 
                   gm.temp_name, 
                   u.username, u.email, u.avatar_url
            FROM bill_splits bs
            JOIN group_members gm ON bs.member_id = gm.id
            LEFT JOIN users u ON gm.user_id = u.id
            WHERE bs.bill_id = $1
        `, [billId]);
        return res.rows;
    }

    async findSplitById(splitId: string): Promise<any> {
        const res = await this.query(`
            SELECT bs.*, b.group_id 
            FROM bill_splits bs
            JOIN bills b ON bs.bill_id = b.id
            WHERE bs.id = $1
        `, [splitId]);
        return res.rows[0];
    }

    async updateSplitAmount(splitId: string, amount: number): Promise<void> {
        await this.query(
            "UPDATE bill_splits SET amount_owed = $1, updated_at = NOW() WHERE id = $2",
            [amount, splitId]
        );
    }

    async updateBillTotalFromSplits(billId: string, userId: string): Promise<void> {
        await this.query(`
            UPDATE bills 
            SET total_amount = (SELECT SUM(amount_owed) FROM bill_splits WHERE bill_id = $1),
                updated_by = $2,
                updated_at = NOW()
            WHERE id = $1
        `, [billId, userId]);
    }
}
