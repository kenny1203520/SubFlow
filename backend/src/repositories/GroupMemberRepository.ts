import { BaseRepository } from './BaseRepository';

export interface GroupMemberRow {
    id: string;
    group_id: string;
    user_id?: string;
    temp_name?: string;
    role: 'admin' | 'member' | 'treasurer';
    created_at: Date;
}

export class GroupMemberRepository extends BaseRepository {
    async addMember(data: Partial<GroupMemberRow>): Promise<void> {
        await this.query(
            "INSERT INTO group_members (group_id, user_id, temp_name, role) VALUES ($1, $2, $3, $4)",
            [data.group_id, data.user_id, data.temp_name, data.role || 'member']
        );
    }

    async findById(id: string): Promise<GroupMemberRow | null> {
        const res = await this.query("SELECT * FROM group_members WHERE id = $1", [id]);
        return res.rows[0] || null;
    }

    async getMembersByGroupId(groupId: string): Promise<any[]> {
        const res = await this.query(
            `SELECT gm.id as member_id, u.id as user_id, COALESCE(u.username, gm.temp_name) as username, 
                    u.email, gm.role, gm.temp_name 
             FROM group_members gm 
             LEFT JOIN users u ON gm.user_id = u.id 
             WHERE gm.group_id = $1`,
            [groupId]
        );
        return res.rows;
    }

    async checkRole(groupId: string, userId: string): Promise<string | null> {
        const res = await this.query(
            "SELECT role FROM group_members WHERE group_id = $1 AND user_id = $2",
            [groupId, userId]
        );
        return res.rows[0]?.role || null;
    }

    async bindMember(memberId: string, userId: string): Promise<void> {
        await this.query(
            "UPDATE group_members SET user_id = $1, temp_name = NULL WHERE id = $2",
            [userId, memberId]
        );
    }

    async updateRole(memberId: string, role: 'admin' | 'member' | 'treasurer'): Promise<void> {
        await this.query(
            "UPDATE group_members SET role = $1 WHERE id = $2",
            [role, memberId]
        );
    }

    async remove(memberId: string): Promise<void> {
        await this.query(
            "DELETE FROM group_members WHERE id = $1",
            [memberId]
        );
    }

    async removeMember(groupId: string, userId: string): Promise<void> {
        await this.query(
            "DELETE FROM group_members WHERE group_id = $1 AND user_id = $2",
            [groupId, userId]
        );
    }
}
