import { Server, Socket } from 'socket.io';
import { SocketController } from './SocketController';
import { NotificationService } from '../services/NotificationService';

export class NotificationController extends SocketController {
    private notifService = new NotificationService();

    constructor(io: Server, socket: Socket) {
        super(io, socket);
    }

    register() {
        this.socket.on("notification:list", (payload, cb) => this.listNotifications(payload, cb));
        this.socket.on("notification:mark_read", (payload, cb) => this.markRead(payload, cb));
        this.socket.on("notification:mark_all_read", (...args: any[]) => this.markAllRead(this.resolveAck(...args) as any));
    }

    async listNotifications(payload: { page?: number }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const result = await this.notifService.getNotifications(userId, payload?.page || 1);
            this.success(cb, result);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to list notifications");
        }
    }

    async markRead(payload: { ids: string[] }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            await this.notifService.markRead(userId, payload.ids);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to mark notifications as read");
        }
    }

    async markAllRead(cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            await this.notifService.markAllRead(userId);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to mark all notifications as read");
        }
    }
}
