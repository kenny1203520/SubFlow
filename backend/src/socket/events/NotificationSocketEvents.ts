export class NotificationSocketEvents {
    readonly LIST = 'notification:list';
    readonly MARK_READ = 'notification:mark_read';
    readonly MARK_ALL_READ = 'notification:mark_all_read';

    all(): string[] {
        return [this.LIST, this.MARK_READ, this.MARK_ALL_READ];
    }
}

export const notificationSocketEvents = new NotificationSocketEvents();
