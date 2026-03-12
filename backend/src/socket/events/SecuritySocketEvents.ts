export class SecuritySocketEvents {
    readonly GET_SETTINGS = 'security:get_settings';
    readonly GENERATE_2FA_SECRET = 'security:generate_2fa_secret';
    readonly ENABLE_2FA = 'security:enable_2fa';
    readonly DISABLE_2FA = 'security:disable_2fa';
    readonly REGENERATE_BACKUP_CODES = 'security:regenerate_backup_codes';
    readonly LIST_DEVICES = 'security:list_devices';
    readonly REVOKE_DEVICE = 'security:revoke_device';
    readonly SESSIONS = 'security:sessions';
    readonly REVOKE_SESSION = 'security:revoke_session';

    all(): string[] {
        return [
            this.GET_SETTINGS,
            this.GENERATE_2FA_SECRET,
            this.ENABLE_2FA,
            this.DISABLE_2FA,
            this.REGENERATE_BACKUP_CODES,
            this.LIST_DEVICES,
            this.REVOKE_DEVICE,
            this.SESSIONS,
            this.REVOKE_SESSION
        ];
    }
}

export const securitySocketEvents = new SecuritySocketEvents();
