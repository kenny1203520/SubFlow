import { BaseRepository } from './BaseRepository';

export interface RoleRow {
    id: string;
    name: string;
    description?: string;
}

export interface PermissionRow {
    id: string;
    scope: string;
    action: string;
    resource: string;
}

export class RBACRepository extends BaseRepository {
    async getMemberPermissions(groupId: string, userId: string): Promise<string[]> {
        const res = await this.query(`
            SELECT DISTINCT p.scope || ':' || p.action || ':' || p.resource as permission
            FROM group_member_roles gmr
            JOIN role_permissions rp ON gmr.role_id = rp.role_id
            JOIN permissions p ON rp.permission_id = p.id
            JOIN group_members gm ON gmr.group_member_id = gm.id
            WHERE gm.group_id = $1 AND gm.user_id = $2
        `, [groupId, userId]);

        return res.rows.map(r => r.permission);
    }

    async assignRoleToMember(memberId: string, roleName: string): Promise<void> {
        await this.query(`
            INSERT INTO group_member_roles (group_member_id, role_id)
            SELECT $1, id FROM roles WHERE name = $2
            ON CONFLICT DO NOTHING
        `, [memberId, roleName]);
    }
}
