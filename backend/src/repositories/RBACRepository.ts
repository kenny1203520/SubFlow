import { PoolClient } from 'pg';
import { BaseRepository } from './BaseRepository';

export interface RoleRow {
    id: string;
    group_id?: string;
    name: string;
    description?: string;
    is_system_role?: boolean;
}

export interface PermissionRow {
    id: string;
    scope: string;
    action: string;
    resource: string;
    description?: string;
}

export class RBACRepository extends BaseRepository {
    /**
     * Get all permissions for a specific member in a group.
     * This includes permissions from:
     * 1. Roles assigned to the member (via group_member_roles)
     * 2. Direct permissions assigned to the user (via permissions_user)
     */
    async getMemberPermissions(groupId: string, userId: string, client?: PoolClient): Promise<string[]> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            WITH member_info AS (
                SELECT id as member_id, role
                FROM group_members
                WHERE group_id = $1 AND user_id = $2
            ),
            role_permissions AS (
                SELECT DISTINCT p.scope, p.action, p.resource
                FROM member_info mi
                JOIN group_member_roles gmr ON gmr.member_id = mi.member_id
                JOIN permissions_group_role pgr ON pgr.role_id = gmr.role_id
                JOIN permissions p ON p.id = pgr.permission_id
            ),
            direct_permissions AS (
                SELECT DISTINCT p.scope, p.action, p.resource
                FROM permissions_user pu
                JOIN permissions p ON p.id = pu.permission_id
                WHERE pu.user_id = $2
                AND p.scope = 'group'
            ),
            legacy_role_permissions AS (
                -- Support legacy 'role' column in group_members for Owner check
                SELECT DISTINCT p.scope, p.action, p.resource
                FROM member_info mi
                JOIN group_roles gr ON gr.group_id = $1 AND gr.name = CASE
                    WHEN mi.role = 'owner' THEN 'Group Owner'
                    WHEN mi.role = 'admin' THEN 'Group Admin'
                    WHEN mi.role = 'treasurer' THEN 'Group Treasurer'
                    WHEN mi.role = 'viewer' THEN 'Group Viewer'
                    ELSE 'Group Member'
                END
                JOIN permissions_group_role pgr ON pgr.role_id = gr.id
                JOIN permissions p ON p.id = pgr.permission_id
            )
            SELECT DISTINCT scope || ':' || action || ':' || resource as permission
            FROM (
                SELECT * FROM role_permissions
                UNION
                SELECT * FROM direct_permissions
                UNION
                SELECT * FROM legacy_role_permissions
            ) all_perms
        `, [groupId, userId]);

        return res.rows.map(r => r.permission);
    }

    async assignRoleToMember(memberId: string, roleId: string, assignedBy: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            INSERT INTO group_member_roles (member_id, role_id, assigned_by)
            VALUES ($1, $2, $3)
            ON CONFLICT (member_id, role_id) DO NOTHING
        `, [memberId, roleId, assignedBy]);
    }

    async removeRoleFromMember(memberId: string, roleId: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            DELETE FROM group_member_roles 
            WHERE member_id = $1 AND role_id = $2
        `, [memberId, roleId]);
    }

    async listAllGroupRoles(groupId: string, client?: PoolClient): Promise<RoleRow[]> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            SELECT * FROM group_roles
            WHERE group_id = $1
            ORDER BY 
                CASE WHEN name = 'Group Owner' THEN 1
                     WHEN name = 'Group Admin' THEN 2
                     WHEN name = 'Group Treasurer' THEN 3
                     WHEN name = 'Group Member' THEN 4
                     WHEN name = 'Group Viewer' THEN 5
                     ELSE 6
                END,
                name
        `, [groupId]);
        return res.rows;
    }

    async findGroupRoleByName(groupId: string, name: string, client?: PoolClient): Promise<RoleRow | null> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(
            'SELECT * FROM group_roles WHERE group_id = $1 AND name = $2',
            [groupId, name]
        );
        return res.rows[0] || null;
    }

    async findGroupRoleById(id: string, client?: PoolClient): Promise<RoleRow | null> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(
            'SELECT * FROM group_roles WHERE id = $1',
            [id]
        );
        return res.rows[0] || null;
    }

    async createGroupRole(groupId: string, name: string, description: string, client?: PoolClient): Promise<RoleRow> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            INSERT INTO group_roles (group_id, name, description, is_system_role)
            VALUES ($1, $2, $3, false)
            RETURNING *
        `, [groupId, name, description]);
        return res.rows[0];
    }

    async updateGroupRole(id: string, groupId: string, name: string, description: string, client?: PoolClient): Promise<RoleRow> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            UPDATE group_roles
            SET name = $2, description = $3, updated_at = CURRENT_TIMESTAMP
            WHERE id = $1 AND group_id = $4 AND is_system_role = false
            RETURNING *
        `, [id, name, description, groupId]);
        return res.rows[0];
    }

    async deleteGroupRole(id: string, groupId: string, client?: PoolClient): Promise<boolean> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            DELETE FROM group_roles
            WHERE id = $1 AND group_id = $2 AND is_system_role = false
        `, [id, groupId]);
        return (res.rowCount || 0) > 0;
    }

    async countGroupRoles(groupId: string, client?: PoolClient): Promise<number> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(
            'SELECT COUNT(*)::int AS count FROM group_roles WHERE group_id = $1',
            [groupId]
        );
        return res.rows[0]?.count ?? 0;
    }

    async listRolePermissions(roleId: string, client?: PoolClient): Promise<PermissionRow[]> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            SELECT p.*
            FROM permissions_group_role pgr
            JOIN permissions p ON p.id = pgr.permission_id
            WHERE pgr.role_id = $1
            ORDER BY p.scope, p.resource, p.action
        `, [roleId]);
        return res.rows;
    }

    async grantPermissionToRole(roleId: string, permissionId: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            INSERT INTO permissions_group_role (role_id, permission_id)
            VALUES ($1, $2)
            ON CONFLICT DO NOTHING
        `, [roleId, permissionId]);
    }

    async revokePermissionFromRole(roleId: string, permissionId: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            DELETE FROM permissions_group_role
            WHERE role_id = $1 AND permission_id = $2
        `, [roleId, permissionId]);
    }

    async listAllPermissions(scope?: string, client?: PoolClient): Promise<PermissionRow[]> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        let query = 'SELECT * FROM permissions';
        const params: any[] = [];
        
        if (scope) {
            query += ' WHERE scope = $1';
            params.push(scope);
        }
        
        query += ' ORDER BY scope, resource, action';
        
        const res = await queryFn(query, params);
        return res.rows;
    }

    async grantDirectPermissionToUser(userId: string, permissionId: string, grantedBy: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            INSERT INTO permissions_user (user_id, permission_id, granted_by)
            VALUES ($1, $2, $3)
            ON CONFLICT DO NOTHING
        `, [userId, permissionId, grantedBy]);
    }

    async revokeDirectPermissionFromUser(userId: string, permissionId: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            DELETE FROM permissions_user
            WHERE user_id = $1 AND permission_id = $2
        `, [userId, permissionId]);
    }
}
