import { BaseRepository } from './BaseRepository';
import crypto from 'crypto';

export interface UserSecurityRow {
    user_id: string;
    two_factor_enabled?: boolean;
    two_factor_secret?: string;
    backup_codes?: string[];
    passkey_enabled: boolean;
    is_suspended?: boolean;
    suspended_at?: Date;
    suspended_until?: Date;
    suspension_reason?: string;
    is_blocked?: boolean;
    blocked_at?: Date;
    block_reason?: string;
    failed_login_attempts?: number;
    auth_provider?: string;
    provider_id?: string;
    ldap_enabled: boolean;
    sso_enabled: boolean;
    notify_new_device: boolean;
    notify_new_location: boolean;
    notify_suspicious_activity: boolean;
    last_password_change?: Date;
    updated_at?: Date;
}

export interface UserDeviceRow {
    id: string;
    user_id: string;
    device_name?: string;
    user_agent?: string;
    ip_address?: string;
    device_fingerprint?: string;
    device_token?: string;
    last_active_at?: Date;
    is_trusted?: boolean;
    trusted_at?: Date;
    is_blocked?: boolean;
    blocked_at?: Date;
    created_at?: Date;
    updated_at?: Date;
}

export const formatUserAgent = (uaString: string): string => {
    if (!uaString) return 'Unknown Device';

    let browser = 'Unknown Browser';
    if (uaString.indexOf("Firefox") > -1) browser = "Firefox";
    else if (uaString.indexOf("SamsungBrowser") > -1) browser = "Samsung Internet";
    else if (uaString.indexOf("Opera") > -1 || uaString.indexOf("OPR") > -1) browser = "Opera";
    else if (uaString.indexOf("Trident") > -1) browser = "Internet Explorer";
    else if (uaString.indexOf("Edge") > -1) browser = "Edge";
    else if (uaString.indexOf("Chrome") > -1) browser = "Chrome";
    else if (uaString.indexOf("Safari") > -1) browser = "Safari";

    let os = 'Unknown OS';
    if (uaString.indexOf("Win") > -1) os = "Windows";
    else if (uaString.indexOf("Mac") > -1) os = "MacOS";
    else if (uaString.indexOf("Linux") > -1) os = "Linux";
    else if (uaString.indexOf("Android") > -1) os = "Android";
    else if (uaString.indexOf("like Mac") > -1) os = "iOS";

    return `${browser} on ${os}`;
};

export class SecurityRepository extends BaseRepository {

    /**
     * Create security settings for a user.
     * @param userId 
     * @param data 
     * @returns 
     */
    async createSecuritySettings(
        userId: string,
        data?: {
            two_factor_enabled?: boolean;
            two_factor_secret?: string;
            backup_codes?: string[];
            is_suspended?: boolean;
            suspended_at?: Date;
            suspended_until?: Date;
            suspension_reason?: string;
            is_blocked?: boolean;
            blocked_at?: Date;
            block_reason?: string;
            failed_login_attempts?: number;
            auth_provider?: string;
            provider_id?: string;
            last_password_change?: Date;   
        }
    ): Promise<UserSecurityRow> {
        const res = await this.query(`
            INSERT INTO user_security (
                user_id,
                two_factor_enabled,
                two_factor_secret,
                backup_codes,
                is_suspended,
                suspended_at,
                suspended_until,
                suspension_reason,
                is_blocked,
                blocked_at,
                block_reason,
                failed_login_attempts,
                auth_provider,
                provider_id,
                last_password_change
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
            RETURNING *
        `, [
            userId, data?.two_factor_enabled || false,
            data?.two_factor_secret || null, data?.backup_codes || [],
            data?.is_suspended || false, data?.suspended_at || null,
            data?.suspended_until || null, data?.suspension_reason || null,
            data?.is_blocked || false, data?.blocked_at || null,
            data?.block_reason || null, data?.failed_login_attempts || 0,
            data?.auth_provider || null, data?.provider_id || null,
            data?.last_password_change || null
        ]);
        return res.rows[0];
    }

