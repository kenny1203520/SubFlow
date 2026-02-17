import { NotificationRepository } from '../repositories/NotificationRepository';

export class NotificationService {
    private notifRepo = new NotificationRepository();

    async createNotification(userId: string, type: string, title: string, message: string, data?: any) {
        return await this.notifRepo.create({
            user_id: userId,
            type,
            title,
            message,
            data
        });
    }

    async getNotifications(userId: string, page: number = 1) {
        const limit = 20;
        const offset = (page - 1) * limit;
        const notifications = await this.notifRepo.findByUserId(userId, limit, offset);
        const unreadCount = await this.notifRepo.countUnread(userId);
        return { notifications, unreadCount };
    }

    async markRead(userId: string, notificationIds: string[]) {
        await this.notifRepo.markAsRead(notificationIds, userId);
    }

    async markAllRead(userId: string) {
        await this.notifRepo.markAllAsRead(userId);
    }
}
