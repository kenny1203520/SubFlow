import { BaseRepository } from "./BaseRepository";

export interface SessionRow {
    id: string;
    user_id: string;
    ip_address?: string;
    user_agent?: string;
    device_fingerprint?: string;
    device_id?: string;
    expires_at: Date;
    created_at?: Date;
    updated_at?: Date;
}

export interface LoginHistoryRow {
    id: string;
    user_id?: string;
    session_id?: string;
    login_at?: Date;
    ip_address?: string;
    user_agent?: string;
    device_fingerprint?: string;
    device_id?: string;
    status?: string;
    failure_reason?: string;
}

export interface IpBlocksRow {
    ip_address: string;
    reason?: string;
    expires_at?: Date;
    created_at?: Date;
    created_by?: string;
    updated_at?: Date;
    updated_by?: string;
}

export interface EmailVerificationTokensRow {
    id: string;
    user_id: string;
    email: string;
    token_hash: string;
    expires_at: Date;
    created_at?: Date;
}

export interface PasswordResetTokensRow {
    id: string;
    user_id: string;
    token_hash: string;
    expires_at: Date;
    is_used?: boolean;
    created_at?: Date;
}

export class AuthRepository extends BaseRepository {

    /**
     * Create a new session for a user.
     * @param userId 
     * @param ipAddress 
     * @param userAgent 
     * @param deviceFingerprint 
     * @param deviceId
     * @param expiresAt 
     * @returns 
     */
    async createSession(userId: string, ipAddress: string, userAgent: string, deviceFingerprint: string, deviceId: string, expiresAt: Date): Promise<SessionRow> {
        const res = await this.query(`
            INSERT INTO sessions (user_id, ip_address, user_agent, device_fingerprint, device_id, expires_at)
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING *
        `, [userId, ipAddress, userAgent, deviceFingerprint, deviceId, expiresAt]);
        return res.rows[0];
    }

    /**
     * Get session by its ID.
     * @param sessionId 
     * @returns 
     */
    async getSessionById(sessionId: string): Promise<SessionRow | null> {
        const res = await this.query(`
            SELECT *
            FROM sessions
            WHERE id = $1
        `, [sessionId]);
        return res.rows[0] || null;
    }

    /**
     * Get all active sessions for a user.
     * @param userId 
     * @returns 
     */
    async getSessionsByUserId(userId: string): Promise<SessionRow[]> {
        const res = await this.query(`
            SELECT *
            FROM sessions
            WHERE user_id = $1
        `, [userId]);
        return res.rows;
    }

    /**
     * Get session(s) by device ID (useful for tracking sessions from a specific device).
     * @param deviceId 
     * @returns 
     */
    async getSessionByDeviceId(deviceId: string): Promise<SessionRow[]> {
        const res = await this.query(`
            SELECT *
            FROM sessions
            WHERE device_id = $1
        `, [deviceId]);
        return res.rows;
    }

    /**
     * Delete a session by its ID (used for logout and session invalidation).
     * @param sessionId 
     */
    async deleteSession(sessionId: string): Promise<void> {
        await this.query(`
            DELETE FROM sessions
            WHERE id = $1
        `, [sessionId]);
    }

    /**
     * Login history entry for a user login attempt.
     * @param user_id 
     * @param session_id 
     * @param login_at 
     * @param ip_address 
     * @param user_agent 
     * @param device_fingerprint 
     * @param device_id
     * @param status 
     * @param failure_reason 
     */
    async createLoginHistory(
        user_id: string,
        session_id: string,
        login_at: Date,
        ip_address: string,
        user_agent: string,
        device_fingerprint: string,
        device_id: string,
        status: string,
        failure_reason?: string
    ) {
        await this.query(`
            INSERT INTO login_history (
                user_id, session_id, login_at, ip_address,
                user_agent, device_fingerprint, device_id, status,
                failure_reason
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        `, [
            user_id, session_id, login_at, ip_address,
            user_agent, device_fingerprint, device_id, status,
            failure_reason
        ]);
    }

    /**
     * Get login history entry by its ID.
     * @param id 
     * @returns 
     */
    async getLoginHistoryById(id: string): Promise<LoginHistoryRow | null> {
        const res = await this.query(`
            SELECT *
            FROM login_history
            WHERE id = $1
        `, [id]);
        return res.rows[0] || null;
    }

    /**
     * Get login history for a user.
     * @param userId 
     * @returns 
     */
    async getLoginHistoryByUserId(userId: string): Promise<LoginHistoryRow[]> {
        const res = await this.query(`
            SELECT *
            FROM login_history
            WHERE user_id = $1
            ORDER BY login_at DESC
        `, [userId]);
        return res.rows;
    }

    /**
     * Get login history for a session.
     * @param sessionId 
     * @returns 
     */
    async getLoginHistoryBySessionId(sessionId: string): Promise<LoginHistoryRow[]> {
        const res = await this.query(`
            SELECT *
            FROM login_history
            WHERE session_id = $1
            ORDER BY login_at DESC
        `, [sessionId]);
        return res.rows;
    }

    /**
     * Get login history for a specific IP address.
     * (useful for detecting suspicious activity).
     * @param ipAddress 
     * @returns 
     */
    async getLoginHistoryByIp(ipAddress: string): Promise<LoginHistoryRow[]> {
        const res = await this.query(`
            SELECT *
            FROM login_history
            WHERE ip_address = $1
            ORDER BY login_at DESC
        `, [ipAddress]);
        return res.rows;
    }

