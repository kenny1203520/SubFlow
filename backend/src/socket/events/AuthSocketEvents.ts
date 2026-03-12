export class AuthSocketEvents {
    readonly USER = 'auth:user';
    readonly VERIFY = 'auth:verify';
    readonly LOGOUT = 'auth:logout';

    all(): string[] {
        return [this.USER, this.VERIFY, this.LOGOUT];
    }
}

export const authSocketEvents = new AuthSocketEvents();