    /**
     * Get user's security settings.
     * User;s security settings including 2FA status, backup codes, and account lock status
     * @param userId 
     * @returns 
     */
    async getSecuritySettings(userId: string): Promise<UserSecurityRow | null> {
        const res = await this.query(`
            SELECT *
            FROM user_security
            WHERE user_id = $1
            `, [userId]);
        return res.rows[0] || null;
    }

    /**
     * Check if 2FA is enabled for the user.
     * @param userId 
     * @returns 
     */
    async is2FAEnabled(userId: string): Promise<boolean> {
        const res = await this.query(`
            SELECT two_factor_enabled
            FROM user_security
            WHERE user_id = $1
        `, [userId]);
        return res.rows[0]?.two_factor_enabled || false;
    }

    /**
     * Update user's 2FA settings.
     * @param userId 
     * @param secret 
     * @param enabled 
     */
    async update2FA(userId: string, secret: string, enabled: boolean): Promise<void> {
        await this.query(`
            INSERT INTO user_security (user_id, two_factor_secret, two_factor_enabled)
            VALUES ($1, $2, $3)
            ON CONFLICT (user_id) DO UPDATE 
            SET two_factor_secret = $2, two_factor_enabled = $3
        `, [userId, secret, enabled]);
    }

    async verify2FASecret(userId: string, secret: string): Promise<boolean> {
        const res = await this.query(`
            SELECT two_factor_secret
            FROM user_security
            WHERE user_id = $1
        `, [userId]);
        const storedSecret = res.rows[0]?.two_factor_secret;
        return storedSecret === secret;
    }

    /**
     * Update user's backup codes.
     * @param userId 
     * @param hashedCodes 
     */
    async updateBackupCodes(userId: string, hashedCodes: string[]): Promise<void> {
        await this.query(`
            UPDATE user_security 
            SET backup_codes = $2
            WHERE user_id = $1
        `, [userId, hashedCodes]);
    }

    /**
     * Verify a backup code for the user.
     * @param userId 
     * @param code 
     * @returns 
     */
    async verifyBackupCode(userId: string, code: string): Promise<boolean> {
        const res = await this.query(`
            SELECT backup_codes
            FROM user_security
            WHERE user_id = $1
        `, [userId]);
        const backupCodes: string[] = res.rows[0]?.backup_codes || [];
        return backupCodes.includes(code);
    }

    /**
     * Disable 2FA for the user.
     * @param userId 
     */
    async disable2FA(userId: string): Promise<void> {
        await this.query(`
            UPDATE user_security
            SET two_factor_enabled = false, two_factor_secret = null, backup_codes = null
            WHERE user_id = $1
        `, [userId]);
    }

    /**
     * Check if the user is currently suspended.
     * If the suspension has expired, it will automatically unsuspend the user.
     * @param userId 
     * @returns 
     */
    async isUserSuspended(userId: string): Promise<boolean> {
        const res = await this.query(`
            SELECT is_suspended, suspended_until
            FROM user_security
            WHERE user_id = $1
        `, [userId]);
        const isSuspended = res.rows[0]?.is_suspended || false;
        const suspendedUntil = res.rows[0]?.suspended_until;
        if (isSuspended && suspendedUntil && new Date() > new Date(suspendedUntil)) {
            // Suspension expired, auto-unsuspend
            await this.unSuspendUser(userId);
            return false;
        }
        return isSuspended;
    }

    /**
     * Suspend a user until a certain date with an optional reason.
     * @param userId 
     * @param until 
     * @param reason 
     */
    async suspendUser(userId: string, until: Date, reason?: string): Promise<void> {
        await this.query(`
            UPDATE user_security
            SET is_suspended = true, suspended_until = $2, suspension_reason = $3
            WHERE user_id = $1
        `, [userId, until, reason || null]);
    }

    /**
     * Unsuspend a user.
     * @param userId 
     */
    async unSuspendUser(userId: string): Promise<void> {
        await this.query(`
            UPDATE user_security
            SET is_suspended = false, suspended_until = null, suspension_reason = null
            WHERE user_id = $1
        `, [userId]);
    }

    async isUserBlocked(userId: string): Promise<boolean> {
        const res = await this.query(`
            SELECT is_blocked
            FROM user_security
            WHERE user_id = $1
        `, [userId]);
        return res.rows[0]?.is_blocked || false;
    }

