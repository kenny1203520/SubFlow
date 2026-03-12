import {
    authSocketEvents,
    billSocketEvents,
    expenseSocketEvents,
    fileSocketEvents,
    groupPermissionSocketEvents,
    groupRoleSocketEvents,
    groupSocketEvents,
    notificationSocketEvents,
    securitySocketEvents,
    serviceSocketEvents,
    subscriptionSocketEvents,
    systemSocketEvents,
    walletSocketEvents
} from '../events';

interface SocketEventCatalog {
    all(): string[];
}

export class SocketEventRegistry {
    private readonly catalogs: SocketEventCatalog[];
    private cachedEvents: Set<string> | null = null;

    constructor(catalogs: SocketEventCatalog[]) {
        this.catalogs = catalogs;
    }

    getAll(): string[] {
        return Array.from(this.getSet()).sort();
    }

    has(eventName: string): boolean {
        return this.getSet().has(eventName);
    }

    private getSet(): Set<string> {
        if (this.cachedEvents) {
            return this.cachedEvents;
        }

        const allEvents = this.catalogs.flatMap((catalog) => catalog.all());
        this.cachedEvents = new Set(allEvents);
        return this.cachedEvents;
    }
}

export const socketEventRegistry = new SocketEventRegistry([
    authSocketEvents,
    billSocketEvents,
    expenseSocketEvents,
    fileSocketEvents,
    groupSocketEvents,
    groupRoleSocketEvents,
    groupPermissionSocketEvents,
    notificationSocketEvents,
    securitySocketEvents,
    serviceSocketEvents,
    subscriptionSocketEvents,
    walletSocketEvents,
    systemSocketEvents
]);
