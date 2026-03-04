import { SecurityRepository } from '../repositories/SecurityRepository';
import { MailService } from './MailService';
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
        await this.securityRepo.updateBackupCodes(userId, hashedCodes);
        
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
        await this.securityRepo.updateBackupCodes(userId, hashedCodes);
        
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
        // Verify device belongs to user before revoking
        const devices = await this.securityRepo.getDevices(undefined, deviceId);
        if (!devices.length || devices[0].user_id !== userId) {
            throw new Error('Device not found or access denied');
        }
        await this.securityRepo.revokeDevice(deviceId);
    }

    // ===================== Device Management =====================

    async trustDevice(userId: string, deviceId: string) {
        const devices = await this.securityRepo.getDevices(undefined, deviceId);
        if (!devices.length || devices[0].user_id !== userId) {
            throw new Error('Device not found or access denied');
        }
        await this.securityRepo.trustDevice(deviceId);
    }

    async untrustDevice(userId: string, deviceId: string) {
        const devices = await this.securityRepo.getDevices(undefined, deviceId);
        if (!devices.length || devices[0].user_id !== userId) {
            throw new Error('Device not found or access denied');
        }
        await this.securityRepo.untrustDevice(deviceId);
    }

    async revokeDevice(userId: string, deviceId: string) {
        const devices = await this.securityRepo.getDevices(undefined, deviceId);
        if (!devices.length || devices[0].user_id !== userId) {
            throw new Error('Device not found or access denied');
        }
        await this.securityRepo.revokeDevice(deviceId);
        
        // Also invalidate all sessions for this device
        // This would require linking sessions to device_id properly
    }

    /**
     * Check if a device is recognized (known) for this user
     * Uses device fingerprint to identify the device
     */
    async isDeviceRecognized(userId: string, deviceFingerprint: string): Promise<boolean> {
        const device = await this.securityRepo.getDeviceByFingerprint(userId, deviceFingerprint);
        return !!device;
    }

    /**
     * Register or update a device for the user
     * Called on each login to track and update device information
     */
    async registerOrUpdateDevice(
        userId: string,
        deviceInfo: {
            fingerprint: string;
            userAgent: string;
            ipAddress: string;
        }
    ): Promise<{ device: any; isNewDevice: boolean }> {
        // Check if device exists
        const existingDevice = await this.securityRepo.getDeviceByFingerprint(userId, deviceInfo.fingerprint);
        
        if (existingDevice) {
            // Update existing device
            await this.securityRepo.updateDevice(existingDevice.id, userId, {
                lastActiveAt: new Date(),
                user_agent: deviceInfo.userAgent,
                fingerprint: deviceInfo.fingerprint
            });
            return { device: existingDevice, isNewDevice: false };
        } else {
            // Create new device
            const newDevice = await this.securityRepo.createDevice(userId, {
                user_agent: deviceInfo.userAgent,
                fingerprint: deviceInfo.fingerprint,
                ip_address: deviceInfo.ipAddress,
                lastActiveAt: new Date()
            });
            return { device: newDevice, isNewDevice: true };
        }
    }

    /**
     * Send new device login notification
     */
    async notifyNewDeviceLogin(
        userId: string,
        deviceId: string,
        deviceInfo: {
            deviceName: string;
            ipAddress: string;
            location?: string;
        }
    ) {
        // Check if user wants notifications
        const shouldNotify = await this.securityRepo.shouldNotifyNewDevice(userId);
        if (!shouldNotify) return;

        // Get user email
        const userRes = await pool.query('SELECT email FROM users WHERE id = $1', [userId]);
        const email = userRes.rows[0]?.email;
        if (!email) return;

        // Create notification in database
        await this.securityRepo.createDeviceNotification(
            userId,
            deviceId,
            'new_device',
            `New device login: ${deviceInfo.deviceName} from ${deviceInfo.ipAddress}`,
            deviceInfo.ipAddress,
            deviceInfo.location
        );

        // Send email notification
        await MailService.sendNewDeviceLoginNotification(email, {
            deviceName: deviceInfo.deviceName,
            ipAddress: deviceInfo.ipAddress,
            location: deviceInfo.location,
            loginTime: new Date()
        });
    }

    /**
     * Get device notifications for user
     */
    async getDeviceNotifications(userId: string, includeRead = false) {
        return await this.securityRepo.getDeviceNotifications(userId, includeRead);
    }

    /**
     * Mark notification as read
     */
    async acknowledgeNotification(userId: string, notificationId: string) {
        await this.securityRepo.markNotificationAsRead(notificationId);
    }
}
