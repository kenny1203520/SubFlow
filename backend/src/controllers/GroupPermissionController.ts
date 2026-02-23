import { Socket } from 'socket.io';
import { BaseController } from './BaseController';
import { RBACService } from '../services/RBACService';

/**
 * GroupPermissionController
 * Handles direct permission assignments to individual users within groups
 * Following OOP/SOLID principles
 */
export class GroupPermissionController extends BaseController {
    private rbacService: RBACService;

    constructor(io: any, socket: Socket) {
        super(io, socket);
        this.rbacService = new RBACService();
    }

    /**
     * Register all socket event handlers for permission management
     */
    register(): void {
        this.socket.on('group:permission:grant', this.handleGrantPermission.bind(this));
        this.socket.on('group:permission:revoke', this.handleRevokePermission.bind(this));
        this.socket.on('group:permission:check', this.handleCheckPermission.bind(this));
    }

    /**
     * Helper to extract user ID from socket data
     */
    getUserId(): string {
        return this.socket.data.user.id;
    }

    /**
     * Grant a specific permission directly to a user
     */
    async handleGrantPermission(payload: any, callback: Function) {
        try {
            const actorUserId = this.getUserId();
            const { groupId, targetUserId, permissionId } = payload;

            if (!groupId || !targetUserId || !permissionId) {
                return callback({
                    status: 'error',
                    message: 'Group ID, target user ID, and permission ID are required'
                });
            }

            await this.rbacService.grantDirectPermission(actorUserId, groupId, targetUserId, permissionId);

            callback({
                status: 'ok',
                message: 'Permission granted successfully'
            });
        } catch (error: any) {
            console.error('[GroupPermissionController] Grant permission error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to grant permission'
            });
        }
    }

    /**
     * Revoke a specific permission from a user
     */
    async handleRevokePermission(payload: any, callback: Function) {
        try {
            const actorUserId = this.getUserId();
            const { groupId, targetUserId, permissionId } = payload;

            if (!groupId || !targetUserId || !permissionId) {
                return callback({
                    status: 'error',
                    message: 'Group ID, target user ID, and permission ID are required'
                });
            }

            await this.rbacService.revokeDirectPermission(actorUserId, groupId, targetUserId, permissionId);

            callback({
                status: 'ok',
                message: 'Permission revoked successfully'
            });
        } catch (error: any) {
            console.error('[GroupPermissionController] Revoke permission error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to revoke permission'
            });
        }
    }

    /**
     * Check if a user has a specific permission in a group
     */
    async handleCheckPermission(payload: any, callback: Function) {
        try {
            const userId = this.getUserId();
            const { groupId, action, resource } = payload;

            if (!groupId || !action || !resource) {
                return callback({
                    status: 'error',
                    message: 'Group ID, action, and resource are required'
                });
            }

            const hasPermission = await this.rbacService.hasPermission(userId, groupId, action, resource);

            callback({
                status: 'ok',
                hasPermission
            });
        } catch (error: any) {
            console.error('[GroupPermissionController] Check permission error:', error);
            callback({
                status: 'error',
                message: error.message || 'Failed to check permission'
            });
        }
    }
}
