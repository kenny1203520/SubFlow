export class FileSocketEvents {
    readonly GET = 'file:get';
    readonly DELETE = 'file:delete';

    all(): string[] {
        return [this.GET, this.DELETE];
    }
}

export const fileSocketEvents = new FileSocketEvents();
