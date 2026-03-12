export class BillSocketEvents {
    readonly LIST = 'bill:list';
    readonly GET = 'bill:get';
    readonly UPDATE_SPLIT = 'bill:update_split';

    all(): string[] {
        return [this.LIST, this.GET, this.UPDATE_SPLIT];
    }
}

export const billSocketEvents = new BillSocketEvents();
