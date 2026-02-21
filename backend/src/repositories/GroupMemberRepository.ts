import { BaseRepository } from './BaseRepository';

export interface GroupMemberRow {
    id: string;
    group_id: string;
    user_id?: string;
    temp_name?: string;
    display_name?: string;
    can_self_edit_nickname: boolean;
    role: 'owner' | 'admin' | 'treasurer' | 'member' | 'viewer';
    joined_at?: Date;
    invited_at: Date;
    created_by?: string;
    created_at: Date;
    updated_at: Date;
}

export class GroupMemberRepository extends BaseRepository {
    async addMember(data: Partial<GroupMemberRow>): Promise<GroupMemberRow> {
        const res = await this.query(
            `INSERT INTO group_members (
                group_id, user_id, temp_name, display_name, can_self_edit_nickname, 
                role, joined_at, invited_at, created_by
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING *`,
            [
                data.group_id, 
                data.user_id, 
                data.temp_name,
                data.display_name,
                data.can_self_edit_nickname ?? true,
                data.role || 'member',
                data.joined_at,
                data.invited_at || new Date(),
                data.created_by
            ]
        );
        return res.rows[0];
    }

    async findById(id: string): Promise<GroupMemberRow | null> {
        const res = await this.query("SELECT * FROM group_members WHERE id = $1", [id]);
        return res.rows[0] || null;
    }

    async getMembersByGroupId(groupId: string): Promise<any[]> {
        const res = await this.query(
            `SELECT gm.id as member_id, u.id as user_id, COALESCE(u.username, gm.temp_name) as username, 
                    u.email, gm.role, gm.temp_name, gm.display_name, gm.joined_at, gm.invited_at, u.avatar_url,
                    (
                        SELECT COALESCE(json_agg(json_build_object('id', gr.id, 'name', gr.name, 'is_system_role', gr.is_system_role)), '[]'::json)
                        FROM group_member_roles gmr
                        JOIN group_roles gr ON gmr.role_id = gr.id
                        WHERE gmr.member_id = gm.id
                    ) as "dynamicRoles"
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
            "UPDATE group_members SET user_id = $1, joined_at = NOW() WHERE id = $2",
            [userId, memberId]
        );
    }

    async updateRole(memberId: string, role: string): Promise<void> {
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

    async updateStatus(memberId: string, joined_at: Date | null): Promise<void> {
        // In the new schema, "status" is implied by joined_at or elsewhere.
        // Assuming we update joined_at to mark someone as active.
        await this.query(
            "UPDATE group_members SET joined_at = $1 WHERE id = $2",
            [joined_at, memberId]
        );
    }

    async updateDisplayName(memberId: string, displayName: string): Promise<void> {
        await this.query(
            "UPDATE group_members SET display_name = $1 WHERE id = $2",
            [displayName, memberId]
        );
    }
}
