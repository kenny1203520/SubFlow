import { BaseController } from './BaseController';
import { GroupService } from '../services/GroupService';

export class GroupController extends BaseController {
    private groupService = new GroupService();

    register() {
        this.socket.on("group:create", (payload, cb) => this.createGroup(payload, cb));
        this.socket.on("group:list", (cb) => this.listGroups(cb));
        this.socket.on("group:get", (payload, cb) => this.getGroup(payload, cb));
        this.socket.on("group:add_member", (payload, cb) => this.addMember(payload, cb));
        this.socket.on("group:bind_member", (payload, cb) => this.bindMember(payload, cb));
        this.socket.on("group:delete", (payload, cb) => this.deleteGroup(payload, cb));
        this.socket.on("group:leave", (payload, cb) => this.leaveGroup(payload, cb));
        this.socket.on("service:search", (payload, cb) => this.searchServices(payload, cb));
    }

    async createGroup(payload: any, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const group = await this.groupService.createGroup(userId, payload);
            this.success(cb, { group });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to create group");
        }
    }

    async listGroups(cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const groups = await this.groupService.listGroups(userId);
            this.success(cb, { groups });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to list groups");
        }
    }

    async getGroup(payload: { groupId: string }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const detail = await this.groupService.getGroupDetail(userId, payload.groupId);
            this.success(cb, detail);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to get group details");
        }
    }

    async addMember(payload: any, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            await this.groupService.addMember(userId, payload.groupId, payload);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to add member");
        }
    }

    async bindMember(payload: { memberId: string }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            await this.groupService.bindMember(userId, payload.memberId);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to bind member");
        }
    }

    async deleteGroup(payload: { groupId: string }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            await this.groupService.deleteGroup(userId, payload.groupId);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to delete group");
        }
    }

    async leaveGroup(payload: { groupId: string }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            await this.groupService.leaveGroup(userId, payload.groupId);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to leave group");
        }
    }

    async searchServices(payload: { query: string }, cb: (res: any) => void) {
        try {
            const services = await this.groupService.searchServices(payload.query);
            this.success(cb, { services });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to search services");
        }
    }
}
