import { BaseRepository } from './BaseRepository';

export interface GroupRoleRow {
    id: string;
    name: string;
    description: string;
    is_system_role: boolean;
}

export class GroupRoleRepository extends BaseRepository {
    async findAll(): Promise<GroupRoleRow[]> {
        const res = await this.query("SELECT * FROM group_roles ORDER BY name");
        return res.rows;
    }

    async findByName(name: string): Promise<GroupRoleRow | null> {
        const res = await this.query("SELECT * FROM group_roles WHERE name = $1", [name]);
        return res.rows[0] || null;
    }

    // Role assignment to group members
    async assignRoleToMember(memberId: string, roleId: string, assignedBy: string): Promise<void> {
        await this.query(
            `INSERT INTO group_member_roles (member_id, role_id, assigned_by) 
             VALUES ($1, $2, $3) ON CONFLICT (member_id, role_id) DO NOTHING`,
            [memberId, roleId, assignedBy]
        );
    }

    async removeRoleFromMember(memberId: string, roleId: string): Promise<void> {
        await this.query(
            "DELETE FROM group_member_roles WHERE member_id = $1 AND role_id = $2",
            [memberId, roleId]
        );
    }

    async getMemberRoles(memberId: string): Promise<GroupRoleRow[]> {
        const res = await this.query(
            `SELECT gr.* 
             FROM group_member_roles gmr
             JOIN group_roles gr ON gmr.role_id = gr.id
             WHERE gmr.member_id = $1`,
            [memberId]
        );
        return res.rows;
    }

    async listAll(): Promise<GroupRoleRow[]> {
        return this.findAll();
    }

    async create(name: string, description?: string): Promise<GroupRoleRow> {
        const res = await this.query(
            `INSERT INTO group_roles (name, description, is_system_role)
             VALUES ($1, $2, false) RETURNING *`,
            [name, description ?? null]
        );
        return res.rows[0];
    }
}

