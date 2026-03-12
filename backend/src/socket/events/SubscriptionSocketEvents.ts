export class SubscriptionSocketEvents {
    readonly ADD = 'subscription:add';
    readonly LIST = 'subscription:list';
    readonly ALL = 'subscription:all';
    readonly UPDATE_STATUS = 'subscription:update_status';

    all(): string[] {
        return [this.ADD, this.LIST, this.ALL, this.UPDATE_STATUS];
    }
}

export const subscriptionSocketEvents = new SubscriptionSocketEvents();
