export class GroupPermissionSocketEvents {
    readonly GRANT = 'group:permission:grant';
    readonly REVOKE = 'group:permission:revoke';
    readonly CHECK = 'group:permission:check';

    all(): string[] {
        return [this.GRANT, this.REVOKE, this.CHECK];
    }
}

export const groupPermissionSocketEvents = new GroupPermissionSocketEvents();
