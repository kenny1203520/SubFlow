export class ServiceSocketEvents {
    readonly SEARCH = 'service:search';
    readonly CREATE = 'service:create';
    readonly LIST = 'service:list';
    readonly UPDATE = 'service:update';
    readonly DELETE = 'service:delete';

    all(): string[] {
        return [this.SEARCH, this.CREATE, this.LIST, this.UPDATE, this.DELETE];
    }
}

export const serviceSocketEvents = new ServiceSocketEvents();
