export class ExpenseSocketEvents {
    readonly LIST = 'expense:list';
    readonly ADD = 'expense:add';
    readonly GET_SPLITS = 'expense:get_splits';
    readonly SETTLE = 'expense:settle';

    all(): string[] {
        return [this.LIST, this.ADD, this.GET_SPLITS, this.SETTLE];
    }
}

export const expenseSocketEvents = new ExpenseSocketEvents();
