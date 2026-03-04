import { RBACRepository, RoleRow, PermissionRow } from '../repositories/RBACRepository';
import { SystemSettingsService } from './SystemSettingsService';

export class RBACService {
    private rbacRepo = new RBACRepository();

    /**
     * List all available permissions (optionally filtered by scope)
     * @param scope 
     * @returns 
     */
    async listAllPermissions(scope?: string): Promise<any[]> {
        const permissions = await this.rbacRepo.listAllPermissions(scope);
        // Add computed name field for frontend
        return permissions.map(p => ({
            ...p,
            name: `${p.scope}:${p.action}:${p.resource}`
        }));
    }

    /**
     * Get all roles assigned to a user
     * @param userId 
     * @returns 
     */
    async getUserRoles(userId: string): Promise<RoleRow[]> {
        return await this.rbacRepo.getUserRoles(userId);
    }

    /**
     * Check if a user has a specific permission in a group or system-wide
     * @param userId 
     * @param scope
     * @param action 
     * @param resource 
     * @param groupId Optional; For 'group' scope, groupId is required.
     * @returns 
     */
    async hasPermission(userId: string, scope: string, action: string, resource: string, groupId?: string): Promise<boolean> {
        if (scope === 'group') {
            if (!groupId) {
                throw new Error('Group ID is required for group-scoped permissions');
            }
            const permissionString = `group:${action}:${resource}`;
            const permissions = await this.rbacRepo.getMemberPermissions(groupId, userId);
            return permissions.includes(permissionString);
        }
        const permissionString = `${scope}:${action}:${resource}`;
        const permissions = await this.rbacRepo.getUserPermissions(userId);
        return permissions.includes(permissionString);
    }

    /**
     * Check permission and throw error if not allowed
     * @param userId 
     * @param groupId 
     * @param action 
     * @param resource 
     */
    async checkPermission(userId: string, groupId: string, action: string, resource: string): Promise<void> {
        const allowed = await this.hasPermission(userId, groupId, action, resource);
        if (!allowed) {
            throw new Error(`Permission denied: Missing permission to ${action} ${resource}`);
        }
    }

    /**
     * Check if user is the group owner
     * @param userId 
     * @param groupId 
     * @returns 
     */
    async isGroupOwner(userId: string, groupId: string): Promise<boolean> {
        return await this.rbacRepo.hasMemberRole(groupId, userId, 'Group Owner');
    }

    async getMemberRoles(groupId: string, memberId: string): Promise<RoleRow[]> {
        return await this.rbacRepo.getMemberRoles(groupId, memberId);
    }

    async hasMemberRole(userId: string, groupId: string, roleName: string): Promise<boolean> {
        return await this.rbacRepo.hasMemberRole(groupId, userId, roleName);
    }

    async hasAnyRole(userId: string, groupId: string, roleNames: string[]): Promise<boolean> {
        for (const roleName of roleNames) {
            if (await this.rbacRepo.hasMemberRole(groupId, userId, roleName)) {
                return true;
            }
        }
        return false;
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
        const existingRole = await this.rbacRepo.getGroupRoleByName(groupId, name);
        if (existingRole) {
            throw new Error('A role with this name already exists in this group');
        }
        
        return await this.rbacRepo.createGroupRole(groupId, name, description);
    }

    /**
     * Update an existing custom role
     */
    async updateGroupRole(userId: string, groupId: string, roleId: string, name: string, description: string): Promise<RoleRow> {
        const role = await this.rbacRepo.getGroupRoleById(roleId);
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
            const existingRole = await this.rbacRepo.getGroupRoleByName(groupId, name);
            if (existingRole) {
                throw new Error('A role with this name already exists in this group');
            }
        }
        
