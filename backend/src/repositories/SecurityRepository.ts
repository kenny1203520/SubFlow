import { BaseRepository } from './BaseRepository';

export class SecurityRepository extends BaseRepository {
    async getSecuritySettings(userId: string) {
        const res = await this.query('SELECT * FROM user_security WHERE user_id = $1', [userId]);
        return res.rows[0];
    }

    async update2FA(userId: string, secret: string, enabled: boolean) {
        await this.query(`
            INSERT INTO user_security (user_id, totp_secret, two_factor_enabled)
            VALUES ($1, $2, $3)
            ON CONFLICT (user_id) DO UPDATE 
            SET totp_secret = $2, two_factor_enabled = $3
        `, [userId, secret, enabled]);
    }

    async logLogin(data: any) {
        await this.query(`
            INSERT INTO login_history (user_id, ip_address, device_fingerprint, status, failure_reason)
            VALUES ($1, $2, $3, $4, $5)
        `, [data.user_id, data.ip, data.fingerprint, data.status, data.reason]);
    }

    async getDevices(userId: string) {
        const res = await this.query('SELECT * FROM user_devices WHERE user_id = $1', [userId]);
        return res.rows;
    }

    async revokeDevice(userId: string, deviceId: string) {
        await this.query('DELETE FROM user_devices WHERE id = $1 AND user_id = $2', [deviceId, userId]);
    }
}
