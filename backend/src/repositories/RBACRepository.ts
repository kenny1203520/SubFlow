import { PoolClient } from 'pg';
import { BaseRepository } from './BaseRepository';

export interface RoleRow {
    id: string;
    group_id?: string;
    name: string;
    description?: string;
    is_system_role?: boolean;
    role_level: number;
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
     * List all permissions in the system.
     * Optionally filtered by scope.
     * @param scope 
     * @param client 
     * @returns 
     */
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

    // System-wide role, user and permission management

    /**
     * Get all roles assigned to a specific user (system-wide)
     * @param userId 
     * @param client 
     * @returns 
     */
    async getUserRoles(userId: string, client?: PoolClient): Promise<RoleRow[]> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            SELECT r.*
            FROM user_roles ur
            JOIN system_roles r ON r.id = ur.role_id
            WHERE ur.user_id = $1
        `, [userId]);
        return res.rows;
    }

    /**
     * Get all permissions for a specific user (system-wide)
     * This includes permissions from:
     * 1. Roles assigned to the user (via user_roles)
     * 2. Direct permissions assigned to the user (via permissions_user)
     * @param userId 
     * @param client 
     * @returns 
     */
    async getUserPermissions(userId: string, client?: PoolClient): Promise<string[]> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            WITH user_roles AS (
                SELECT DISTINCT p.scope, p.action, p.resource
                FROM user_roles ur
                JOIN permissions_system_role psr ON psr.role_id = ur.role_id
                JOIN permissions p ON p.id = psr.permission_id
                WHERE ur.user_id = $1
            ),
            direct_permissions AS (
                SELECT DISTINCT p.scope, p.action, p.resource
                FROM permissions_user pu
                JOIN permissions p ON p.id = pu.permission_id
                WHERE pu.user_id = $1
            )
            SELECT DISTINCT scope || ':' || action || ':' || resource as permission
            FROM (
                SELECT * FROM user_roles
                UNION
                SELECT * FROM direct_permissions
            ) all_perms;
        `, [userId]);

        return res.rows.map(r => r.permission);
    }

    /**
     * Grant a direct permission to a user (not via role)
     * @param userId 
     * @param permissionId 
     * @param grantedBy 
     * @param client 
     */
    async grantDirectPermissionToUser(userId: string, permissionId: string, grantedBy: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            INSERT INTO permissions_user (user_id, permission_id, granted_by)
            VALUES ($1, $2, $3)
            ON CONFLICT DO NOTHING
        `, [userId, permissionId, grantedBy]);
    }

    /**
     * Revoke a direct permission from a user (not via role)
     * @param userId 
     * @param permissionId 
     * @param client 
     */
    async revokeDirectPermissionFromUser(userId: string, permissionId: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            DELETE FROM permissions_user
            WHERE user_id = $1 AND permission_id = $2
        `, [userId, permissionId]);
    }

    /**
     * Assign a system role to a user (e.g., Adimisrator, Support Agent)
     * @param userId 
     * @param roleId 
     * @param assignedBy 
     * @param client 
     */
    async assignRoleToUser(userId: string, roleId: string, assignedBy?: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            INSERT INTO user_roles (user_id, role_id, assigned_by)
            VALUES ($1, $2, $3)
            ON CONFLICT (user_id, role_id) DO NOTHING
        `, [userId, roleId, assignedBy]);
    }

    /**
     * Get all direct permissions granted to a user
     * @param userId 
     * @param client 
     * @returns 
     */
    async getDirectPermissionsForUser(userId: string, client?: PoolClient): Promise<any[]> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            SELECT p.id as permission_id, p.scope, p.action, p.resource,
                p.description, pu.*
            FROM permissions_user pu
            JOIN permissions p ON pu.permission_id = p.id
            WHERE pu.user_id = $1
            ORDER BY p.scope, p.action, p.resource
        `, [userId]);
        return res.rows;
    }

    /**
     * Check if a user has a specific system role
     * @param userId 
     * @param roleName 
     * @param client 
     */
    async hasUserRole(userId: string, roleName: string, client?: PoolClient): Promise<boolean> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            SELECT EXISTS (
                SELECT 1
                FROM user_roles ur
                JOIN roles r ON r.id = ur.role_id
                WHERE ur.user_id = $1 AND r.name = $2
                LIMIT 1
            )
        `, [userId, roleName]);
        return res.rows[0]?.exists ?? false;
    }

    /**
     * Remove a system role from a user
     * @param userId 
     * @param roleId 
     * @param client 
     */
    async removeRoleFromUser(userId: string, roleId: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            DELETE FROM user_roles 
            WHERE user_id = $1 AND role_id = $2
        `, [userId, roleId]);
    }

    /**
     * List all system roles in the database
     * @param client 
     * @returns 
     */
    async listAllSystemRoles(client?: PoolClient): Promise<RoleRow[]> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            SELECT *
            FROM system_roles
            ORDER BY role_level ASC, name DESC
        `, []);
        return res.rows;
    }
    
    /**
     * Get a system role by its name
     * @param name 
     * @param client 
     * @returns 
     */
    async getSystemRoleByName(name: string, client?: PoolClient): Promise<RoleRow | null> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn( `
                SELECT *
                FROM system_roles
                WHERE name = $1
                LIMIT 1
            `,
            [name]
        );
        return res.rows[0] || null;
    }

    /**
     * Get a system role by its ID
     * @param id 
     * @param client 
     * @returns 
     */
    async getSystemRoleById(id: string, client?: PoolClient): Promise<RoleRow | null> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            SELECT *
            FROM system_roles
            WHERE id = $1
            LIMIT 1
        `, [id]);
        return res.rows[0] || null;
    }
    
    /**
     * Create a new system role (e.g., Administrator, Support Agent)
     * @param name 
     * @param description 
     * @param roleLevel 
     * @param client 
     * @returns 
     */
    async createSystemRole(name: string, description: string, roleLevel: number, client?: PoolClient): Promise<RoleRow> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            INSERT INTO system_roles (name, description, role_level)
            VALUES ($1, $2, $3)
            RETURNING *
        `, [name, description, roleLevel]);
        return res.rows[0];
    }
    
    /**
     * Update an existing system role's information.
     * @param id 
     * @param name 
     * @param description 
     * @param roleLevel 
     * @param client 
     * @returns 
     */
    async updateSystemRole(id: string, name?: string, description?: string, roleLevel?: number, client?: PoolClient): Promise<RoleRow> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            UPDATE system_roles
            SET name = COALESCE($2, name),
                description = COALESCE($3, description),
                role_level = COALESCE($4, role_level)
            WHERE id = $1
            RETURNING *
        `, [id, name, description, roleLevel]);
        return res.rows[0];
    }

    /**
     * Delete a system role by ID.
     * @param id 
     * @param client 
     * @returns 
     */
    async deleteSystemRole(id: string, client?: PoolClient): Promise<boolean> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            DELETE FROM system_roles
            WHERE id = $1
        `, [id]);
        return (res.rowCount || 0) > 0;
    }
    
    /**
     * List all system permissions available in the database.
     * @param client 
     * @returns 
     */
    async listAllSystemPermissions(client?: PoolClient): Promise<PermissionRow[]> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            SELECT *
            FROM system_permissions
        `, []);
        return res.rows;
    }

    /**
     * Grant a permission to a system role.
     * @param roleId 
     * @param permissionId 
     * @param client 
     */
    async grantPermissionToSystemRole(roleId: string, permissionId: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            INSERT INTO permissions_system_role (role_id, permission_id)
            VALUES ($1, $2)
            ON CONFLICT (role_id, permission_id) DO NOTHING
        `, [roleId, permissionId]);
    }

    /**
     * Revoke a permission from a system role.
     * @param roleId 
     * @param permissionId 
     * @param client 
     */
    async revokePermissionFromSystemRole(roleId: string, permissionId: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            DELETE FROM permissions_system_role
            WHERE role_id = $1 AND permission_id = $2
        `, [roleId, permissionId]);
    }

    // Group-level role, member and permission management

    async getMemberRoles(groupId: string, userId: string, client?: PoolClient): Promise<RoleRow[]> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            SELECT gr.*
            FROM group_members gm
            JOIN group_member_roles gmr ON gmr.member_id = gm.id
            JOIN group_roles gr ON gr.id = gmr.role_id
            WHERE gm.group_id = $1 AND gm.user_id = $2
            ORDER BY gr.role_level ASC, gr.name DESC
        `, [groupId, userId]);
        return res.rows;
    }

    /**
     * Get all permissions for a specific member in a group.
     * This includes permissions from:
     * 1. Roles assigned to the member (via group_member_roles)
     * 2. Direct permissions assigned to the user (via permissions_user)
     * @param groupId
     * @param userId
     * @param client
     * @returns 
     */
    async getMemberPermissions(groupId: string, userId: string, client?: PoolClient): Promise<string[]> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            WITH member_info AS (
                SELECT id as member_id
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
            member_direct_permissions AS (
                SELECT DISTINCT p.scope, p.action, p.resource
                FROM member_info mi
                JOIN permissions_group_member pgm ON pgm.member_id = mi.member_id
                JOIN permissions p ON p.id = pgm.permission_id
                WHERE pgm.group_id = $1
            )
            SELECT DISTINCT scope || ':' || action || ':' || resource as permission
            FROM (
                SELECT * FROM role_permissions
                UNION
                SELECT * FROM direct_permissions
                UNION
                SELECT * FROM member_direct_permissions
            ) all_perms
        `, [groupId, userId]);

        return res.rows.map(r => r.permission);
    }

    /**
     * Assign a role to a group member (e.g., Group Admin, Group Owner)
     * @param memberId 
     * @param roleId 
     * @param assignedBy 
     * @param client 
     */
    async assignRoleToMember(memberId: string, roleId: string, assignedBy?: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            INSERT INTO group_member_roles (member_id, role_id, assigned_by)
            VALUES ($1, $2, $3)
            ON CONFLICT (member_id, role_id) DO NOTHING
        `, [memberId, roleId, assignedBy]);
    }

    /**
     * Check if a user has a specific role in a group.
     * @param groupId 
     * @param userId 
     * @param roleName 
     * @param client 
     * @returns 
     */
    async hasMemberRole(groupId: string, userId: string, roleName: string, client?: PoolClient): Promise<boolean> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            SELECT EXISTS (
                SELECT 1
                FROM group_members gm
                JOIN group_member_roles gmr ON gmr.member_id = gm.id
                JOIN group_roles gr ON gr.id = gmr.role_id
                WHERE gm.group_id = $1 AND gm.user_id = $2 AND gr.name = $3
                LIMIT 1
            )
        `, [groupId, userId, roleName]
        );
        return res.rows[0]?.exists ?? false;
    }

    /**
     * Remove a role from a group member.
     * @param memberId 
     * @param roleId 
     * @param client 
     */
    async removeRoleFromMember(memberId: string, roleId: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            DELETE FROM group_member_roles 
            WHERE member_id = $1 AND role_id = $2
        `, [memberId, roleId]);
    }

    /**
     * List all roles available in a group.
     * ordered by role level (if defined) and then name
     * @param groupId 
     * @param client 
     * @returns 
     */
    async listAllGroupRoles(groupId: string, client?: PoolClient): Promise<RoleRow[]> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            SELECT *
            FROM group_roles
            WHERE group_id = $1
            ORDER BY role_level ASC, name DESC
        `, [groupId]);
        return res.rows;
    }

    /**
     * Get a group role by its name within a specific group.
     * @param groupId 
     * @param name 
     * @param client 
     * @returns 
     */
    async getGroupRoleByName(groupId: string, name: string, client?: PoolClient): Promise<RoleRow | null> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            SELECT *
            FROM group_roles
            WHERE group_id = $1 AND name = $2
            LIMIT 1
        `, [groupId, name]
        );
        return res.rows[0] || null;
    }

    /**
     * Get a group role by its ID.
     * @param id 
     * @param client 
     * @returns 
     */
    async getGroupRoleById(id: string, client?: PoolClient): Promise<RoleRow | null> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            SELECT *
            FROM group_roles
            WHERE id = $1
            LIMIT 1
        `, [id]
        );
        return res.rows[0] || null;
    }

    /**
     * Create a new role within a group (e.g., Group Admin, Group Member)
     * @param groupId 
     * @param name 
     * @param description 
     * @param client 
     * @returns 
     */
    async createGroupRole(groupId: string, name: string, description: string, client?: PoolClient): Promise<RoleRow> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            INSERT INTO group_roles (group_id, name, description, is_system_role)
            VALUES ($1, $2, $3, false)
            RETURNING *
        `, [groupId, name, description]);
        return res.rows[0];
    }

    /**
     * Update an existing group role's information.
     * @param id 
     * @param groupId 
     * @param name 
     * @param description 
     * @param roleLevel 
     * @param client 
     * @returns 
     */
    async updateGroupRole(id: string, groupId: string, name?: string, description?: string, roleLevel?: number, client?: PoolClient): Promise<RoleRow> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            UPDATE group_roles
            SET name = COALESCE($2, name),
                description = COALESCE($3, description),
                role_level = COALESCE($4, role_level)
            WHERE id = $1 AND group_id = $5
            RETURNING *
        `, [id, name, description, roleLevel, groupId]);
        return res.rows[0];
    }

    /**
     * Delete a group role by ID.
     * @param id 
     * @param groupId 
     * @param client 
     * @returns 
     */
    async deleteGroupRole(id: string, groupId: string, client?: PoolClient): Promise<boolean> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            DELETE FROM group_roles
            WHERE id = $1 AND group_id = $2
        `, [id, groupId]);
        return (res.rowCount || 0) > 0;
    }

    /**
     * Count the number of roles in a group.
     * @param groupId 
     * @param client 
     * @returns 
     */
    async countGroupRoles(groupId: string, client?: PoolClient): Promise<number> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            SELECT COUNT(*)::int AS count
            FROM group_roles
            WHERE group_id = $1
        `, [groupId]);
        return res.rows[0]?.count ?? 0;
    }

    /**
     * List all permissions assigned to a specific group role.
     * @param roleId 
     * @param client 
     * @returns 
     */
    async listGroupRolePermissions(roleId: string, client?: PoolClient): Promise<PermissionRow[]> {
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

    /**
     * Grant a permission to a group role.
     * @param roleId 
     * @param permissionId 
     * @param client 
     */
    async grantPermissionToGroupRole(roleId: string, permissionId: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            INSERT INTO permissions_group_role (role_id, permission_id)
            VALUES ($1, $2)
            ON CONFLICT DO NOTHING
        `, [roleId, permissionId]);
    }

    /**
     * Revoke a permission from a group role.
     * @param roleId 
     * @param permissionId 
     * @param client 
     */
    async revokePermissionFromGroupRole(roleId: string, permissionId: string, client?: PoolClient): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            DELETE FROM permissions_group_role
            WHERE role_id = $1 AND permission_id = $2
        `, [roleId, permissionId]);
    }

    /**
     * Check if an actor can manage a target role (actor must have higher privilege)
     * Returns true if actor's max level is strictly lower (higher privilege) than target
     * @param actorUserId 
     * @param groupId 
     * @param targetRoleId 
     * @param client 
     * @returns 
     */
    async canManageRoleInGroup(actorUserId: string, groupId: string, targetRoleId: string, client?: PoolClient): Promise<boolean> {
        // Get actor's roles
        const actorRoles = await this.getMemberRoles(groupId, actorUserId, client);
        if (actorRoles.length === 0) {
            return false; // No roles means no permissions
        }

        let actorMaxLevel = Number.POSITIVE_INFINITY;

        for (const role of actorRoles) {
            if (role.role_level === null) {
                return false; // Roles without defined level cannot manage others
            }
            const permissions = await this.listGroupRolePermissions(role.id, client);
            if (permissions.some(p => p.scope === 'group' && p.action === 'manage' && p.resource === 'roles')) {
                if (role.role_level < actorMaxLevel) {
                    actorMaxLevel = role.role_level;
                }
            }
        }
        
        // If actor has no managing permissions, return false
        if (actorMaxLevel === Number.POSITIVE_INFINITY) {
            return false;
        }
        
        // Get target role's level
        const targetRole = await this.getGroupRoleById(targetRoleId, client);
        if (!targetRole) {
            return false;
        }
        
        // Actor must have strictly lower level (higher privilege) than target
        return actorMaxLevel < targetRole.role_level;
    }

    /**
     * Check if an actor can manage direct permissions for a target member
     * Actor must have strictly higher privilege (lower role_level)
     * @param actorUserId
     * @param groupId
     * @param memberId
     * @param client
     */
    async canManagePermissionToMember(
        actorUserId: string,
        groupId: string,
        memberId: string,
        client?: PoolClient
    ): Promise<boolean> {
        const actorRoles = await this.getMemberRoles(groupId, actorUserId, client);
        if (actorRoles.length === 0) {
            return false;
        }

        const targetRoles = await this.getMemberRoles(groupId, memberId, client);
        if (targetRoles.length === 0) {
            return false;
        }

        const actorMinLevel = Math.min(...actorRoles.map(r => r.role_level ?? Number.POSITIVE_INFINITY));
        const targetMinLevel = Math.min(...targetRoles.map(r => r.role_level ?? Number.POSITIVE_INFINITY));

        if (!Number.isFinite(actorMinLevel) || !Number.isFinite(targetMinLevel)) {
            return false;
        }

        return actorMinLevel < targetMinLevel;
    }

    /**
     * Grant a direct permission to a group member
     * Subject to their role hierarchy level
     * @param groupId 
     * @param memberId 
     * @param permissionId 
     * @param grantedBy 
     * @param client 
     */
    async grantDirectPermissionToGroupMember(
        groupId: string,
        memberId: string,
        permissionId: string,
        grantedBy: string,
        client?: PoolClient
    ): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            INSERT INTO permissions_group_member (group_id, member_id, permission_id, granted_by)
            VALUES ($1, $2, $3, $4)
            ON CONFLICT (group_id, member_id, permission_id) DO NOTHING
        `, [groupId, memberId, permissionId, grantedBy]);
    }

    /**
     * Revoke a direct permission from a group member
     * @param groupId 
     * @param memberId 
     * @param permissionId 
     * @param client 
     */
    async revokeDirectPermissionFromGroupMember(
        groupId: string,
        memberId: string,
        permissionId: string,
        client?: PoolClient
    ): Promise<void> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        await queryFn(`
            DELETE FROM permissions_group_member
            WHERE group_id = $1 AND member_id = $2 AND permission_id = $3
        `, [groupId, memberId, permissionId]);
    }

    /**
     * Get all direct permissions granted to a group member
     * @param groupId 
     * @param memberId 
     * @param client 
     * @returns 
     */
    async getDirectPermissionsForMember(groupId: string, memberId: string, client?: PoolClient): Promise<any[]> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            SELECT p.id as permission_id, p.scope, p.action, p.resource,
                p.description, pgm.*
            FROM permissions_group_member pgm
            JOIN permissions p ON pgm.permission_id = p.id
            WHERE pgm.group_id = $1 AND pgm.member_id = $2
            ORDER BY p.scope, p.action, p.resource
        `, [groupId, memberId]);
        return res.rows;
    }

    /**
     * Get the maximum (highest privilege) role level for a user in a group
     * Lower numeric values = higher privilege
     * @param groupId 
     * @param userId 
     * @param client 
     * @returns The minimum role_level (highest privilege)
     */
    async getUserMaxRoleLevelInGroup(groupId: string, userId: string, client?: PoolClient): Promise<number> {
        const queryFn = client ? (sql: string, params: any[]) => client.query(sql, params) : (sql: string, params: any[]) => this.query(sql, params);
        const res = await queryFn(`
            SELECT MIN(gr.role_level) as max_role_level
            FROM group_members gm
            JOIN group_member_roles gmr ON gmr.member_id = gm.id
            JOIN group_roles gr ON gr.id = gmr.role_id
            WHERE gm.group_id = $1 AND gm.user_id = $2
        `, [groupId, userId]);
        return res.rows[0]?.max_role_level ?? 999; // 999 = no role
    }
}

