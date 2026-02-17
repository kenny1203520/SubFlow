import { BaseRepository } from './BaseRepository';

export interface SubscriptionRow {
    id: string;
    group_id: string;
    name: string;
    amount: number;
    billing_cycle: string;
    start_date: Date;
    status: 'active' | 'paused' | 'cancelled';
    next_payment_date?: Date;
}

export class SubscriptionRepository extends BaseRepository {
    async create(data: Partial<SubscriptionRow>): Promise<SubscriptionRow> {
        const res = await this.query(
            "INSERT INTO subscriptions (group_id, name, amount, billing_cycle, start_date) VALUES ($1, $2, $3, $4, $5) RETURNING *",
            [data.group_id, data.name, data.amount, data.billing_cycle, data.start_date]
        );
        return res.rows[0];
    }

    async findByGroupId(groupId: string): Promise<SubscriptionRow[]> {
        const res = await this.query(
            "SELECT * FROM subscriptions WHERE group_id = $1",
            [groupId]
        );
        return res.rows;
    }

    async findAllByUserId(userId: string): Promise<SubscriptionRow[]> {
        const res = await this.query(
            `SELECT s.* FROM subscriptions s
             JOIN group_members gm ON s.group_id = gm.group_id
             WHERE gm.user_id = $1`,
            [userId]
        );
        return res.rows;
    }

    async updateStatus(id: string, status: string): Promise<void> {
        await this.query(
            "UPDATE subscriptions SET status = $1 WHERE id = $2",
            [status, id]
        );
    }
}