    /**
     * Get login history for a specific device ID.
     * (useful for tracking logins from a specific device).
     * @param deviceId 
     * @returns 
     */
    async getLoginHistoryByDeviceId(deviceId: string): Promise<LoginHistoryRow[]> {
        const res = await this.query(`
            SELECT *
            FROM login_history
            WHERE device_id = $1
            ORDER BY login_at DESC
        `, [deviceId]);
        return res.rows;
    }

    /**
     * Block an IP address.
     * (used for preventing access from malicious IPs).
     * @param ipAddress 
     * @param reason 
     * @param expiresAt 
     * @param createdBy 
     */
    async blockIp(ipAddress: string, reason?: string, expiresAt?: Date, createdBy?: string): Promise<void> {
        await this.query(`
            INSERT INTO ip_blocks (ip_address, reason, expires_at, created_by)
            VALUES ($1, $2, $3, $4)
        `, [ipAddress, reason || null, expiresAt || null, createdBy || null]);
    }

    /**
     * Check if an IP address is currently blocked.
     * @param ipAddress 
     * @returns 
     */
    async isIpBlocked(ipAddress: string): Promise<boolean> {
        const res = await this.query(`
            SELECT *
            FROM ip_blocks
            WHERE ip_address = $1 AND (expires_at IS NULL OR expires_at > NOW())
        `, [ipAddress]);
        return res.rows.length > 0;
    }

    /**
     * Update block details for an IP address (e.g. extend block duration, update reason).
     * @param ipAddress 
     * @param reason 
     * @param expiresAt 
     * @param updatedBy 
     */
    async updateBlockedIp(ipAddress: string, reason?: string, expiresAt?: Date, updatedBy?: string): Promise<void> {
        await this.query(`
            UPDATE ip_blocks
            SET reason = COALESCE($2, reason),
                expires_at = COALESCE($3, expires_at),
                updated_by = $4
            WHERE ip_address = $1
        `, [ipAddress, reason || null, expiresAt || null, updatedBy || null]);
    }
    
    /**
     * Unblock an IP address.
     * @param ipAddress 
     * @param updatedBy 
     */
    async unBlockIp(ipAddress: string): Promise<void> {
        await this.query(`
            DELETE FROM ip_blocks
            WHERE ip_address = $1 
        `, [ipAddress]);
    }

    async createEmailVerificationToken(userId: string, email: string, tokenHash: string, expiresAt: Date): Promise<EmailVerificationTokensRow> {
        const res = await this.query(`
            INSERT INTO email_verification_tokens (user_id, email, token_hash, expires_at)
            VALUES ($1, $2, $3, $4)
            RETURNING *
        `, [userId, email, tokenHash, expiresAt]);
        return res.rows[0];
    }

    async getEmailVerificationTokensByUserId(userId: string): Promise<EmailVerificationTokensRow[]> {
        const res = await this.query(`
            SELECT *
            FROM email_verification_tokens
            WHERE user_id = $1 AND expires_at > NOW()
        `, [userId]);
        return res.rows;
    }

    async getEmailVerificationTokenByEmail(userId: string, email: string): Promise<EmailVerificationTokensRow | null> {
        const res = await this.query(`
            SELECT *
            FROM email_verification_tokens
            WHERE user_id = $1 AND email = $2 AND expires_at > NOW()
        `, [userId, email]);
        return res.rows[0] || null;
    }

    async getEmailVerificationTokenByTokenHash(tokenHash: string): Promise<EmailVerificationTokensRow | null> {
        const res = await this.query(`
            SELECT *
            FROM email_verification_tokens
            WHERE token_hash = $1 AND expires_at > NOW()
        `, [tokenHash]);
        return res.rows[0] || null;
    }
    
    async deleteEmailVerificationToken(tokenId: string): Promise<void> {
        await this.query(`
            DELETE FROM email_verification_tokens
            WHERE id = $1
        `, [tokenId]);
    }

    async deleteEmailVerificationTokensByUserId(userId: string): Promise<void> {
        await this.query(`
            DELETE FROM email_verification_tokens
            WHERE user_id = $1
        `, [userId]);
    }

    async createPasswordResetToken(userId: string, tokenHash: string, expiresAt: Date): Promise<PasswordResetTokensRow> {
        const res = await this.query(`
            INSERT INTO password_reset_tokens (user_id, token_hash, expires_at, is_used)
            VALUES ($1, $2, $3, false)
            RETURNING *
        `, [userId, tokenHash, expiresAt]);
        return res.rows[0];
    }

    async getPasswordResetTokensByUserId(userId: string): Promise<PasswordResetTokensRow[]> {
        const res = await this.query(`
            SELECT *
            FROM password_reset_tokens
            WHERE user_id = $1 AND expires_at > NOW() AND is_used = false
        `, [userId]);
        return res.rows;
    }

    async getPasswordResetTokenByTokenHash(tokenHash: string): Promise<PasswordResetTokensRow | null> {
        const res = await this.query(`
            SELECT *
            FROM password_reset_tokens
            WHERE token_hash = $1 AND expires_at > NOW() AND is_used = false
        `, [tokenHash]);
        return res.rows[0] || null;
    }

    async markPasswordResetTokenUsed(tokenId: string): Promise<void> {
        await this.query(`
            UPDATE password_reset_tokens
            SET is_used = true
            WHERE id = $1
        `, [tokenId]);
    }
}