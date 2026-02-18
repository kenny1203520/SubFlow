import { BaseRepository } from './BaseRepository';

export interface SubscriptionRow {
    id: string;
    owner_id: string;
    service_name: string;
    amount: number;
    cycle: 'monthly' | 'yearly';
    status: 'active' | 'paused' | 'cancelled';
    next_payment_date?: Date;
    created_at: Date;
    updated_at: Date;
}

export class SubscriptionRepository extends BaseRepository {
    async create(data: Partial<SubscriptionRow>): Promise<SubscriptionRow> {
        const res = await this.query(
            `INSERT INTO subscriptions (
                owner_id, service_name, amount, cycle, status, next_payment_date
            ) VALUES ($1, $2, $3, $4, $5, $6) RETURNING *`,
            [
                data.owner_id,
                data.service_name,
                data.amount,
                data.cycle,
                data.status || 'active',
                data.next_payment_date
            ]
        );
        return res.rows[0];
    }

    // Deprecated or Needs Update: Subscriptions are now user-owned, not group-owned in this schema.
    // Keeping this but querying by owner_id for now if we assume 'groupId' was logically mapped to a user?
    // Or just remove usage. For "My Subscriptions", we use findAllByUserId.
    // If we need group subscriptions, schema needs update. For now, let's strictly follow the User-Owner schema.

    async findAllByUserId(userId: string): Promise<SubscriptionRow[]> {
        const res = await this.query(
            "SELECT * FROM subscriptions WHERE owner_id = $1 ORDER BY created_at DESC",
            [userId]
        );
        return res.rows;
    }

    async updateStatus(id: string, status: string): Promise<void> {
        await this.query(
            "UPDATE subscriptions SET status = $1, updated_at = NOW() WHERE id = $2",
            [status, id]
        );
    }
}
