export class SystemSocketEvents {
    readonly PING = 'ping';
    readonly DASHBOARD_STATS = 'dashboard:stats';

    all(): string[] {
        return [this.PING, this.DASHBOARD_STATS];
    }
}

export const systemSocketEvents = new SystemSocketEvents();
