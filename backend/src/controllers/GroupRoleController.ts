import { Socket, Server } from 'socket.io';
import { SocketController } from './SocketController';
import { RBACService } from '../services/RBACService';
import { GroupService } from '../services/GroupService';
import { groupRoleSocketEvents } from '../socket/events';

/**
 * GroupRoleController
 * Handles all role management operations within groups
 * Following OOP/SOLID principles with clear separation of concerns
 */
export class GroupRoleController extends SocketController {
    private rbacService: RBACService;
    private groupService: GroupService;

    constructor(io: Server, socket: Socket) {
        super(io, socket);
        this.rbacService = new RBACService();
        this.groupService = new GroupService();
    }

    /**
     * Register all socket event handlers for role management
     */
    register(): void {
        if (!this.socket) return;
        this.socket.on(groupRoleSocketEvents.LIST, this.handleListRoles.bind(this));
        this.socket.on(groupRoleSocketEvents.QUANTITY_LIMIT, this.handleQuantityLimit.bind(this));
        this.socket.on(groupRoleSocketEvents.USER_MAX_ROLE_LEVEL, this.handleGetUserMaxRoleLevel.bind(this));
        this.socket.on(groupRoleSocketEvents.GET_PERMISSIONS, this.handleGetRolePermissions.bind(this));
        this.socket.on(groupRoleSocketEvents.CREATE, this.handleCreateRole.bind(this));
        this.socket.on(groupRoleSocketEvents.UPDATE, this.handleUpdateRole.bind(this));
        this.socket.on(groupRoleSocketEvents.UPDATE_LEVEL, this.handleUpdateRoleLevel.bind(this));
        this.socket.on(groupRoleSocketEvents.DELETE, this.handleDeleteRole.bind(this));
        this.socket.on(groupRoleSocketEvents.ASSIGN, this.handleAssignRole.bind(this));
        this.socket.on(groupRoleSocketEvents.REMOVE, this.handleRemoveRole.bind(this));
        this.socket.on(groupRoleSocketEvents.GRANT_PERMISSION, this.handleGrantPermissionToRole.bind(this));
        this.socket.on(groupRoleSocketEvents.REVOKE_PERMISSION, this.handleRevokePermissionFromRole.bind(this));
        this.socket.on(groupRoleSocketEvents.MEMBER_GRANT_PERMISSION, this.handleGrantDirectPermissionToMember.bind(this));
        this.socket.on(groupRoleSocketEvents.MEMBER_REVOKE_PERMISSION, this.handleRevokeDirectPermissionFromMember.bind(this));
        this.socket.on(groupRoleSocketEvents.MEMBER_LIST_DIRECT_PERMISSIONS, this.handleListDirectPermissionsForMember.bind(this));
        this.socket.on(groupRoleSocketEvents.LIST_ALL_PERMISSIONS, this.handleListAllPermissions.bind(this));
        this.socket.on(groupRoleSocketEvents.TRANSFER_OWNERSHIP, this.handleTransferOwnership.bind(this));
    }

    /**
     * Helper to extract user ID from socket data
     */
    getUserId(): string {
        if (!this.socket || !this.socket.data || !this.socket.data.user) {
            throw new Error('User not authenticated');
        }
        return this.socket.data.user.id;
    }

