import { SecurityRepository } from '../repositories/SecurityRepository';
import { TOTP } from 'otplib';

export class SecurityService {
    private securityRepo = new SecurityRepository();
    private totp = new TOTP();

    async getSettings(userId: string) {
        return await this.securityRepo.getSecuritySettings(userId);
    }

    async generate2FASecret(userId: string) {
        return this.totp.generateSecret();
    }

    async enable2FA(userId: string, secret: string, code: string) {
        const isValid = this.totp.verify(code, { secret });
        if (!isValid) {
            throw new Error("Invalid verification code");
        }
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
