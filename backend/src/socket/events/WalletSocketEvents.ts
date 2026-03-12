export class WalletSocketEvents {
    readonly LIST = 'wallet:list';
    readonly DETAILS = 'wallet:details';
    readonly DEPOSIT = 'wallet:deposit';
    readonly TRANSFER = 'wallet:transfer';

    all(): string[] {
        return [this.LIST, this.DETAILS, this.DEPOSIT, this.TRANSFER];
    }
}

export const walletSocketEvents = new WalletSocketEvents();
