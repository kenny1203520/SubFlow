import { SecurityRepository } from '../repositories/SecurityRepository';
import { generateSecret, verify, generateURI } from 'otplib';
import QRCode from 'qrcode';
import { lucia } from '../auth/lucia';
import { pool } from '../db';
import * as crypto from 'crypto';
import { Scrypt } from 'lucia';

const scrypt = new Scrypt();

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

    private generateBackupCodes(count: number = 10): string[] {
        const codes: string[] = [];
        for (let i = 0; i < count; i++) {
            // Generate an 8-character hex string (4 bytes = 8 hex chars)
            codes.push(crypto.randomBytes(4).toString('hex'));
        }
        return codes;
    }

    async enable2FA(userId: string, secret: string, code: string) {
        const result = verify({ token: code, secret });
        if (!result) {
            throw new Error("Invalid verification code");
        }
        await this.securityRepo.update2FA(userId, secret, true);
        
        // Generate and store new backup codes
        const plaintextCodes = this.generateBackupCodes(10);
        const hashedCodes = await Promise.all(
            plaintextCodes.map(c => scrypt.hash(c))
        );
        await this.securityRepo.saveBackupCodes(userId, hashedCodes);
        
        return { backupCodes: plaintextCodes };
    }

    async regenerateBackupCodes(userId: string) {
        const settings = await this.getSettings(userId);
        if (!settings || !settings.two_factor_enabled) {
            throw new Error("2FA is not enabled for this user");
        }
        
        const plaintextCodes = this.generateBackupCodes(10);
        const hashedCodes = await Promise.all(
            plaintextCodes.map(c => scrypt.hash(c))
        );
        await this.securityRepo.saveBackupCodes(userId, hashedCodes);
        
        return { backupCodes: plaintextCodes };
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
