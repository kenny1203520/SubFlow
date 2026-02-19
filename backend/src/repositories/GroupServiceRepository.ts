import { BaseRepository } from './BaseRepository';

export interface GroupServiceRow {
    id: string;
    group_id: string;
    service_id?: string;
    service_name?: string;
    website?: string;
    plan_name?: string;
    status: 'active' | 'paused' | 'cancelled';

    // Billing Amount & Currency
    amount: number;
    service_currency: string;
    payment_currency: string;
    
    exchange_rate_mode: 'fixed_custom' | 'fixed_current' | 'floating';
    fixed_rate_value?: number;

    // Rounding Rules
    rounding_method: 'round' | 'ceil' | 'floor';
    rounding_precision: number;

    // Billing Cycle Configuration
    billing_type: 'once' | 'recurring';
    interval_unit?: 'minute' | 'hour' | 'day' | 'week' | 'month' | 'quarter' | 'year';
    interval_value: number;
    days_of_week?: number[];
    start_date?: Date;
    end_condition: 'never' | 'occurrences' | 'on_date';
    end_value?: string;

    // Billing Method
    billing_method: 'equal' | 'fixed' | 'percentage';
    
    // Fees & Taxes
    extra_fee_percentage: number;
    fixed_fee_amount: number;

    next_payment_date?: Date;
    
    created_by: string;
    created_at: Date;
    updated_at: Date;
}

export interface GroupServiceMemberRow {
    id: string;
    group_service_id: string;
    member_id: string;
    share_ratio?: number;
    fixed_amount?: number;
    role: 'owner' | 'member';
    status: 'active' | 'paused' | 'cancelled';
    started_at?: Date;
    ended_at?: Date;
    created_at: Date;
    updated_at: Date;
}

export class GroupServiceRepository extends BaseRepository {
    async create(data: Partial<GroupServiceRow>): Promise<GroupServiceRow> {
        const res = await this.query(
            `INSERT INTO group_services (
                group_id, service_id, service_name, website, plan_name, status,
                amount, service_currency, payment_currency, exchange_rate_mode, fixed_rate_value,
                rounding_method, rounding_precision, billing_type, interval_unit, interval_value,
                days_of_week, start_date, end_condition, end_value, billing_method,
                extra_fee_percentage, fixed_fee_amount, next_payment_date, created_by
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25) RETURNING *`,
            [
                data.group_id, data.service_id, data.service_name, data.website, data.plan_name, data.status || 'active',
                data.amount || 0, data.service_currency || 'TWD', data.payment_currency || 'TWD', data.exchange_rate_mode || 'floating', data.fixed_rate_value,
                data.rounding_method || 'round', data.rounding_precision || 0, data.billing_type || 'recurring', data.interval_unit, data.interval_value || 1,
                data.days_of_week, data.start_date, data.end_condition || 'never', data.end_value, data.billing_method || 'equal',
                data.extra_fee_percentage || 0, data.fixed_fee_amount || 0, data.next_payment_date, data.created_by
            ]
        );
        return res.rows[0];
    }

    async findById(id: string): Promise<GroupServiceRow | null> {
        const res = await this.query('SELECT * FROM group_services WHERE id = $1', [id]);
        return res.rows[0] || null;
    }

    async findByGroupId(groupId: string): Promise<GroupServiceRow[]> {
        const res = await this.query('SELECT * FROM group_services WHERE group_id = $1', [groupId]);
        return res.rows;
    }

    async update(id: string, data: Partial<GroupServiceRow>): Promise<GroupServiceRow | null> {
        const keys = Object.keys(data).filter(k => k !== 'id' && k !== 'created_at' && k !== 'created_by' && k !== 'group_id');
        if (keys.length === 0) return this.findById(id);

        const setClause = keys.map((k, i) => `${k} = $${i + 2}`).join(', ');
        const values = keys.map(k => (data as any)[k]);

        const res = await this.query(
            `UPDATE group_services SET ${setClause}, updated_at = NOW() WHERE id = $1 RETURNING *`,
            [id, ...values]
        );
        return res.rows[0] || null;
    }

    async delete(id: string): Promise<void> {
        await this.query('DELETE FROM group_services WHERE id = $1', [id]);
    }

    // Service Members (Splits) management
    async addServiceMember(data: Partial<GroupServiceMemberRow>): Promise<GroupServiceMemberRow> {
        const res = await this.query(
            `INSERT INTO group_service_members (
                group_service_id, member_id, share_ratio, fixed_amount, role, status, started_at, ended_at
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *`,
            [
                data.group_service_id, data.member_id, data.share_ratio, data.fixed_amount,
                data.role || 'member', data.status || 'active', data.started_at, data.ended_at
            ]
        );
        return res.rows[0];
    }

    async getServiceMembers(groupServiceId: string): Promise<GroupServiceMemberRow[]> {
        const res = await this.query('SELECT * FROM group_service_members WHERE group_service_id = $1', [groupServiceId]);
        return res.rows;
    }

    async updateServiceMember(id: string, data: Partial<GroupServiceMemberRow>): Promise<GroupServiceMemberRow | null> {
        const keys = Object.keys(data).filter(k => k !== 'id' && k !== 'created_at' && k !== 'group_service_id' && k !== 'member_id');
        if (keys.length === 0) {
             const res = await this.query('SELECT * FROM group_service_members WHERE id = $1', [id]);
             return res.rows[0] || null;
        }

        const setClause = keys.map((k, i) => `${k} = $${i + 2}`).join(', ');
        const values = keys.map(k => (data as any)[k]);

        const res = await this.query(
            `UPDATE group_service_members SET ${setClause}, updated_at = NOW() WHERE id = $1 RETURNING *`,
            [id, ...values]
        );
        return res.rows[0] || null;
    }

    async removeServiceMember(id: string): Promise<void> {
        await this.query('DELETE FROM group_service_members WHERE id = $1', [id]);
    }
}
