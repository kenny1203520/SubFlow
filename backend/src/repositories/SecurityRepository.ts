import { BaseRepository } from './BaseRepository';

export class SecurityRepository extends BaseRepository {
    async getSecuritySettings(userId: string) {
        const res = await this.query('SELECT * FROM user_security WHERE user_id = $1', [userId]);
        return res.rows[0];
    }

    async update2FA(userId: string, secret: string, enabled: boolean) {
        await this.query(`
            INSERT INTO user_security (user_id, two_factor_secret, two_factor_enabled)
            VALUES ($1, $2, $3)
            ON CONFLICT (user_id) DO UPDATE 
            SET two_factor_secret = $2, two_factor_enabled = $3
        `, [userId, secret, enabled]);
    }

    async logLogin(data: any) {
        await this.query(`
            INSERT INTO login_history (user_id, session_id, ip_address, user_agent, device_fingerprint, status, failure_reason)
            VALUES ($1, $2, $3, $4, $5, $6, $7)
        `, [data.user_id, data.session_id, data.ip, data.user_agent, data.fingerprint, data.status, data.reason]);
    }

    async getDevices(userId: string) {
        const res = await this.query('SELECT * FROM user_devices WHERE user_id = $1', [userId]);
        return res.rows;
    }

    async updateDevice(userId: string, data: { name: string, fingerprint: string, ip: string }) {
        const existing = await this.query('SELECT id FROM user_devices WHERE user_id = $1 AND device_fingerprint = $2', [userId, data.fingerprint]);
        if (existing.rows.length > 0) {
            await this.query('UPDATE user_devices SET last_active_at = NOW(), device_name = $2 WHERE id = $1', [existing.rows[0].id, data.name]);
        } else {
            await this.query(`
                INSERT INTO user_devices (user_id, device_name, device_fingerprint, last_active_at, is_trusted)
                VALUES ($1, $2, $3, NOW(), false)
            `, [userId, data.name, data.fingerprint]);
        }
    }

    async revokeDevice(userId: string, deviceId: string) {
        await this.query('DELETE FROM user_devices WHERE id = $1 AND user_id = $2', [deviceId, userId]);
    }
}
