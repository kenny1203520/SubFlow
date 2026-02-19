import { SecurityRepository } from '../repositories/SecurityRepository';
import { TOTP } from 'otplib';
import { lucia } from '../auth/lucia';

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

    async getSessions(userId: string) {
        return await lucia.getUserSessions(userId);
    }

    async revokeSession(userId: string, sessionId: string) {
        const session = await lucia.validateSession(sessionId);
        if (session.session && session.session.userId === userId) {
            await lucia.invalidateSession(sessionId);
        }
    }

    async logoutDevice(userId: string, deviceId: string) {
        await this.securityRepo.revokeDevice(userId, deviceId);
    }
}