    /**
     * Block a user with an optional reason.
     * @param userId 
     * @param reason 
     */
    async blockUser(userId: string, reason?: string): Promise<void> {
        await this.query(`
            UPDATE user_security
            SET is_blocked = true, block_reason = $2, blocked_at = NOW()
            WHERE user_id = $1
        `, [userId, reason || null]);
    }

    /**
     * Unblock a user.
     * @param userId 
     */
    async unBlockUser(userId: string): Promise<void> {
        await this.query(`
            UPDATE user_security
            SET is_blocked = false, block_reason = null, blocked_at = null
            WHERE user_id = $1
        `, [userId]);
    }

    /**
     * Get the number of failed login attempts for a user.
     * @param userId 
     * @returns 
     */
    async getFailedLoginAttempts(userId: string): Promise<number> {
        const res = await this.query(`
            SELECT failed_login_attempts
            FROM user_security
            WHERE user_id = $1
        `, [userId]);
        return res.rows[0]?.failed_login_attempts || 0;
    }

    /**
     * Increment failed login attempts for a user.
     * This can be used for implementing account lockout after a certain number of failed attempts.
     * @param userId 
     */
    async incrementFailedLoginAttempts(userId: string): Promise<void> {
        await this.query(`
            UPDATE user_security
            SET failed_login_attempts = COALESCE(failed_login_attempts, 0) + 1
            WHERE user_id = $1
        `, [userId]);
    }

    /**
     * Reset failed login attempts for a user.
     * @param userId 
     */
    async resetFailedLoginAttempts(userId: string): Promise<void> {
        await this.query(`
            UPDATE user_security
            SET failed_login_attempts = 0
            WHERE user_id = $1
        `, [userId]);
    }

    /**
     * Get the authentication provider information for a user.
     * @param userId 
     * @returns 
     */
    async getAuthProvider(userId: string): Promise<{ auth_provider: string, provider_id: string } | null> {
        const res = await this.query(`
            SELECT auth_provider, provider_id
            FROM user_security
            WHERE user_id = $1
        `, [userId]);
        return res.rows[0] || null;
    }

    /**
     * Update the authentication provider information for a user.
     * @param userId 
     * @param authProvider 
     * @param providerId 
     */
    async updateAuthProvider(userId: string, authProvider: string, providerId: string): Promise<void> {
        await this.query(`
            UPDATE user_security
            SET auth_provider = $2, provider_id = $3
            WHERE user_id = $1
        `, [userId, authProvider, providerId]);
    }

    /**
     * Get the last password change time for a user.
     * @param userId 
     * @returns 
     */
    async getPasswordChangeTime(userId: string): Promise<Date | null> {
        const res = await this.query(`
            SELECT last_password_change
            FROM user_security
            WHERE user_id = $1
        `, [userId]);
        return res.rows[0]?.last_password_change || null;
    }
    
    /**
     * Update the last password change time for a user.
     * @param userId 
     * @param changeTime 
     */
    async updatePasswordChangeTime(userId: string, changeTime: Date): Promise<void> {
        await this.query(`
            UPDATE user_security
            SET last_password_change = $2
            WHERE user_id = $1
        `, [userId, changeTime]);
    }

    /**
     * Create a new device record for the user.
     * This can be used for tracking active sessions and remembered devices.
     * @param userId 
     * @param data 
     * @returns 
     */
    async createDevice(
        userId: string,
        data?: {
            name?: string,
            user_agent?: string,
            ip_address?: string,
            fingerprint?: string,
            token?: string,
            lastActiveAt?: Date,
            isTrusted?: boolean,
            isBlocked?: boolean
        }
    ): Promise<UserDeviceRow> {
        if (!userId) {
            throw new Error("User ID is required to create a device record");
        }
        
        // Generate device token if not provided
        const deviceToken = data?.token || crypto.randomBytes(32).toString('hex');
        
        const res = await this.query(`
            INSERT INTO user_devices (
                user_id, device_name, user_agent, ip_address,
                device_fingerprint, device_token, last_active_at, 
                is_trusted, trusted_at, is_blocked
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
            RETURNING *
        `, [
            userId, 
            data?.name || formatUserAgent(data?.user_agent || ''),
            data?.user_agent || null, 
            data?.ip_address || null,
            data?.fingerprint || null,
            deviceToken,
            data?.lastActiveAt || new Date(),
            data?.isTrusted || false,
            data?.isTrusted ? new Date() : null,
            data?.isBlocked || false
        ]);
        return res.rows[0];
    }