        return await this.rbacRepo.updateGroupRole(roleId, groupId, name, description);
    }

    /**
     * Update role priority level (only for custom roles)
     */
    async updateGroupRoleLevel(userId: string, groupId: string, roleId: string, roleLevel: number): Promise<RoleRow> {
        const role = await this.rbacRepo.getGroupRoleById(roleId);
        if (!role) {
            throw new Error('Role not found');
        }
        if (role.group_id && role.group_id !== groupId) {
            throw new Error('Role does not belong to this group');
        }
        
        // Validate role level range
        // System roles: 1-49 (reserved for high-priority roles)
        // Custom roles: 50-999
        if (roleLevel < 1 || roleLevel > 999) {
            throw new Error('Invalid role level. Must be between 1 and 999');
        }
        
        // Warn if trying to set system-level priority for custom role
        if (!role.is_system_role && roleLevel < 50) {
            throw new Error('Custom roles cannot have priority below 50');
        }
        
        return await this.rbacRepo.updateGroupRole(roleId, groupId, undefined, undefined, roleLevel);
    }

    /**
     * Delete a custom role
     */
    async deleteGroupRole(userId: string, groupId: string, roleId: string): Promise<boolean> {
        const role = await this.rbacRepo.getGroupRoleById(roleId);
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
     * Get all permissions for a specific role
     * @param roleId 
     * @returns 
     */
    async getRolePermissions(roleId: string): Promise<PermissionRow[]> {
        return await this.rbacRepo.listGroupRolePermissions(roleId);
    }

    /**
     * Get all permissions for a user (both direct and via roles)
     * @param userId 
     * @returns 
     */
    async getUserPermissions(userId: string): Promise<string[]> {
        return await this.rbacRepo.getUserPermissions(userId);
    }

    /**
     * Grant a permission to a role
     */
    async grantPermissionToRole(userId: string, roleId: string, permissionId: string): Promise<void> {
        // Prevent modifying Group Owner role permissions
        const role = await this.rbacRepo.getGroupRoleById(roleId);
        if (!role) {
            throw new Error('Role not found');
        }
        if (role.name === 'Group Owner') {
            throw new Error('Cannot modify Group Owner role permissions');
        }
        await this.rbacRepo.grantPermissionToGroupRole(roleId, permissionId);
    }

    /**
     * Revoke a permission from a role
     */
    async revokePermissionFromRole(userId: string, roleId: string, permissionId: string): Promise<void> {
        // Prevent modifying Group Owner role permissions
        const role = await this.rbacRepo.getGroupRoleById(roleId);
        if (!role) {
            throw new Error('Role not found');
        }
        if (role.name === 'Group Owner') {
            throw new Error('Cannot modify Group Owner role permissions');
        }
        await this.rbacRepo.revokePermissionFromGroupRole(roleId, permissionId);
    }

    /**
     * Assign a role to a group member
     */
    async assignRoleToMember(userId: string, groupId: string, memberId: string, roleId: string): Promise<void> {
        await this.checkPermission(userId, groupId, 'assign', 'roles');
        
        // Check if actor can manage the target role (hierarchy check)
        const canManage = await this.rbacRepo.canManageRoleInGroup(userId, groupId, roleId);
        if (!canManage) {
            throw new Error('Cannot assign role: Target role is at or above your privilege level');
        }
        
        await this.rbacRepo.assignRoleToMember(memberId, roleId, userId);
    }

    async assignRoleToUser(userId: string, roleId: string): Promise<void> {
        
    }

    /**
     * Remove a role from a group member
     */
    async removeRoleFromMember(userId: string, groupId: string, memberId: string, roleId: string): Promise<void> {
        await this.checkPermission(userId, groupId, 'remove', 'role_assignment');
        
        // Check if actor can manage the target role (hierarchy check)
        const canManage = await this.rbacRepo.canManageRoleInGroup(userId, groupId, roleId);
        if (!canManage) {
            throw new Error('Cannot remove role: Target role is at or above your privilege level');
        }
        
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

    /**
     * Grant a direct permission to a group member
     * Permission is constrained by member's role level
     */
    async grantDirectPermissionToGroupMember(
        actorUserId: string,
        groupId: string,
        memberId: string,
        permissionId: string
    ): Promise<void> {
        // Check if actor has permission to grant member permissions
        await this.checkPermission(actorUserId, groupId, 'grant', 'member_permissions');
        
        // Check hierarchy: actor must have higher privilege than target member
        const canGrant = await this.rbacRepo.canManagePermissionToMember(
            actorUserId,
            groupId,
            memberId,
        );
        
        if (!canGrant) {
            throw new Error('Cannot grant permission: Target member has equal or higher privilege level');
        }
        
        await this.rbacRepo.grantDirectPermissionToGroupMember(
            groupId,
            memberId,
            permissionId,
            actorUserId
        );
    }

    /**
     * Revoke a direct permission from a group member
     */
    async revokeDirectPermissionFromGroupMember(
        actorUserId: string,
        groupId: string,
        memberId: string,
        permissionId: string
    ): Promise<void> {
        // Check if actor has permission
        await this.checkPermission(actorUserId, groupId, 'revoke', 'member_permissions');
        
        // Check hierarchy
        const canRevoke = await this.rbacRepo.canManagePermissionToMember(
            actorUserId,
            groupId,
            memberId,
        );
        
        if (!canRevoke) {
            throw new Error('Cannot revoke permission: Target member has equal or higher privilege level');
        }
        
        await this.rbacRepo.revokeDirectPermissionFromGroupMember(groupId, memberId, permissionId);
    }

    /**
     * Get all direct permissions for a group member
     */
    async getDirectPermissionsForMember(groupId: string, memberId: string): Promise<any[]> {
        const permissions = await this.rbacRepo.getDirectPermissionsForMember(groupId, memberId);
        // Add computed name field for frontend
        return permissions.map(p => ({
            ...p,
            name: `${p.scope}:${p.action}:${p.resource}`
        }));
    }}