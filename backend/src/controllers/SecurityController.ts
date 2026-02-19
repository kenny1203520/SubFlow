import { BaseController } from './BaseController';
import { SecurityService } from '../services/SecurityService';

export class SecurityController extends BaseController {
    private securityService = new SecurityService();

    register() {
        this.socket.on("security:settings", (cb) => this.getSettings(cb));
        this.socket.on("security:generate_2fa_secret", (cb) => this.generate2FASecret(cb));
        this.socket.on("security:enable_2fa", (payload, cb) => this.enable2FA(payload, cb));
        this.socket.on("security:disable_2fa", (cb) => this.disable2FA(cb));
        this.socket.on("security:devices", (cb) => this.listDevices(cb));
        this.socket.on("security:revoke_device", (payload, cb) => this.revokeDevice(payload, cb));
    }

    async getSettings(cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const settings = await this.securityService.getSettings(userId);
            this.success(cb, { settings });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to fetch settings");
        }
    }

    async generate2FASecret(cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const secret = await this.securityService.generate2FASecret(userId);
            this.success(cb, { secret });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to generate secret");
        }
    }

    async enable2FA(payload: { secret: string, code: string }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            await this.securityService.enable2FA(userId, payload.secret, payload.code);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to enable 2FA");
        }
    }

    async disable2FA(cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            await this.securityService.disable2FA(userId);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to disable 2FA");
        }
    }

    async listDevices(cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            const devices = await this.securityService.listDevices(userId);
            this.success(cb, { devices });
        } catch (error: any) {
            this.error(cb, error.message || "Failed to list devices");
        }
    }

    async revokeDevice(payload: { deviceId: string }, cb: (res: any) => void) {
        try {
            const userId = this.socket.data.user.id;
            await this.securityService.logoutDevice(userId, payload.deviceId);
            this.success(cb);
        } catch (error: any) {
            this.error(cb, error.message || "Failed to revoke device");
        }
    }
}