    /**
     * Get device by device token (for cookie-based device authentication)
     * @param deviceToken 
     * @returns 
     */
    async getDeviceByToken(deviceToken: string): Promise<UserDeviceRow | null> {
        const res = await this.query(`
            SELECT *
            FROM user_devices
            WHERE device_token = $1 AND is_blocked = false
        `, [deviceToken]);
        return res.rows[0] || null;
    }

    /**
     * Get device by fingerprint for a specific user
     * @param userId 
     * @param fingerprint 
     * @returns 
     */
    async getDeviceByFingerprint(userId: string, fingerprint: string): Promise<UserDeviceRow | null> {
        const res = await this.query(`
            SELECT *
            FROM user_devices
            WHERE user_id = $1 AND device_fingerprint = $2
            ORDER BY last_active_at DESC
            LIMIT 1
        `, [userId, fingerprint]);
        return res.rows[0] || null;
    }

    /**
     * Get all devices associated with the user (for active sessions and remembered devices)
     * userId and deviceId are mutually exclusive parameters. Provide one to filter by user or device.
     * @param userId 
     * @param deviceId
     * @returns 
     */
    async getDevices(userId?: string, deviceId?: string): Promise<UserDeviceRow[]> {
        if (!userId && !deviceId) {
            throw new Error("Either userId or deviceId must be provided");
        }
        if (userId && deviceId) {
            throw new Error("Only one of userId or deviceId should be provided");
        }
        if (deviceId) {
            const res = await this.query(`
                SELECT *
                FROM user_devices
                WHERE id = $1
                `, [deviceId]);
            return res.rows;
        }
        const res = await this.query(`
            SELECT *
            FROM user_devices
            WHERE user_id = $1
            `, [userId]);
        return res.rows;
    }

    /**
     * Update device information.
     * This can be used for updating last active time, marking a device as trusted, or blocking a device.
     * @param deviceId 
     * @param userId 
     * @param data 
     * @returns 
     */
    async updateDevice(
        deviceId: string,
        userId?: string,
        data?: {
            name?: string,
            user_agent?: string,
            ip_address?: string,
            fingerprint?: string,
            token?: string,
            lastActiveAt?: Date,
            isTrusted?: boolean,
            isBlocked?: boolean
        }
    ): Promise<UserDeviceRow> {
        if (!userId && !deviceId) {
            throw new Error("Either userId or deviceId must be provided");
        }
        const res = await this.query(`
            UPDATE user_devices
            SET 
                device_name = COALESCE($3, device_name),
                user_agent = COALESCE($4, user_agent),
                ip_address = COALESCE($5, ip_address),
                device_fingerprint = COALESCE($6, device_fingerprint),
                device_token = COALESCE($7, device_token),
                last_active_at = COALESCE($8, last_active_at),
                is_trusted = COALESCE($9, is_trusted),
                trusted_at = CASE WHEN $9 = true THEN COALESCE(trusted_at, CURRENT_TIMESTAMP) ELSE trusted_at END,
                is_blocked = COALESCE($10, is_blocked),
                blocked_at = CASE WHEN $10 = true THEN COALESCE(blocked_at, CURRENT_TIMESTAMP) ELSE blocked_at END
            WHERE id = $1 AND (user_id = $2 OR $2 IS NULL)
            RETURNING *
        `, [
            deviceId, userId || null, data?.name || null,
            data?.user_agent || null, data?.ip_address || null,
            data?.fingerprint || null, data?.token || null,
            data?.lastActiveAt || null,
            data?.isTrusted || false, data?.isBlocked || false
        ]);
        return res.rows[0];
    }

