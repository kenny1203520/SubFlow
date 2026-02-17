import { BaseRepository } from './BaseRepository';

export interface GroupRow {
    id: string;
    name: string;
    description?: string;
    service_name?: string;
    service_id?: string;
    website?: string;
    plan_name?: string;
    amount: number;
    service_currency: string;
    payment_currency: string;
    exchange_rate_mode: string;
    fixed_rate_value?: number;
    rounding_method: string;
    rounding_precision: number;
    billing_type: string;
    interval_unit?: string;
    interval_value?: number;
    days_of_week?: number[];
    start_date?: Date;
    end_condition: string;
    end_value?: string;
    max_members: number;
    billing_method: string;
    extra_fee_percentage: number;
    fixed_fee_amount: number;
    created_by: string;
    status: string;
    created_at: Date;
    updated_at: Date;
    next_payment_date?: Date;
}

export class GroupRepository extends BaseRepository {
    async create(data: Partial<GroupRow>): Promise<GroupRow> {
        const res = await this.query(
            `INSERT INTO groups (
                name, service_name, service_id, website, plan_name, amount, 
                service_currency, payment_currency, billing_type, interval_unit, interval_value, 
                max_members, billing_method, created_by, next_payment_date,
                start_date, end_condition, end_value
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18) RETURNING *`,
            [
                data.name, data.service_name, data.service_id, data.website, data.plan_name,
                data.amount, data.service_currency || 'TWD', data.payment_currency || 'TWD',
                data.billing_type || 'recurring', data.interval_unit, data.interval_value || 1,
                data.max_members, data.billing_method || 'equal', data.created_by,
                data.next_payment_date,
                data.start_date, data.end_condition || 'indefinite', data.end_value
            ]
        );
        return res.rows[0];
    }

    async findById(id: string): Promise<GroupRow | null> {
        const res = await this.query('SELECT * FROM groups WHERE id = $1', [id]);
        return res.rows[0] || null;
    }

    async delete(id: string): Promise<void> {
        await this.query('DELETE FROM groups WHERE id = $1', [id]);
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
}
