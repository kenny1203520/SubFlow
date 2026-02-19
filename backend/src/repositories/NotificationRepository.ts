import { BaseRepository } from './BaseRepository';

export interface NotificationRow {
    id: string;
    user_id: string;
    type: string;
    title: string;
    message: string;
    is_read: boolean;
    is_solved: boolean;
    is_deleted: boolean;
    data: any;
    created_at: Date;
    updated_at: Date;
}

export class NotificationRepository extends BaseRepository {
    async create(data: Partial<NotificationRow>): Promise<NotificationRow> {
        const res = await this.query(
            "INSERT INTO notifications (user_id, type, title, message, data) VALUES ($1, $2, $3, $4, $5) RETURNING *",
            [data.user_id, data.type, data.title, data.message, data.data]
        );
        return res.rows[0];
    }

    async findByUserId(userId: string, limit: number = 20, offset: number = 0): Promise<NotificationRow[]> {
        const res = await this.query(
            "SELECT * FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3",
            [userId, limit, offset]
        );
        return res.rows;
    }

    async countUnread(userId: string): Promise<number> {
        const res = await this.query(
            "SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false",
            [userId]
        );
        return parseInt(res.rows[0].count);
    }

    async markAsRead(ids: string[], userId: string): Promise<void> {
        if (ids.length === 0) return;
        const placeholders = ids.map((_, i) => `$${i + 2}`).join(',');
        await this.query(
            `UPDATE notifications SET is_read = true WHERE user_id = $1 AND id IN (${placeholders})`,
            [userId, ...ids]
        );
    }

    async markAllAsRead(userId: string): Promise<void> {
        await this.query(
            "UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false",
            [userId]
        );
    }
}