    /**
     * Revoke a device.
     * Device can be used for logging out of sessions or blocking compromised devices.
     * @param deviceId 
     */
    async revokeDevice(deviceId: string) {
        await this.query(`
            UPDATE user_devices
            SET is_blocked = true
            WHERE id = $1
        `, [deviceId]);
    }

    /**
     * Trust a device (useful for "remember this device" functionality)
     * @param deviceId 
     */
    async trustDevice(deviceId: string): Promise<void> {
        await this.query(`
            UPDATE user_devices
            SET is_trusted = true, trusted_at = CURRENT_TIMESTAMP
            WHERE id = $1
        `, [deviceId]);
    }

    /**
     * Untrust a device
     * @param deviceId 
     */
    async untrustDevice(deviceId: string): Promise<void> {
        await this.query(`
            UPDATE user_devices
            SET is_trusted = false, trusted_at = null
            WHERE id = $1
        `, [deviceId]);
    }

    /**
     * Block a device (useful for blocking suspicious devices or sessions)
     * @param deviceId 
     */
    async blockDevice(deviceId: string): Promise<void> {
        await this.query(`
            UPDATE user_devices
            SET is_blocked = true, blocked_at = CURRENT_TIMESTAMP
            WHERE id = $1
        `, [deviceId]);
    }

    /**
     * Unblock a device
     * @param deviceId 
     */
    async unblockDevice(deviceId: string): Promise<void> {
        await this.query(`
            UPDATE user_devices
            SET is_blocked = false, blocked_at = null
            WHERE id = $1
        `, [deviceId]);
    }

    // ===================== Device Notifications =====================

    /**
     * Create a device notification
     * @param userId 
     * @param deviceId 
     * @param type 
     * @param message 
     * @param ipAddress 
     * @param location 
     */
    async createDeviceNotification(
        userId: string,
        deviceId: string | null,
        type: 'new_device' | 'new_location' | 'suspicious_activity',
        message: string,
        ipAddress?: string,
        location?: string
    ): Promise<any> {
        const res = await this.query(`
            INSERT INTO device_notifications (
                user_id, device_id, notification_type, message,
                ip_address, location
            )
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING *
        `, [userId, deviceId, type, message, ipAddress || null, location || null]);
        return res.rows[0];
    }

    /**
     * Get device notifications for a user
     * @param userId 
     * @param includeRead 
     */
    async getDeviceNotifications(userId: string, includeRead = false): Promise<any[]> {
        const query = includeRead
            ? `SELECT * FROM device_notifications WHERE user_id = $1 ORDER BY created_at DESC`
            : `SELECT * FROM device_notifications WHERE user_id = $1 AND is_read = false ORDER BY created_at DESC`;
        
        const res = await this.query(query, [userId]);
        return res.rows;
    }

    /**
     * Mark notification as read
     * @param notificationId 
     */
    async markNotificationAsRead(notificationId: string): Promise<void> {
        await this.query(`
            UPDATE device_notifications
            SET is_read = true, acknowledged_at = CURRENT_TIMESTAMP
            WHERE id = $1
        `, [notificationId]);
    }

    /**
     * Check if user should be notified about new device
     * @param userId 
     */
    async shouldNotifyNewDevice(userId: string): Promise<boolean> {
        const res = await this.query(`
            SELECT notify_new_device
            FROM user_security
            WHERE user_id = $1
        `, [userId]);
        return res.rows[0]?.notify_new_device ?? true;
    }

    /**
     * Check if user should be notified about new location
     * @param userId 
     */
    async shouldNotifyNewLocation(userId: string): Promise<boolean> {
        const res = await this.query(`
            SELECT notify_new_location
            FROM user_security
            WHERE user_id = $1
        `, [userId]);
        return res.rows[0]?.notify_new_location ?? true;
    }

    /**
     * Check if user should be notified about suspicious activity
     * @param userId 
     */
    async shouldNotifySuspiciousActivity(userId: string): Promise<boolean> {
        const res = await this.query(`
            SELECT notify_suspicious_activity
            FROM user_security
            WHERE user_id = $1
        `, [userId]);
        return res.rows[0]?.notify_suspicious_activity ?? true;
    }
}

