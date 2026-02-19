import { BaseRepository } from './BaseRepository';

export interface GroupRow {
    id: string;
    name: string;
    description?: string;
    max_members: number;
    status: 'active' | 'paused' | 'cancelled';
    created_by: string;
    created_at: Date;
    updated_at: Date;
}

export class GroupRepository extends BaseRepository {
    async create(data: Partial<GroupRow>): Promise<GroupRow> {
        const res = await this.query(
            `INSERT INTO groups (
                name, description, max_members, status, created_by
            ) VALUES ($1, $2, $3, $4, $5) RETURNING *`,
            [
                data.name, 
                data.description, 
                data.max_members || 1, 
                data.status || 'active', 
                data.created_by
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

    async update(id: string, data: Partial<GroupRow>): Promise<GroupRow | null> {
        const keys = Object.keys(data).filter(k => k !== 'id' && k !== 'created_at' && k !== 'created_by');
        if (keys.length === 0) return this.findById(id);

        const setClause = keys.map((k, i) => `${k} = $${i + 2}`).join(', ');
        const values = keys.map(k => (data as any)[k]);

        const res = await this.query(
            `UPDATE groups SET ${setClause}, updated_at = NOW() WHERE id = $1 RETURNING *`,
            [id, ...values]
        );
        return res.rows[0] || null;
    }
}
