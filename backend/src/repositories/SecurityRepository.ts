import { BaseRepository } from './BaseRepository';

export interface UserSecurityRow {
    user_id: string;
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
    updated_at?: Date;
}

export interface UserDeviceRow {
    id: string;
    user_id: string;
    device_name?: string;
    device_fingerprint?: string;
    last_active_at?: Date;
    is_trusted?: boolean;
    is_blocked?: boolean;
    created_at?: Date;
    updated_at?: Date;
}

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
     * Update user's 2FA settings.
     * @param userId 
     * @param secret 
     * @param enabled 
     */
    async update2FA(userId: string, secret: string, enabled: boolean) {
        await this.query(`
            INSERT INTO user_security (user_id, two_factor_secret, two_factor_enabled)
            VALUES ($1, $2, $3)
            ON CONFLICT (user_id) DO UPDATE 
            SET two_factor_secret = $2, two_factor_enabled = $3
        `, [userId, secret, enabled]);
    }

    /**
     * Update user's backup codes.
     * @param userId 
     * @param hashedCodes 
     */
    async updateBackupCodes(userId: string, hashedCodes: string[]) {
        await this.query(`
            UPDATE user_security 
            SET backup_codes = $2
            WHERE user_id = $1
        `, [userId, hashedCodes]);
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
            fingerprint?: string,
            lastActiveAt?: Date,
            isTrusted?: boolean,
            isBlocked?: boolean
        }
    ): Promise<UserDeviceRow> {
        if (!userId) {
            throw new Error("User ID is required to create a device record");
        }
        const res = await this.query(`
            INSERT INTO user_devices (user_id, device_name, device_fingerprint, last_active_at, is_trusted, is_blocked)
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING *
        `, [userId, data?.name || null, data?.fingerprint || null, data?.lastActiveAt || null, data?.isTrusted || false, data?.isBlocked || false]);
        return res.rows[0];
    }

    /**
     * Get all devices associated with the user (for active sessions and remembered devices)
     * @param userId 
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
            fingerprint?: string,
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
            SET user_id = COALESCE($1, user_id),
                device_name = COALESCE($2, device_name),
                device_fingerprint = COALESCE($3, device_fingerprint),
                last_active_at = COALESCE($4, last_active_at),
                is_trusted = COALESCE($5, is_trusted),
                is_blocked = COALESCE($6, is_blocked)
            WHERE id = $1
            RETURNING *
        `, [
            deviceId, userId || null, data?.name || null,
            data?.fingerprint || null, data?.lastActiveAt || null,
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
}
