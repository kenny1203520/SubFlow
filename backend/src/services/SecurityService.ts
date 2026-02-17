import { SecurityRepository } from '../repositories/SecurityRepository';

export class SecurityService {
    private securityRepo = new SecurityRepository();

    async getSettings(userId: string) {
        return await this.securityRepo.getSecuritySettings(userId);
    }

    // Simplified TOTP for now
    async enable2FA(userId: string, secret: string) {
        await this.securityRepo.update2FA(userId, secret, true);
    }

    async disable2FA(userId: string) {
        await this.securityRepo.update2FA(userId, '', false);
    }

    async listDevices(userId: string) {
        return await this.securityRepo.getDevices(userId);
    }

    async logoutDevice(userId: string, deviceId: string) {
        await this.securityRepo.revokeDevice(userId, deviceId);
    }
}
