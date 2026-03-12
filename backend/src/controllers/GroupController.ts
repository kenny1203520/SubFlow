import { Socket, Server } from 'socket.io';
import { SocketController } from './SocketController';
import { GroupService } from '../services/GroupService';
import { groupSocketEvents } from '../socket/events';

export class GroupController extends SocketController {
    private groupService: GroupService;
    
    constructor(io: Server, socket: Socket) {
        super(io, socket);
        this.groupService = new GroupService();
    }

    /**
     * Register all socket event handlers for role management
     */
    register(): void {
        if (!this.socket) return;
        this.socket.on(groupSocketEvents.CREATE, (payload, cb) => this.createGroup(payload, cb));
        this.socket.on(groupSocketEvents.LIST, (...args: any[]) => this.listGroups(this.resolveAck(...args) as any));
        this.socket.on(groupSocketEvents.GET, (payload, cb) => this.getGroup(payload, cb));
        this.socket.on(groupSocketEvents.GET_OVERVIEW, (payload, cb) => this.getOverview(payload, cb));
        this.socket.on(groupSocketEvents.ADD_MEMBER, (payload, cb) => this.addMember(payload, cb));
        this.socket.on(groupSocketEvents.BIND_MEMBER, (payload, cb) => this.bindMember(payload, cb));
        this.socket.on(groupSocketEvents.DELETE, (payload, cb) => this.deleteGroup(payload, cb));
        this.socket.on(groupSocketEvents.LEAVE, (payload, cb) => this.leaveGroup(payload, cb));
        this.socket.on(groupSocketEvents.ACCEPT_INVITE, (payload, cb) => this.acceptInvite(payload, cb));
        this.socket.on(groupSocketEvents.REJECT_INVITE, (payload, cb) => this.rejectInvite(payload, cb));
        this.socket.on(groupSocketEvents.BIND_MEMBER_INVITE, (payload, cb) => this.bindMemberInvite(payload, cb));
        this.socket.on(groupSocketEvents.UPDATE, (payload, cb) => this.updateGroup(payload, cb));
        this.socket.on(groupSocketEvents.UPDATE_MEMBER_ROLE, (payload, cb) => this.updateMemberRole(payload, cb));
        this.socket.on(groupSocketEvents.ASSIGN_DYNAMIC_ROLE, (payload, cb) => this.assignDynamicRole(payload, cb));
        this.socket.on(groupSocketEvents.REMOVE_DYNAMIC_ROLE, (payload, cb) => this.removeDynamicRole(payload, cb));
        this.socket.on(groupSocketEvents.REMOVE_MEMBER, (payload, cb) => this.removeMember(payload, cb));
        this.socket.on(groupSocketEvents.CANCEL_INVITE, (payload, cb) => this.cancelInvite(payload, cb));
        this.socket.on(groupSocketEvents.LIST_ROLES, (payload, cb) => this.listRoles(payload, cb));
        this.socket.on(groupSocketEvents.CREATE_ROLE, (payload, cb) => this.createRole(payload, cb));
    }

    async createGroup(payload: any, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            const group = await this.groupService.createGroup(userId, payload);
            this.success(cb, { group });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to create group");
        }
    }

    async listGroups(cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            const groups = await this.groupService.listGroups(userId);
            this.success(cb, { groups });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to list groups");
        }
    }

    async listRoles(payload: any, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            if (!payload?.groupId) {
                return this.error(cb, "Group ID is required");
            }
            const roles = await this.groupService.listRoles(payload.groupId);
            this.success(cb, roles);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to list group roles");
        }
    }

    async createRole(payload: { name: string, description: string }, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            if (!payload || !(payload as any).groupId) {
                return this.error(cb, "Group ID is required");
            }
            const role = await this.groupService.createRole(userId, (payload as any).groupId, payload);
            this.success(cb, role);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to create group role");
        }
    }

    async getGroup(payload: { groupId: string }, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            const detail = await this.groupService.getGroupDetail(userId, payload.groupId);
            this.success(cb, detail);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to get group details");
        }
    }

    async getOverview(payload: { groupId: string }, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            const overview = await this.groupService.getGroupOverview(userId, payload.groupId);
            this.success(cb, overview);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to get group overview");
        }
    }

    async addMember(payload: any, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            await this.groupService.addMember(userId, payload.groupId, payload);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to add member");
        }
    }

    async bindMember(payload: { memberId: string }, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            await this.groupService.bindMember(userId, payload.memberId);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to bind member");
        }
    }

    async deleteGroup(payload: { groupId: string }, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            await this.groupService.deleteGroup(userId, payload.groupId);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to delete group");
        }
    }

    async leaveGroup(payload: { groupId: string }, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            await this.groupService.leaveGroup(userId, payload.groupId);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to leave group");
        }
    }

    async acceptInvite(payload: { groupId: string, memberId?: string }, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            await this.groupService.acceptInvite(userId, payload.groupId, payload.memberId);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to accept invite");
        }
    }

    async rejectInvite(payload: { groupId: string }, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            await this.groupService.rejectInvite(userId, payload.groupId);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to reject invite");
        }
    }

    async bindMemberInvite(payload: { groupId: string, memberId: string, email?: string, username?: string }, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            await this.groupService.bindMemberInvite(userId, payload.groupId, payload.memberId, { email: payload.email, username: payload.username });
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to send binding invite");
        }
    }

    async updateGroup(payload: any, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            const group = await this.groupService.updateGroup(userId, payload.id, payload);
            this.success(cb, { group });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to update group");
        }
    }

    async updateMemberRole(payload: { groupId: string, memberId: string, role: 'admin' | 'member' }, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            await this.groupService.updateMemberRole(userId, payload.groupId, payload.memberId, payload.role);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to update role");
        }
    }

    async assignDynamicRole(payload: { groupId: string, memberId: string, roleId: string }, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            await this.groupService.assignDynamicRole(userId, payload.groupId, payload.memberId, payload.roleId);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to assign dynamic role");
        }
    }

    async removeDynamicRole(payload: { groupId: string, memberId: string, roleId: string }, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            await this.groupService.removeDynamicRole(userId, payload.groupId, payload.memberId, payload.roleId);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to remove dynamic role");
        }
    }

    async removeMember(payload: { groupId: string, memberId: string }, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            await this.groupService.removeMember(userId, payload.groupId, payload.memberId);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to remove member");
        }
    }

    async cancelInvite(payload: { groupId: string, memberId: string }, cb: (res: any) => void) {
        if (!this.socket) return;
        try {
            const userId = this.socket.data.user.id;
            await this.groupService.cancelInvite(userId, payload.groupId, payload.memberId);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to cancel invite");
        }
    }

}
