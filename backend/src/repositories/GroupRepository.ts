import { BaseRepository } from './BaseRepository';

export interface GroupRow {
    id: string;
    name: string;
    service_name?: string;
    service_id?: string;
    website?: string;
    plan_name?: string;
    amount: number;
    currency: string;
    billing_cycle: string;
    max_members: number;
    billing_method: string;
    created_by: string;
    created_at: Date;
    next_payment_date?: Date;
}

export class GroupRepository extends BaseRepository {
    async create(data: Partial<GroupRow>): Promise<GroupRow> {
        const res = await this.query(
            `INSERT INTO groups (
                name, service_name, service_id, website, plan_name, amount, 
                currency, billing_cycle, max_members, billing_method, created_by,
                next_payment_date
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING *`,
            [
                data.name, data.service_name, data.service_id, data.website, data.plan_name,
                data.amount, data.currency || 'TWD', data.billing_cycle,
                data.max_members, data.billing_method || 'equal', data.created_by,
                data.next_payment_date
            ]
        );
        return res.rows[0];
    }

    async findByUserId(userId: string): Promise<GroupRow[]> {
        const res = await this.query(
            `SELECT g.* FROM groups g 
             JOIN group_members gm ON g.id = gm.group_id 
             WHERE gm.user_id = $1`,
            [userId]
        );
        return res.rows;
    }

    async findById(id: string): Promise<GroupRow | null> {
        const res = await this.query('SELECT * FROM groups WHERE id = $1', [id]);
        return res.rows[0] || null;
    }
}
