import { Server, Socket } from 'socket.io';
import { SocketController } from './SocketController';
import { SubscriptionService } from '../services/SubscriptionService';

export class SubscriptionController extends SocketController {
    private subService = new SubscriptionService();

    constructor(io: Server, socket: Socket) {
        super(io, socket);
    }

    register() {
        this.socket.on("subscription:add", (payload, cb) => this.addSubscription(payload, cb));
        this.socket.on("subscription:list", (payload, cb) => this.listSubscriptions(payload, cb));
        this.socket.on("subscription:all", (...args: any[]) => this.listAllSubscriptions(this.resolveAck(...args) as any));
        this.socket.on("subscription:update_status", (payload, cb) => this.updateStatus(payload, cb));
    }

    async addSubscription(payload: any, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const subscription = await this.subService.addSubscription(userId, payload);
            this.success(cb, { subscription });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to add subscription");
        }
    }

    async listSubscriptions(payload: { groupId: string }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const subscriptions = await this.subService.listSubscriptions(userId, payload.groupId);
            this.success(cb, { subscriptions });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to list subscriptions");
        }
    }

    async listAllSubscriptions(cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const subscriptions = await this.subService.listAllSubscriptions(userId);
            this.success(cb, { subscriptions });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to list all subscriptions");
        }
    }

    async updateStatus(payload: { subscriptionId: string, status: string }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            await this.subService.updateStatus(userId, payload.subscriptionId, payload.status);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to update status");
        }
    }
}
