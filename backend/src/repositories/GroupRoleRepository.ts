import { PoolClient } from 'pg';
import { BaseRepository } from './BaseRepository';

export interface GroupRoleRow {
    id: string;
    group_id: string;
    name: string;
    description: string;
    is_system_role: boolean;
}

export class GroupRoleRepository extends BaseRepository {
    async findAll(groupId: string, client?: PoolClient): Promise<GroupRoleRow[]> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn("SELECT * FROM group_roles WHERE group_id = $1 ORDER BY name", [groupId]);
        return res.rows;
    }

    async findByName(groupId: string, name: string, client?: PoolClient): Promise<GroupRoleRow | null> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn("SELECT * FROM group_roles WHERE group_id = $1 AND name = $2", [groupId, name]);
        return res.rows[0] || null;
    }

    async findById(roleId: string, client?: PoolClient): Promise<GroupRoleRow | null> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn("SELECT * FROM group_roles WHERE id = $1", [roleId]);
        return res.rows[0] || null;
    }

    // Role assignment to group members
    async assignRoleToMember(memberId: string, roleId: string, assignedBy: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(
            `INSERT INTO group_member_roles (member_id, role_id, assigned_by) 
             VALUES ($1, $2, $3) ON CONFLICT (member_id, role_id) DO NOTHING`,
            [memberId, roleId, assignedBy]
        );
    }

    async removeRoleFromMember(memberId: string, roleId: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(
            "DELETE FROM group_member_roles WHERE member_id = $1 AND role_id = $2",
            [memberId, roleId]
        );
    }

    async getMemberRoles(memberId: string, client?: PoolClient): Promise<GroupRoleRow[]> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(
            `SELECT gr.* 
             FROM group_member_roles gmr
             JOIN group_roles gr ON gmr.role_id = gr.id
             WHERE gmr.member_id = $1`,
            [memberId]
        );
        return res.rows;
    }

    async listAll(groupId: string, client?: PoolClient): Promise<GroupRoleRow[]> {
        return this.findAll(groupId, client ? client : undefined);
    }

    async create(groupId: string, name: string, description?: string, client?: PoolClient): Promise<GroupRoleRow> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(
            `INSERT INTO group_roles (group_id, name, description, is_system_role)
             VALUES ($1, $2, $3, false) RETURNING *`,
            [groupId, name, description ?? null]
        );
        return res.rows[0];
    }

    async countByGroupId(groupId: string, client?: PoolClient): Promise<number> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn("SELECT COUNT(*)::int AS count FROM group_roles WHERE group_id = $1", [groupId]);
        return res.rows[0]?.count ?? 0;
    }
}

