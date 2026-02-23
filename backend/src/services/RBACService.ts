import { RBACRepository, RoleRow, PermissionRow } from '../repositories/RBACRepository';
import { GroupMemberRepository } from '../repositories/GroupMemberRepository';
import { SystemSettingsService } from './SystemSettingsService';

export class RBACService {
    private rbacRepo = new RBACRepository();
    private memberRepo = new GroupMemberRepository();

    /**
     * Check if a user has a specific permission in a group
     */
    async hasPermission(userId: string, groupId: string, action: string, resource: string): Promise<boolean> {
        const permissionString = `group:${action}:${resource}`;
        const permissions = await this.rbacRepo.getMemberPermissions(groupId, userId);
        return permissions.includes(permissionString);
    }

    /**
     * Check permission and throw error if not allowed
     */
    async checkPermission(userId: string, groupId: string, action: string, resource: string): Promise<void> {
        const allowed = await this.hasPermission(userId, groupId, action, resource);
        if (!allowed) {
            throw new Error(`Permission denied: Missing permission to ${action} ${resource}`);
        }
    }

    /**
     * Check if user is the group owner
     */
    async isGroupOwner(userId: string, groupId: string): Promise<boolean> {
        const member = await this.memberRepo.findByGroupAndUser(groupId, userId);
        return member?.role === 'owner';
    }

    /**
     * List all available group roles
     */
    async listGroupRoles(groupId: string): Promise<RoleRow[]> {
        return await this.rbacRepo.listAllGroupRoles(groupId);
    }

    /**
     * Get system setting for groups
     */
    async getSystemSettingsGroups(): Promise<{ groupLimit: number; roleLimit: number; memberLimit: number }> {
        const groupLimitCfg = await SystemSettingsService.getSetting('groups.group_limit');
        const roleLimitCfg = await SystemSettingsService.getSetting('groups.role_limit');
        const memberLimitCfg = await SystemSettingsService.getSetting('groups.member_limit');
        return {
            groupLimit: typeof groupLimitCfg === 'number' ? groupLimitCfg : (groupLimitCfg?.max ?? 100),
            roleLimit: typeof roleLimitCfg === 'number' ? roleLimitCfg : (roleLimitCfg?.max ?? 20),
            memberLimit: typeof memberLimitCfg === 'number' ? memberLimitCfg : (memberLimitCfg?.max ?? 1000)
        };
    }

    /**
     * Create a new custom group role
     */
    async createGroupRole(userId: string, groupId: string, name: string, description: string): Promise<RoleRow> {
        // Note: System-level permission check should happen in controller
        const cfg = await SystemSettingsService.getSetting('groups.role_limit');
        const maxRoles = typeof cfg === 'number' ? cfg : (cfg?.max ?? null);
        if (typeof maxRoles === 'number' && maxRoles > 0) {
            const currentCount = await this.rbacRepo.countGroupRoles(groupId);
            if (currentCount >= maxRoles) {
                throw new Error('Role limit reached for this group');
            }
        }
        
        // Check if role name already exists in this group
        const existingRole = await this.rbacRepo.findGroupRoleByName(groupId, name);
        if (existingRole) {
            throw new Error('A role with this name already exists in this group');
        }
        
        return await this.rbacRepo.createGroupRole(groupId, name, description);
    }

    /**
     * Update an existing custom role
     */
    async updateGroupRole(userId: string, groupId: string, roleId: string, name: string, description: string): Promise<RoleRow> {
        const role = await this.rbacRepo.findGroupRoleById(roleId);
        if (!role) {
            throw new Error('Role not found');
        }
        if (role.group_id && role.group_id !== groupId) {
            throw new Error('Role does not belong to this group');
        }
        if (role.is_system_role) {
            throw new Error('Cannot modify system roles');
        }
        
        // Check if new name conflicts with other roles (case-insensitive)
        if (name.toLowerCase() !== role.name.toLowerCase()) {
            const existingRole = await this.rbacRepo.findGroupRoleByName(groupId, name);
            if (existingRole) {
                throw new Error('A role with this name already exists in this group');
            }
        }
        
        return await this.rbacRepo.updateGroupRole(roleId, groupId, name, description);
    }

    /**
     * Delete a custom role
     */
    async deleteGroupRole(userId: string, groupId: string, roleId: string): Promise<boolean> {
        const role = await this.rbacRepo.findGroupRoleById(roleId);
        if (!role) {
            throw new Error('Role not found');
        }
        if (role.group_id && role.group_id !== groupId) {
            throw new Error('Role does not belong to this group');
        }
        if (role.is_system_role) {
            throw new Error('Cannot delete system roles');
        }
        return await this.rbacRepo.deleteGroupRole(roleId, groupId);
    }

    /**
     * List all permissions for a specific role
     */
    async listRolePermissions(roleId: string): Promise<PermissionRow[]> {
        return await this.rbacRepo.listRolePermissions(roleId);
    }

    /**
     * Grant a permission to a role
     */
    async grantPermissionToRole(userId: string, roleId: string, permissionId: string): Promise<void> {
        // Prevent modifying Group Owner role permissions
        const role = await this.rbacRepo.findGroupRoleById(roleId);
        if (!role) {
            throw new Error('Role not found');
        }
        if (role.name === 'Group Owner') {
            throw new Error('Cannot modify Group Owner role permissions');
        }
        await this.rbacRepo.grantPermissionToRole(roleId, permissionId);
    }

    /**
     * Revoke a permission from a role
     */
    async revokePermissionFromRole(userId: string, roleId: string, permissionId: string): Promise<void> {
        // Prevent modifying Group Owner role permissions
        const role = await this.rbacRepo.findGroupRoleById(roleId);
        if (!role) {
            throw new Error('Role not found');
        }
        if (role.name === 'Group Owner') {
            throw new Error('Cannot modify Group Owner role permissions');
        }
        await this.rbacRepo.revokePermissionFromRole(roleId, permissionId);
    }

    /**
     * List all available permissions (optionally filtered by scope)
     */
    async listAllPermissions(scope?: string): Promise<PermissionRow[]> {
        return await this.rbacRepo.listAllPermissions(scope);
    }

    /**
     * Assign a role to a group member
     */
    async assignRoleToMember(userId: string, groupId: string, memberId: string, roleId: string): Promise<void> {
        await this.checkPermission(userId, groupId, 'assign', 'roles');
        await this.rbacRepo.assignRoleToMember(memberId, roleId, userId);
    }

    /**
     * Remove a role from a group member
     */
    async removeRoleFromMember(userId: string, groupId: string, memberId: string, roleId: string): Promise<void> {
        await this.checkPermission(userId, groupId, 'remove', 'role_assignment');
        await this.rbacRepo.removeRoleFromMember(memberId, roleId);
    }

    /**
     * Grant a direct permission to a user (within a group context)
     */
    async grantDirectPermission(actorUserId: string, groupId: string, targetUserId: string, permissionId: string): Promise<void> {
        await this.checkPermission(actorUserId, groupId, 'grant', 'permissions');
        await this.rbacRepo.grantDirectPermissionToUser(targetUserId, permissionId, actorUserId);
    }

    /**
     * Revoke a direct permission from a user
     */
    async revokeDirectPermission(actorUserId: string, groupId: string, targetUserId: string, permissionId: string): Promise<void> {
        await this.checkPermission(actorUserId, groupId, 'revoke', 'permissions');
        await this.rbacRepo.revokeDirectPermissionFromUser(targetUserId, permissionId);
    }
}