    /**
     * List all available group roles
     */
    async handleListRoles(payload: any, callback: Function) {
        try {
            const userId = this.getUserId();
            const { groupId } = payload;

            if (!groupId) {
                return callback({ status: 'error', message: 'Group ID is required' });
            }

            const hasPremission = await this.rbacService.hasPermission(userId, groupId, 'read', 'roles');
            if (!hasPremission) {
                return callback({ status: 'error', message: 'Permission denied' });
            }
            
            const roles = await this.rbacService.listGroupRoles(groupId);
            
            callback({
                status: 'ok',
                roles
            });
        } catch (error: any) {
            console.error('[GroupRoleController] List roles error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to list roles'
            });
        }
    }
    
    /**
     * List Group Roles with quantity limit check (for UI that needs to know if they can create more roles or not)
     */
    async handleQuantityLimit(payload: any, callback: Function) {
        try {
            const userId = this.getUserId();
            const { groupId } = payload;
            if (!groupId) {
                return callback({ status: 'error', message: 'Group ID is required' });
            }
            const hasPremission = await this.rbacService.hasPermission(userId, groupId, 'read', 'roles');
            if (!hasPremission) {
                return callback({ status: 'error', message: 'Permission denied' });
            }
            
            const roles = await this.rbacService.listGroupRoles(groupId);
            const systemRolesCount = roles.filter(r => r.is_system_role).length;
            const customRolesCount = roles.length - systemRolesCount;
            const limits = await this.rbacService.getSystemSettingsGroups();
            const canCreateMore = customRolesCount < limits.roleLimit;

            callback({
                status: 'ok',
                canCreateMore,
                customRolesCount,
                systemRolesCount,
                roleLimit: limits.roleLimit
            });
        } catch (error: any) {
            console.error('[GroupRoleController] List available roles error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to list available roles'
            });
        }
    }

    /**
     * Get permissions for a specific role
     */
    async handleGetRolePermissions(payload: any, callback: Function) {
        try {
            const userId = this.getUserId();
            const { roleId } = payload;

            if (!roleId) {
                return callback({ status: 'error', message: 'Role ID is required' });
            }

            const permissions = await this.rbacService.getRolePermissions(roleId);

            callback({
                status: 'ok',
                permissions
            });
        } catch (error: any) {
            console.error('[GroupRoleController] Get role permissions error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to get role permissions'
            });
        }
    }

    /**
     * Create a new custom group role
     */
    async handleCreateRole(payload: any, callback: Function) {
        try {
            const userId = this.getUserId();
            const { groupId, name, description } = payload;

            if (!groupId || !name) {
                return callback({ status: 'error', message: 'Group ID and role name are required' });
            }

            // Check if user has permission to create roles in this group
            const hasPremission = await this.rbacService.hasPermission(userId, groupId, 'create', 'roles');
            if (!hasPremission) {
                return callback({ status: 'error', message: 'Permission denied' });
            }

            // Check if role amount limit is reached
            const existingRoles = await this.rbacService.listGroupRoles(groupId);
            const systemRolesCount = existingRoles.filter(r => r.is_system_role).length;
            const customRolesCount = existingRoles.length - systemRolesCount;
            const limits = await this.rbacService.getSystemSettingsGroups();
            if (customRolesCount >= limits.roleLimit) {
                return callback({ status: 'error', message: 'Role limit reached for this group' });
            }

            const role = await this.rbacService.createGroupRole(userId, groupId, name, description);

            callback({
                status: 'ok',
                role,
                message: 'Role created successfully'
            });
        } catch (error: any) {
            console.error('[GroupRoleController] Create role error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to create role'
            });
        }
    }

    /**
     * Update an existing custom role
     */
    async handleUpdateRole(payload: any, callback: Function) {
        try {
            const userId = this.getUserId();
            const { groupId, roleId, name, description } = payload;

            if (!groupId || !roleId) {
                return callback({ status: 'error', message: 'Group ID and Role ID are required' });
            }

            // Check permission
            const hasPremission = await this.rbacService.hasPermission(userId, groupId, 'update', 'roles');
            if (!hasPremission) {
                return callback({ status: 'error', message: 'Permission denied' });
            }

            const role = await this.rbacService.updateGroupRole(userId, groupId, roleId, name, description);

            callback({
                status: 'ok',
                role,
                message: 'Role updated successfully'
            });
        } catch (error: any) {
            console.error('[GroupRoleController] Update role error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to update role'
            });
        }
    }

    /**
     * Update role priority level
     */
    async handleUpdateRoleLevel(payload: any, callback: Function) {
        try {
            const userId = this.getUserId();
            const { groupId, roleId, roleLevel } = payload;

            if (!groupId || !roleId || roleLevel === undefined) {
                return callback({ status: 'error', message: 'Group ID, Role ID, and role level are required' });
            }

            // Check permission
            const hasPremission = await this.rbacService.hasPermission(userId, groupId, 'update', 'roles');
            if (!hasPremission) {
                return callback({ status: 'error', message: 'Permission denied' });
            }

            const role = await this.rbacService.updateGroupRoleLevel(userId, groupId, roleId, roleLevel);

            callback({
                status: 'ok',
                role,
                message: 'Role priority updated successfully'
            });
        } catch (error: any) {
            console.error('[GroupRoleController] Update role level error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to update role priority'
            });
        }
    }

    /**
     * Delete a custom role
     */
    async handleDeleteRole(payload: any, callback: Function) {
        try {
            const userId = this.getUserId();
            const { groupId, roleId } = payload;

            if (!groupId || !roleId) {
                return callback({ status: 'error', message: 'Group ID and Role ID are required' });
            }

            // Check permission
            const hasPremission = await this.rbacService.hasPermission(userId, groupId, 'delete', 'roles');
            if (!hasPremission) {
                return callback({ status: 'error', message: 'Permission denied' });
            }

            const success = await this.rbacService.deleteGroupRole(userId, groupId, roleId);

            if (!success) {
                return callback({ status: 'error', message: 'Failed to delete role' });
            }

            callback({
                status: 'ok',
                message: 'Role deleted successfully'
            });
        } catch (error: any) {
            console.error('[GroupRoleController] Delete role error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to delete role'
            });
        }
    }

    /**
     * Assign a role to a group member
     */
    async handleAssignRole(payload: any, callback: Function) {
        try {
            const userId = this.getUserId();
            const { groupId, memberId, roleId } = payload;

            if (!groupId || !memberId || !roleId) {
                return callback({ status: 'error', message: 'Group ID, Member ID, and Role ID are required' });
            }

            await this.rbacService.assignRoleToMember(userId, groupId, memberId, roleId);

            callback({
                status: 'ok',
                message: 'Role assigned successfully'
            });
        } catch (error: any) {
            console.error('[GroupRoleController] Assign role error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to assign role'
            });
        }
    }

    /**
     * Remove a role from a group member
     */
    async handleRemoveRole(payload: any, callback: Function) {
        try {
            const userId = this.getUserId();
            const { groupId, memberId, roleId } = payload;

            if (!groupId || !memberId || !roleId) {
                return callback({ status: 'error', message: 'Group ID, Member ID, and Role ID are required' });
            }

            await this.rbacService.removeRoleFromMember(userId, groupId, memberId, roleId);

            callback({
                status: 'ok',
                message: 'Role removed successfully'
            });
        } catch (error: any) {
            console.error('[GroupRoleController] Remove role error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to remove role'
            });
        }
    }

    /**
     * Grant a specific permission to a role
     */
    async handleGrantPermissionToRole(payload: any, callback: Function) {
        try {
            const userId = this.getUserId();
            const { groupId, roleId, permissionId } = payload;

            if (!groupId || !roleId || !permissionId) {
                return callback({ status: 'error', message: 'Group ID, Role ID, and Permission ID are required' });
            }

            // Check if user can manage role permissions
            const hasPremission = await this.rbacService.hasPermission(userId, groupId, 'update', 'roles');
            if (!hasPremission) {
                return callback({ status: 'error', message: 'Permission denied' });
            }

            await this.rbacService.grantPermissionToRole(userId, roleId, permissionId);

            callback({
                status: 'ok',
                message: 'Permission granted to role'
            });
        } catch (error: any) {
            console.error('[GroupRoleController] Grant permission error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to grant permission'
            });
        }
    }

    /**
     * Revoke a permission from a role
     */
    async handleRevokePermissionFromRole(payload: any, callback: Function) {
        try {
            const userId = this.getUserId();
            const { groupId, roleId, permissionId } = payload;

            if (!groupId || !roleId || !permissionId) {
                return callback({ status: 'error', message: 'Group ID, Role ID, and Permission ID are required' });
            }

            // Check if user can manage role permissions
            const hasPremission = await this.rbacService.hasPermission(userId, groupId, 'update', 'roles');
            if (!hasPremission) {
                return callback({ status: 'error', message: 'Permission denied' });
            }

            await this.rbacService.revokePermissionFromRole(userId, roleId, permissionId);

            callback({
                status: 'ok',
                message: 'Permission revoked from role'
            });
        } catch (error: any) {
            console.error('[GroupRoleController] Revoke permission error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to revoke permission'
            });
        }
    }

    /**
     * List all available permissions (for UI pickers)
     */
    async handleListAllPermissions(payload: any, callback: Function) {
        try {
            const { scope } = payload;

            const permissions = await this.rbacService.listAllPermissions(scope || 'group');

            callback({
                status: 'ok',
                permissions
            });
        } catch (error: any) {
            console.error('[GroupRoleController] List permissions error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to list permissions'
            });
        }
    }

    /**
     * Transfer group ownership
     */
    async handleTransferOwnership(payload: any, callback: Function) {
        try {
            const userId = this.getUserId();
            const { groupId, newOwnerId } = payload;

            if (!groupId || !newOwnerId) {
                return callback({ status: 'error', message: 'Group ID and new owner ID are required' });
            }

            if (userId === newOwnerId) {
                return callback({ status: 'error', message: 'You are already the owner' });
            }

            await this.groupService.transferOwnership(userId, groupId, newOwnerId);

            callback({
                status: 'ok',
                message: 'Ownership transferred successfully'
            });
        } catch (error: any) {
            console.error('[GroupRoleController] Transfer ownership error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to transfer ownership'
            });
        }
    }

    /**
     * Get user's highest role level in a group
     */
    async handleGetUserMaxRoleLevel(payload: any, callback: Function) {
        try {
            const userId = this.getUserId();
            const { groupId } = payload;

            if (!groupId) {
                return callback({ status: 'error', message: 'Group ID is required' });
            }

            const maxLevel = await this.groupService.getUserMaxRoleLevel(userId, groupId);

            callback({
                status: 'ok',
                maxLevel
            });
        } catch (error: any) {
            console.error('[GroupRoleController] Get user max role level error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to get user max role level'
            });
        }
    }

    /**
     * Grant a direct permission to a group member
     */
    async handleGrantDirectPermissionToMember(payload: any, callback: Function) {
        try {
            const userId = this.getUserId();
            const { groupId, memberId, permissionId } = payload;

            if (!groupId || !memberId || !permissionId) {
                return callback({ status: 'error', message: 'Group ID, member ID, and permission ID are required' });
            }

            await this.rbacService.grantDirectPermissionToGroupMember(
                userId,
                groupId,
                memberId,
                permissionId
            );

            callback({
                status: 'ok',
                message: 'Permission granted successfully'
            });
        } catch (error: any) {
            console.error('[GroupRoleController] Grant direct permission error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to grant permission'
            });
        }
    }

    /**
     * Revoke a direct permission from a group member
     */
    async handleRevokeDirectPermissionFromMember(payload: any, callback: Function) {
        try {
            const userId = this.getUserId();
            const { groupId, memberId, permissionId } = payload;

            if (!groupId || !memberId || !permissionId) {
                return callback({ status: 'error', message: 'Group ID, member ID, and permission ID are required' });
            }

            await this.rbacService.revokeDirectPermissionFromGroupMember(
                userId,
                groupId,
                memberId,
                permissionId
            );

            callback({
                status: 'ok',
                message: 'Permission revoked successfully'
            });
        } catch (error: any) {
            console.error('[GroupRoleController] Revoke direct permission error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to revoke permission'
            });
        }
    }

    /**
     * List all direct permissions for a group member
     */
    async handleListDirectPermissionsForMember(payload: any, callback: Function) {
        try {
            const userId = this.getUserId();
            const { groupId, memberId } = payload;

            if (!groupId || !memberId) {
                return callback({ status: 'error', message: 'Group ID and member ID are required' });
            }

            // Check if requester has permission to view member permissions
            const hasPermission = await this.rbacService.hasPermission(
                userId,
                groupId,
                'read',
                'member_permissions'
            );

            if (!hasPermission && userId !== memberId) {
                return callback({ status: 'error', message: 'Permission denied' });
            }

            const directPermissions = await this.rbacService.getDirectPermissionsForMember(
                groupId,
                memberId
            );

            callback({
                status: 'ok',
                permissions: directPermissions
            });
        } catch (error: any) {
            console.error('[GroupRoleController] List direct permissions error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to list direct permissions'
            });
        }
    }
}

