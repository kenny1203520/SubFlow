import { SecurityRepository } from '../repositories/SecurityRepository';
import { generateSecret, verify, generateURI } from 'otplib';
import QRCode from 'qrcode';
import { lucia } from '../auth/lucia';
import { pool } from '../db';

export class SecurityService {
    private securityRepo = new SecurityRepository();

    async getSettings(userId: string) {
        return await this.securityRepo.getSecuritySettings(userId);
    }

    async generate2FASecret(userId: string) {
        const secret = generateSecret();
        
        // Fetch user email for the URI label
        const userRes = await pool.query('SELECT email FROM users WHERE id = $1', [userId]);
        const userEmail = userRes.rows[0]?.email || 'User';

        const uri = generateURI({
            issuer: 'SubFlow',
            label: userEmail,
            secret,
        });
        
        const qrDataUrl = await QRCode.toDataURL(uri);

        return { secret, qrDataUrl };
    }

    async enable2FA(userId: string, secret: string, code: string) {
        const result = verify({ token: code, secret });
        if (!result) {
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
